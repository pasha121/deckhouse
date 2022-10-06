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
	"fmt"
	"strconv"

	kcm "github.com/flant/addon-operator/pkg/kube_config_manager"
	"github.com/flant/addon-operator/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/deckhouse/deckhouse/go_lib/deckhouse-config/conversion"
	d8config_v1 "github.com/deckhouse/deckhouse/go_lib/deckhouse-config/v1"
	"github.com/deckhouse/deckhouse/go_lib/set"
)

type transformer struct {
	// Possible names for DeckhouseConfig objects (known modules + "global").
	possibleNames set.Set
}

func NewTransformer(possibleNames set.Set) *transformer {
	return &transformer{
		possibleNames: possibleNames,
	}
}

// DeckhouseConfigListToConfigMap creates new Data for ConfigMap from existing DeckhouseConfig objects.
// It creates module sections for known modules using a cached set of possible names.
func (t *transformer) DeckhouseConfigListToConfigMap(allConfigs []*d8config_v1.DeckhouseConfig) (map[string]string, error) {
	data := make(map[string]string)

	// Note: possibleNames are kebab-cased, cfg.Name should also be kebab-cased.
	for _, cfg := range allConfigs {
		name := cfg.GetName()

		// Ignore unknown module names.
		if !t.possibleNames.Has(name) {
			continue
		}

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

// ConfigMapToDeckhouseConfigList returns a list of DeckhouseConfig objects.
// It transforms 'global' section and all modules sections in ConfigMap/deckhouse.
// Conversion chain is triggered for each section to convert values to the latest
// version. If module has no conversions, configVersion: 1 is used.
// It ignores sections with unknown names.
func (t *transformer) ConfigMapToDeckhouseConfigList(cmData map[string]string) ([]*d8config_v1.DeckhouseConfig, error) {
	// Use ConfigMap parser from addon-operator.
	cfg, err := kcm.ParseConfigMapData(cmData)
	if err != nil {
		return nil, fmt.Errorf("parse cm/deckhouse data: %v", err)
	}

	// Construct list of sections.
	sections := make([]*configMapSection, 0)
	sections = append(sections, &configMapSection{
		name:      "global",
		valuesKey: "global",
		values:    cfg.Global.Values,
		isEnabled: nil,
	})
	for _, modCfg := range cfg.Modules {
		// Note: possibleNames items and modCfg.ModuleName keys are kebab-cased, modCfg.ModuleConfigKey is camelCased.
		// Ignore unknown module names.
		if !t.possibleNames.Has(modCfg.ModuleName) {
			continue
		}

		sections = append(sections, &configMapSection{
			name:      modCfg.ModuleName,
			valuesKey: modCfg.ModuleConfigKey,
			values:    modCfg.Values,
			isEnabled: modCfg.IsEnabled,
		})
	}

	// Transform ConfigMap sections to DeckhouseConfig objects.
	cfgList := make([]*d8config_v1.DeckhouseConfig, 0)
	for _, section := range sections {
		// Transform ConfigMap section to DeckhouseConfig object.
		cfg, err := section.getDeckhouseConfig()
		if err != nil {
			return nil, err
		}

		// Convert values to the latest schema if conversion chain is present.
		chain := conversion.Registry().Chain(cfg.GetName())
		if chain != nil {
			newVersion, newValues, err := chain.ConvertToLatest(cfg.Spec.ConfigVersion, cfg.Spec.ConfigValues)
			if err != nil {
				return nil, err
			}
			cfg.Spec.ConfigVersion = newVersion
			cfg.Spec.ConfigValues = newValues
		}

		cfgList = append(cfgList, cfg)
	}

	return cfgList, nil
}

type configMapSection struct {
	name      string
	valuesKey string
	values    utils.Values
	isEnabled *bool
}

// getValuesMap returns a values map or nil for module or global section.
func (s *configMapSection) getValuesMap() (map[string]interface{}, error) {
	untypedValues := s.values[s.valuesKey]

	isValidType := true
	switch v := untypedValues.(type) {
	case map[string]interface{}:
		// Module section is not empty, and it is a map.
		return v, nil
	case nil:
		// Module values are nil when ConfigMap has Enabled flag without module section.
		// Transform empty values to the empty 'configValues' field.
		return nil, nil
	case string:
		// Transform empty string to the empty 'configValues' field.
		if v != "" {
			isValidType = false
		}
	case []interface{}:
		// Transform empty string to the empty 'configValues' field.
		if len(v) != 0 {
			isValidType = false
		}
	default:
		// Consider other types are not valid.
		isValidType = false
	}
	if !isValidType {
		return nil, fmt.Errorf("configmap section '%s' is not an object, need map[string]interface{}, got %T:(%+v)", s.valuesKey, untypedValues, untypedValues)
	}
	return nil, nil
}

// getDeckhouseConfig constructs DeckhouseConfig object from ConfigMap's section.
func (s *configMapSection) getDeckhouseConfig() (*d8config_v1.DeckhouseConfig, error) {
	cfgValues, err := s.getValuesMap()
	if err != nil {
		return nil, err
	}

	return &d8config_v1.DeckhouseConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DeckhouseConfig",
			APIVersion: "deckhouse.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: s.name,
		},
		Spec: d8config_v1.DeckhouseConfigSpec{
			ConfigVersion: 1,
			Enabled:       s.isEnabled,
			ConfigValues:  cfgValues,
		},
	}, nil
}
