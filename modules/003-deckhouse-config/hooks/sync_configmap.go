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
	"fmt"

	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	"github.com/flant/addon-operator/sdk"
	"github.com/flant/shell-operator/pkg/kube/object_patch"
	"github.com/flant/shell-operator/pkg/kube_events_manager/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"

	d8config "github.com/deckhouse/deckhouse/go_lib/deckhouse-config"
	d8config_v1 "github.com/deckhouse/deckhouse/go_lib/deckhouse-config/v1"
	"github.com/deckhouse/deckhouse/go_lib/set"
)

/**
This hook tracks changes in DeckhouseConfig resources and
updates ConfigMap accordingly.
It also converts DeckhouseConfig's field configValues
to the latest version and validates them against related config-values.yaml schema.

Notes:
- No way to ignore specific module configs. All known configs are considered as configuration source.
- Deletion of DeckhouseConfig resource leads to immediate converge. But delete may be a part of
  recreation. Logic to postpone deletion handling may be useful.
*/

var _ = sdk.RegisterFunc(&go_hook.HookConfig{
	Queue: "/modules/deckhouse-config/updater",
	Kubernetes: []go_hook.KubernetesConfig{
		{
			Name:                   "configs",
			ApiVersion:             "deckhouse.io/v1",
			Kind:                   "DeckhouseConfig",
			WaitForSynchronization: pointer.BoolPtr(true),
			FilterFunc:             filterDeckhouseConfigs,
		},
		{
			Name:       "generated-cm",
			ApiVersion: "v1",
			Kind:       "ConfigMap",
			NamespaceSelector: &types.NamespaceSelector{
				NameSelector: &types.NameSelector{
					MatchNames: []string{d8config.DeckhouseNS},
				},
			},
			NameSelector: &types.NameSelector{
				MatchNames: []string{d8config.GeneratedConfigMapName},
			},
			ExecuteHookOnEvents:          pointer.BoolPtr(false),
			ExecuteHookOnSynchronization: pointer.BoolPtr(false),
			FilterFunc:                   filterGeneratedConfigMap,
		},
	},
}, updateGeneratedConfigMap)

// filterModuleSettings returns spec for DeckhouseConfig objects.
func filterDeckhouseConfigs(unstructured *unstructured.Unstructured) (go_hook.FilterResult, error) {
	var cfg d8config_v1.DeckhouseConfig

	err := sdk.FromUnstructured(unstructured, &cfg)
	if err != nil {
		return nil, err
	}

	// Extract name and spec into empty DeckhouseConfig.
	return &d8config_v1.DeckhouseConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: cfg.Name,
		},
		Spec: cfg.Spec,
	}, nil
}

type configData map[string]string

// filterGeneratedConfigMap returns Data field for ConfigMap.
func filterGeneratedConfigMap(unstructured *unstructured.Unstructured) (go_hook.FilterResult, error) {
	var cm v1.ConfigMap

	err := sdk.FromUnstructured(unstructured, &cm)
	if err != nil {
		return nil, err
	}

	return configData(cm.Data), nil
}

// updateGeneratedConfigMap converts DeckhouseConfig resources into ConfigMap data.
func updateGeneratedConfigMap(input *go_hook.HookInput) error {
	possibleNames := d8config.Service().PossibleNames()
	allConfigs := knownConfigsFromSnapshot(input.Snapshots["configs"], possibleNames)

	for _, cfg := range allConfigs {
		err := d8config.Service().ConfigValidator().ValidateConfig(cfg)
		if err != nil {
			return err
		}
	}

	cmData, err := d8config.Service().Transformer().DeckhouseConfigListToConfigMap(allConfigs)
	if err != nil {
		return fmt.Errorf("convert DeckhouseConfig objects to ConfigMap: %s", err)
	}

	cm := d8config.GeneratedConfigMap(cmData)
	input.PatchCollector.Create(cm, object_patch.UpdateIfExists())

	return nil
}

func knownConfigsFromSnapshot(snapshot []go_hook.FilterResult, possibleNames set.Set) []*d8config_v1.DeckhouseConfig {
	configs := make([]*d8config_v1.DeckhouseConfig, 0)
	for _, item := range snapshot {
		cfg := item.(*d8config_v1.DeckhouseConfig)
		// Ignore configurations for unknown modules.
		if !possibleNames.Has(cfg.GetName()) {
			continue
		}
		configs = append(configs, cfg)
	}
	return configs
}
