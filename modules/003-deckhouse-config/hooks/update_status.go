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
	"strconv"

	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	"github.com/flant/addon-operator/pkg/module_manager/go_hook/metrics"
	"github.com/flant/addon-operator/sdk"
	"github.com/flant/shell-operator/pkg/kube/object_patch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"

	d8config "github.com/deckhouse/deckhouse/go_lib/deckhouse-config"
	"github.com/deckhouse/deckhouse/go_lib/deckhouse-config/conversion"
	d8config_v1 "github.com/deckhouse/deckhouse/go_lib/deckhouse-config/v1"
)

/*
This hook tracks changes in ModuleConfig resources and updates
their statuses.
It uses AddonOperator dependency to get enabled state for all modules
and get access to each module state.

ModuleConfig status consists of:
- 'enabled' field - describes a module's enabled state:
    * Enabled
    * Disabled
    * Disabled by script - module is disabled by 'enabled' script
    * Enabled/Disabled by config - module state is determined by ModuleConfig
- 'status' field - describes state of the module:
    * Unknown module name - ModuleConfig resource name is not a known module name.
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
			Kind:                         "ModuleConfig",
			FilterFunc:                   filterModuleConfigForStatus,
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
}, updateModuleConfigStatuses)

// filterModuleConfigForStatus returns name, enabled flag and the current status from ModuleConfig object.
func filterModuleConfigForStatus(unstructured *unstructured.Unstructured) (go_hook.FilterResult, error) {
	var cfg d8config_v1.ModuleConfig

	err := sdk.FromUnstructured(unstructured, &cfg)
	if err != nil {
		return nil, err
	}

	// Extract name and enabled.
	return &d8config_v1.ModuleConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: cfg.Name,
		},
		Spec: d8config_v1.ModuleConfigSpec{
			Enabled: cfg.Spec.Enabled,
		},
		Status: cfg.Status,
	}, nil
}

const (
	d8ConfigGroup      = "deckhouse_config_metrics"
	d8ConfigMetricName = "deckhouse_config_obsolete_version"
)

func updateModuleConfigStatuses(input *go_hook.HookInput) error {
	allConfigs := snapshotToModuleConfigList(input.Snapshots["configs"])

	for _, cfg := range allConfigs {
		cfgStatus := d8config.Service().StatusReporter().ForConfig(cfg)
		statusPatch := makeStatusPatch(cfg, cfgStatus)
		if statusPatch != nil {
			input.LogEntry.Infof("Patch /status for %s: change (enabled=%s, status=%s) to enabled=%s, status=%s", cfg.GetName(), cfg.Status.Enabled, cfg.Status.Status, statusPatch.Enabled, statusPatch.Status)
			input.PatchCollector.MergePatch(statusPatch, "deckhouse.io/v1", "ModuleConfig", "", cfg.GetName(), object_patch.WithSubresource("/status"))
		}
	}

	// Export metrics for configs with obsolete versions.
	input.MetricsCollector.Expire(d8ConfigGroup)
	for _, cfg := range allConfigs {
		modName := cfg.GetName()
		modChain := conversion.Registry().Chain(modName)

		if modChain == nil {
			continue
		}

		cfgVersion := cfg.Spec.Version
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

func makeStatusPatch(cfg *d8config_v1.ModuleConfig, cfgStatus d8config.Status) *statusPatch {
	if cfg.Status.Enabled == cfgStatus.Enabled && cfg.Status.Status == cfgStatus.Status {
		return nil
	}

	return &statusPatch{
		Status:  cfgStatus.Status,
		Enabled: cfgStatus.Enabled,
	}
}

type statusPatch d8config_v1.ModuleConfigStatus

func (sp statusPatch) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"status": d8config_v1.ModuleConfigStatus(sp),
	}

	return json.Marshal(m)
}

// snapshotToModuleConfigList returns a typed array of ModuleConfig items from untyped items in the snapshot.
func snapshotToModuleConfigList(snapshot []go_hook.FilterResult) []*d8config_v1.ModuleConfig {
	configs := make([]*d8config_v1.ModuleConfig, 0)
	for _, item := range snapshot {
		cfg := item.(*d8config_v1.ModuleConfig)
		configs = append(configs, cfg)
	}
	return configs
}
