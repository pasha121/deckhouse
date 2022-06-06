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

	"github.com/flant/addon-operator/pkg/utils"

	"github.com/deckhouse/deckhouse/go_lib/deckhouse-config/conversion"
	v1 "github.com/deckhouse/deckhouse/go_lib/deckhouse-config/v1"
)

// ConfigValidator is a validator for values in ModuleConfig.
type ConfigValidator struct {
	valuesValidator ValuesValidator
}

func NewConfigValidator(valuesValidator ValuesValidator) *ConfigValidator {
	return &ConfigValidator{
		valuesValidator: valuesValidator,
	}
}

// ValuesValidator is a part of ValuesValidator from addon-operator with needed
// methods to validate config values.
type ValuesValidator interface {
	ValidateGlobalConfigValues(values utils.Values) error
	ValidateModuleConfigValues(moduleName string, values utils.Values) error
}

// ValidateConfig converts values in ModuleConfig to latest version and validates
// them against OpenAPI schema defined in related config-values.yaml file.
func (c *ConfigValidator) ValidateConfig(cfg *v1.ModuleConfig) error {
	// Ignore conversion and validation for empty values if module is not enabled explicitly.
	if len(cfg.Spec.Settings) == 0 && (cfg.Spec.Enabled == nil || !*cfg.Spec.Enabled) {
		return nil
	}

	origVersion := cfg.Spec.Version

	// Run registered conversions if version is not the latest.
	versionMsg := fmt.Sprintf("version %d", origVersion)
	chain := conversion.Registry().Chain(cfg.GetName())
	if chain != nil && chain.LatestVersion() != cfg.Spec.Version {
		newVersion, newValues, err := chain.ConvertToLatest(cfg.Spec.Version, cfg.Spec.Settings)
		if err != nil {
			return fmt.Errorf("convert %s config values from version %d to latest: %v", cfg.GetName(), cfg.Spec.Version, err)
		}
		cfg.Spec.Version = newVersion
		cfg.Spec.Settings = newValues
		versionMsg = fmt.Sprintf("version %d converted to %d", origVersion, newVersion)
	}

	err := c.validateValues(cfg.GetName(), cfg.Spec.Settings)
	if err != nil {
		return fmt.Errorf("%s config values of version %s are not valid: %v", cfg.GetName(), versionMsg, err)
	}

	return nil
}

// validateValues validates values using ValuesValidator.
// cfgName arg is a kebab-cased name of ModuleConfig.
// cfgValues is a content of configValues.
// (Note: cfgValues are a 'plain values' without root key with the module name).
func (c *ConfigValidator) validateValues(cfgName string, cfgValues map[string]interface{}) error {
	// Ignore empty validator.
	if c.valuesValidator == nil {
		return nil
	}

	valuesKey := valuesKeyFromObjectName(cfgName)
	values := map[string]interface{}{
		valuesKey: cfgValues,
	}

	if cfgName == "global" {
		return c.valuesValidator.ValidateGlobalConfigValues(values)
	}

	return c.valuesValidator.ValidateModuleConfigValues(valuesKey, values)
}

func valuesKeyFromObjectName(name string) string {
	if name == "global" {
		return name
	}
	return utils.ModuleNameToValuesKey(name)
}
