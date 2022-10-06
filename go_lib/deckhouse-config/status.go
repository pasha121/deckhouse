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
	v1 "github.com/deckhouse/deckhouse/go_lib/deckhouse-config/v1"
	"github.com/deckhouse/deckhouse/go_lib/set"
	"github.com/flant/addon-operator/pkg/module_manager"
)

type ConfigStatus struct {
	Enabled string
	Status  string
}

type configStatus struct {
	moduleManager ModuleManager
	possibleNames set.Set
}

func NewConfigStatus(mm ModuleManager, possibleNames set.Set) *configStatus {
	return &configStatus{
		moduleManager: mm,
		possibleNames: possibleNames,
	}
}

func (s *configStatus) Get(cfg *v1.DeckhouseConfig) ConfigStatus {
	if cfg.GetName() == "global" {
		return ConfigStatus{
			Enabled: "Always On",
		}
	}

	if !s.possibleNames.Has(cfg.GetName()) {
		return ConfigStatus{
			Status: "Unknown module name",
		}
	}

	status := ConfigStatus{}

	// First, get effective "enabled" from ModuleManager.
	isModuleEnabled := s.moduleManager.IsModuleEnabled(cfg.GetName())
	if isModuleEnabled {
		status.Enabled = "Enabled"
	} else {
		status.Enabled = "Disabled"
	}

	mod := s.moduleManager.GetModule(cfg.GetName())

	// Consider merged static enabled flags as '*Enabled flags from the bundle'.
	enabledByBundle := mergeEnabled(mod.CommonStaticConfig.IsEnabled, mod.StaticConfig.IsEnabled)

	enabledByConfig := cfg.Spec.Enabled != nil && *cfg.Spec.Enabled
	disabledByConfig := cfg.Spec.Enabled != nil && !*cfg.Spec.Enabled

	// No '*Enabled' flags in the bundle, 'enabled: true' in the DeckhouseConfig, enabled script returns 'true'.
	if !enabledByBundle && enabledByConfig {
		status.Enabled = "Enabled by config"
	}
	// '*Enabled: true' in the bundle or 'enabled: true' in the DeckhouseConfig, but enabled script returns 'false'.
	if mergeEnabled(&enabledByBundle, cfg.Spec.Enabled) && !isModuleEnabled {
		status.Enabled = "Disabled by script"
	}

	// '*Enabled: true' in the bundle, 'enabled: false' in the DeckhouseConfig, module is disabled.
	if enabledByBundle && disabledByConfig && !isModuleEnabled {
		status.Enabled = "Disabled by config"
	}

	// Calculate status for enabled module.
	if isModuleEnabled {
		status.Status = "Running"
		if mod.State.Phase == module_manager.CanRunHelm {
			status.Status = "Ready"
		}

		lastHookErr := mod.State.GetLastHookErr()
		if lastHookErr != nil {
			status.Status = fmt.Sprintf("HookError: %v", lastHookErr)
		}
		if mod.State.LastModuleErr != nil {
			status.Status = fmt.Sprintf("ModuleError: %v", mod.State.LastModuleErr)
		}
	}

	return status
}

// mergeEnabled merges enabled flags. Enabled flag can be nil.
//
// If all flags are nil, then false is returned — module is disabled by default.
// Note: copy-paste from AddonOperator.moduleManager
func mergeEnabled(enabledFlags ...*bool) bool {
	result := false
	for _, enabled := range enabledFlags {
		if enabled == nil {
			continue
		} else {
			result = *enabled
		}
	}

	return result
}
