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
	"github.com/deckhouse/deckhouse/go_lib/set"
	"sync"
)

// deckhouse-config Service is a middleware between ModuleManager instance and hooks to
// safely retrieve information about modules.

var (
	serviceInstance     *service
	serviceInstanceLock sync.Mutex
)

func InitService(mm ModuleManager) {
	serviceInstanceLock.Lock()
	defer serviceInstanceLock.Unlock()

	possibleNames := set.New(mm.GetModuleNames()...)
	possibleNames.Add("global")

	serviceInstance = &service{
		moduleManager:   mm,
		possibleNames:   possibleNames,
		transformer:     NewTransformer(possibleNames),
		configValidator: NewConfigValidator(mm.GetValuesValidator()),
		configStatus:    NewConfigStatus(mm, possibleNames),
	}
}

func Service() *service {
	if serviceInstance == nil {
		panic("deckhouse-config Service is not initialized")
	}
	return serviceInstance
}

type service struct {
	moduleManager   ModuleManager
	possibleNames   set.Set
	transformer     *transformer
	configValidator *configValidator
	configStatus    *configStatus
}

func (srv *service) PossibleNames() set.Set {
	return srv.possibleNames
}

func (srv *service) Transformer() *transformer {
	return srv.transformer
}

func (srv *service) ConfigValidator() *configValidator {
	return srv.configValidator
}

func (srv *service) ConfigStatus() *configStatus {
	return srv.configStatus
}
