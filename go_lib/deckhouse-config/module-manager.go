package deckhouse_config

import (
	"github.com/flant/addon-operator/pkg/module_manager"
	"github.com/flant/addon-operator/pkg/values/validation"
)

// ModuleManager interface is a part of addon-operator's ModuleManager interface
// with methods needed for deckhouse-config package.
type ModuleManager interface {
	IsModuleEnabled(modName string) bool
	GetModule(modName string) *module_manager.Module
	GetModuleNames() []string
	GetValuesValidator() *validation.ValuesValidator
}
