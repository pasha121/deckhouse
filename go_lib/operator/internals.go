/*
Copyright 2021 Flant JSC

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

package operator

import (
	"sync"

	addon_operator "github.com/flant/addon-operator/pkg/addon-operator"
)

var (
	internals     *InternalsRegistry
	internalsOnce sync.Once
)

func Internals() *InternalsRegistry {
	internalsOnce.Do(func() {
		internals = new(InternalsRegistry)
	})
	return internals
}

type InternalsRegistry struct {
	addonOperator *AddonOperatorWrapper
}

func (r *InternalsRegistry) AddonOperator() *AddonOperatorWrapper {
	return r.addonOperator
}

func (r *InternalsRegistry) WrapAddonOperator(addonOperator *addon_operator.AddonOperator) {
	r.addonOperator = &AddonOperatorWrapper{addonOperator}
}
