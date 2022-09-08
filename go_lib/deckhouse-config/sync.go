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

package deckhouse_config

import (
	"strconv"

	"github.com/flant/addon-operator/pkg/utils"
	"sigs.k8s.io/yaml"

	d8config_v1 "github.com/deckhouse/deckhouse/go_lib/deckhouse-config/v1"
)

// SyncFromDeckhouseConfigs creates new Data for ConfigMap from DeckhouseConfig objects.
func SyncFromDeckhouseConfigs(allConfigs []*d8config_v1.DeckhouseConfig) (map[string]string, error) {
	data := make(map[string]string)

	for _, cfg := range allConfigs {
		name := cfg.GetName()

		valuesKey := utils.ModuleNameToValuesKey(name)
		if cfg.Spec.ConfigValues != nil {
			sectionBytes, err := yaml.Marshal(cfg.Spec.ConfigValues)
			if err != nil {
				return nil, err
			}
			data[valuesKey] = string(sectionBytes)
		}

		// Prevent creating 'globalEnabled' key.
		if name == "global" {
			continue
		}

		if cfg.Spec.Enabled != nil {
			enabledKey := valuesKey + "Enabled"
			data[enabledKey] = strconv.FormatBool(*cfg.Spec.Enabled)
		}
	}

	return data, nil
}
