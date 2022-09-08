/*
Copyright 2022 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hooks

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/deckhouse/deckhouse/go_lib/deckhouse-config/conversion"
	d8config_v1 "github.com/deckhouse/deckhouse/go_lib/deckhouse-config/v1"
	"github.com/deckhouse/deckhouse/go_lib/operator"
	"github.com/deckhouse/deckhouse/go_lib/set"
	"github.com/deckhouse/deckhouse/modules/019-deckhouse-config/hooks/internal"
	"github.com/flant/addon-operator/pkg/module_manager"
	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	"github.com/flant/addon-operator/pkg/module_manager/go_hook/metrics"
	"github.com/flant/addon-operator/sdk"
	"github.com/flant/shell-operator/pkg/kube/object_patch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"
)

/*
This hook tracks changes in DeckhouseConfig resources and updates
their statuses.
It uses AddonOperator dependency to get enabled state for all modules
and get access to each module state.

DeckhouseConfig status consists of:
- 'enabled' field - describes a module's enabled state:
    * Enabled
    * Disabled
    * Disabled by script - module is disabled by 'enabled' script
    * Enabled/Disabled by config - module state is determined by DeckhouseConfig
- 'status' field - describes state of the module:
    * Unknown module name - DeckhouseConfig resource name is not a known module name.
    * Running - ModuleRun task is in progress: module starts or reloads.
    * Ready - helm install for module was successful.
    * HookError: ... - problem with module's hook.
    * ModuleError: ... - problem during installing helm chart.
*/

var _ = sdk.RegisterFunc(&go_hook.HookConfig{
	Queue: "/modules/deckhouse-config/status",
	Kubernetes: []go_hook.KubernetesConfig{
		{
			Name:                         "configs",
			ApiVersion:                   "deckhouse.io/v1",
			Kind:                         "DeckhouseConfig",
			FilterFunc:                   filterDeckhouseConfigsForStatus,
			ExecuteHookOnSynchronization: pointer.BoolPtr(true),
			ExecuteHookOnEvents:          pointer.BoolPtr(false),
		},
	},
	Schedule: []go_hook.ScheduleConfig{
		{
			Name:    "update_statuses",
			Crontab: "*/15 * * * * *",
		},
	},
	Settings: &go_hook.HookConfigSettings{
		EnableSchedulesOnStartup: true,
	},
}, updateDeckhouseConfigStatuses)

// filterDeckhouseConfigNames returns names of DeckhouseConfig objects.
func filterDeckhouseConfigsForStatus(unstructured *unstructured.Unstructured) (go_hook.FilterResult, error) {
	var cfg d8config_v1.DeckhouseConfig

	err := sdk.FromUnstructured(unstructured, &cfg)
	if err != nil {
		return nil, err
	}

	// Extract name and enabled into empty DeckhouseConfig.
	return &d8config_v1.DeckhouseConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: cfg.Name,
		},
		Spec: d8config_v1.DeckhouseConfigSpec{
			Enabled: cfg.Spec.Enabled,
		},
	}, nil
}

const (
	d8ConfigGroup      = "deckhouse_config_metrics"
	d8ConfigMetricName = "deckhouse_config_obsolete_version"
)

func updateDeckhouseConfigStatuses(input *go_hook.HookInput) error {
	addonOperator := operator.Internals().AddonOperator()

	knownModuleNames := set.New(addonOperator.GetModuleNames()...)

	allConfigs := internal.ConfigsFromSnapshot(input.Snapshots["configs"])

	for _, cfg := range allConfigs {
		statusPatch := getConfigStatus(cfg, addonOperator, knownModuleNames)
		input.LogEntry.Infof("Patch /status for %s: enabled=%s, status=%s", cfg.GetName(), statusPatch.Enabled, statusPatch.Status)
		input.PatchCollector.MergePatch(statusPatch, "deckhouse.io/v1", "DeckhouseConfig", "", cfg.GetName(), object_patch.WithSubresource("/status"))
	}

	// Export metrics for configs with obsolete versions.
	input.MetricsCollector.Expire(d8ConfigGroup)
	for _, cfg := range allConfigs {
		modName := cfg.GetName()
		modChain := conversion.Registry().Chain(modName)

		if modChain == nil {
			continue
		}

		cfgVersion := cfg.Spec.ConfigVersion
		latestVersion := modChain.LatestVersion()

		if cfgVersion != latestVersion {
			input.MetricsCollector.Set(d8ConfigMetricName, 1.0, map[string]string{
				"module":  modName,
				"version": strconv.Itoa(cfgVersion),
				"latest":  strconv.Itoa(latestVersion),
			}, metrics.WithGroup(d8ConfigGroup))
		}
	}

	return nil
}

func getConfigStatus(cfg *d8config_v1.DeckhouseConfig, addonOperator *operator.AddonOperatorWrapper, possibleNames set.Set) statusPatch {
	if cfg.GetName() == "global" {
		return statusPatch{
			Enabled: "Always On",
		}
	}

	if !possibleNames.Has(cfg.GetName()) {
		return statusPatch{
			Status: "Unknown module name",
		}
	}

	sp := statusPatch{}

	isEnabled := addonOperator.IsModuleEnabled(cfg.GetName())
	if isEnabled {
		sp.Enabled = "Enabled"
	} else {
		sp.Enabled = "Disabled"
	}

	mod := addonOperator.GetModule(cfg.GetName())

	enabledByBundle := internal.MergeEnabled(mod.CommonStaticConfig.IsEnabled, mod.StaticConfig.IsEnabled)
	enabledByConfig := cfg.Spec.Enabled != nil && *cfg.Spec.Enabled
	disabledByConfig := cfg.Spec.Enabled != nil && !*cfg.Spec.Enabled
	// Module is disabled by default but enabled in DeckhouseConfig.
	if !enabledByBundle && enabledByConfig {
		sp.Enabled = "Enabled by config"
	}
	// Module enabled by bundle or by DeckhouseConfig but disabled by 'enabled' script.
	if internal.MergeEnabled(&enabledByBundle, cfg.Spec.Enabled) && !isEnabled {
		sp.Enabled = "Disabled by script"
	}
	// Module enabled by bundle, but disabled in DeckhouseConfig.
	if enabledByBundle && disabledByConfig && !isEnabled {
		sp.Enabled = "Disabled by config"
	}

	// Calculate status for enabled module.
	if isEnabled {
		sp.Status = "Running"
		if mod.State.Phase == module_manager.CanRunHelm {
			sp.Status = "Ready"
		}

		// TODO: Change addon-operator to get error for module.
		lastHookErr := mod.State.GetLastHookErr()
		if lastHookErr != nil {
			sp.Status = fmt.Sprintf("HookError: %v", lastHookErr)
		}
		if mod.State.LastModuleErr != nil {
			sp.Status = fmt.Sprintf("ModuleError: %v", mod.State.LastModuleErr)
		}
	}

	return sp
}

type statusPatch d8config_v1.DeckhouseConfigStatus

func (sp statusPatch) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"status": d8config_v1.DeckhouseConfigStatus(sp),
	}

	return json.Marshal(m)
}
