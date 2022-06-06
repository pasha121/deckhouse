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
	"strings"

	kcm "github.com/flant/addon-operator/pkg/kube_config_manager"
	"github.com/flant/addon-operator/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/deckhouse/deckhouse/go_lib/deckhouse-config/conversion"
	d8config_v1 "github.com/deckhouse/deckhouse/go_lib/deckhouse-config/v1"
	"github.com/deckhouse/deckhouse/go_lib/set"
)

type Transformer struct {
	// Possible names for ModuleConfig objects (known modules + "global").
	possibleNames set.Set
}

func NewTransformer(possibleNames set.Set) *Transformer {
	return &Transformer{
		possibleNames: possibleNames,
	}
}

// ModuleConfigListToConfigMap creates new Data for ConfigMap from existing ModuleConfig objects.
// It creates module sections for known modules using a cached set of possible names.
func (t *Transformer) ModuleConfigListToConfigMap(allConfigs []*d8config_v1.ModuleConfig) (map[string]string, error) {
	data := make(map[string]string)

	// Note: possibleNames are kebab-cased, cfg.Name should also be kebab-cased.
	for _, cfg := range allConfigs {
		name := cfg.GetName()

		// Ignore unknown module names.
		if !t.possibleNames.Has(name) {
			continue
		}

		// Put module section to ConfigMap if ModuleConfig object has at least one field in values.
		valuesKey := utils.ModuleNameToValuesKey(name)
		cfgValues := cfg.Spec.Settings
		if len(cfgValues) > 0 {
			sectionBytes, err := yaml.Marshal(cfg.Spec.Settings)
			if err != nil {
				return nil, err
			}
			data[valuesKey] = string(sectionBytes)
		}

		// Prevent useless 'globalEnabled' key.
		if name == "global" {
			continue
		}

		// Put '*Enabled' flag to ConfigMap if 'enabled' is present in ModuleConfig object.
		if cfg.Spec.Enabled != nil {
			enabledKey := valuesKey + "Enabled"
			data[enabledKey] = strconv.FormatBool(*cfg.Spec.Enabled)
		}
	}

	return data, nil
}

// ConfigMapToModuleConfigList returns a list of ModuleConfig objects.
// It transforms 'global' section and all modules sections in ConfigMap/deckhouse.
// Conversion chain is triggered for each section to convert values to the latest
// version. If module has no conversions, 'version: 1' is used.
// It ignores sections with unknown names.
func (t *Transformer) ConfigMapToModuleConfigList(cmData map[string]string) ([]*d8config_v1.ModuleConfig, []string, error) {
	// Messages to log.
	msgs := make([]string, 0)

	// Use ConfigMap parser from addon-operator.
	cfg, err := kcm.ParseConfigMapData(cmData)
	if err != nil {
		return nil, msgs, fmt.Errorf("parse cm/deckhouse data: %v", err)
	}

	// Construct list of sections from *KubeConfig objects.
	sections := make([]*configMapSection, 0)
	var globalValues utils.Values
	if cfg.Global != nil {
		globalValues = cfg.Global.Values
	}
	sections = append(sections, &configMapSection{
		name:        "global",
		valuesKey:   "global",
		values:      globalValues,
		enabledFlag: nil,
	})
	for _, modCfg := range cfg.Modules {
		sections = append(sections, &configMapSection{
			name:        modCfg.ModuleName,
			valuesKey:   modCfg.ModuleConfigKey,
			values:      modCfg.Values,
			enabledFlag: modCfg.IsEnabled,
		})
	}

	// Transform ConfigMap sections to ModuleConfig objects.
	cfgList := make([]*d8config_v1.ModuleConfig, 0)
	for _, section := range sections {
		// Note: possibleNames items and modCfg.ModuleName keys are kebab-cased, modCfg.ModuleConfigKey is camelCased.
		// Ignore unknown module names.
		if !t.possibleNames.Has(section.name) {
			msgs = append(msgs, fmt.Sprintf("migrate '%s': module unknown, ignore", section.name))
			continue
		}

		cfg, msg, err := section.getModuleConfig()
		if err != nil {
			return nil, nil, err
		}
		msgs = append(msgs, msg)
		if cfg != nil {
			cfgList = append(cfgList, cfg)
		}
	}

	return cfgList, msgs, nil
}

type configMapSection struct {
	name        string
	valuesKey   string
	values      utils.Values
	enabledFlag *bool
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
		// Array is not a valid module section, but it is ok if array is empty, just ignore it.
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

func (s *configMapSection) convertValues() (map[string]interface{}, int, error) {
	// Values without conversion has version 1.
	latestVersion := 1
	latestValues, err := s.getValuesMap()
	if err != nil {
		return nil, 0, err
	}

	chain := conversion.Registry().Chain(s.name)
	if chain != nil {
		latestVersion, latestValues, err = chain.ConvertToLatest(latestVersion, latestValues)
		if err != nil {
			return nil, 0, err
		}
	}

	return latestValues, latestVersion, nil
}

// getModuleConfig constructs ModuleConfig object from ConfigMap's section.
// It converts section values to the latest version of module settings.
func (s *configMapSection) getModuleConfig() (*d8config_v1.ModuleConfig, string, error) {
	// Convert values to the latest schema if conversion chain is present.
	values, version, err := s.convertValues()
	if err != nil {
		return nil, "", err
	}

	cfg := &d8config_v1.ModuleConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ModuleConfig",
			APIVersion: "deckhouse.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: s.name,
		},
		Spec: d8config_v1.ModuleConfigSpec{},
	}

	msgs := make([]string, 0)

	if len(s.values) > 0 {
		msgs = append(msgs, "has values")
	} else {
		msgs = append(msgs, "no values")
	}

	if len(values) > 0 {
		msgs = append(msgs, fmt.Sprintf("values converted to version %d", version))
		cfg.Spec.Settings = values
		cfg.Spec.Version = version
	}

	if len(s.values) > 0 && len(values) == 0 {
		msgs = append(msgs, "converted to empty values")
	}

	// Enabled flag is not applicable for global section.
	if s.name != "global" {
		if s.enabledFlag != nil {
			msgs = append(msgs, "has enabled flag")
			cfg.Spec.Enabled = s.enabledFlag
		} else {
			msgs = append(msgs, "no enabled flag")
		}
	}

	if s.enabledFlag == nil && len(values) == 0 {
		cfg = nil
		msgs = append(msgs, "ignore creating empty object")
	}

	msg := fmt.Sprintf("section '%s': %s", s.name, strings.Join(msgs, ", "))

	return cfg, msg, nil
}
