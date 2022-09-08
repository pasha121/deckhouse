package operator

import (
	addon_operator "github.com/flant/addon-operator/pkg/addon-operator"
	"github.com/flant/addon-operator/pkg/module_manager"
)

type AddonOperatorWrapper struct {
	Operator *addon_operator.AddonOperator
}

func (w *AddonOperatorWrapper) IsModuleEnabled(modName string) bool {
	w.ensureModuleManager()
	return w.Operator.ModuleManager.IsModuleEnabled(modName)
}

func (w *AddonOperatorWrapper) GetModule(modName string) *module_manager.Module {
	w.ensureModuleManager()
	return w.Operator.ModuleManager.GetModule(modName)
}

func (w *AddonOperatorWrapper) GetModuleNames() []string {
	w.ensureModuleManager()
	return w.Operator.ModuleManager.GetModuleNames()
}

func (w *AddonOperatorWrapper) ensureModuleManager() {
	if w.Operator == nil {
		panic("Underlying AddonOperator is nil")
	}
	if w.Operator.ModuleManager == nil {
		panic("Underlying AddonOperator.ModuleManager is nil")
	}
}
