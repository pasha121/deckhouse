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

package internal

import (
	"github.com/flant/addon-operator/pkg/module_manager/go_hook"

	d8config_v1 "github.com/deckhouse/deckhouse/go_lib/deckhouse-config/v1"
)

// ConfigsFromSnapshot returns a typed array of DeckhouseConfig items from untyped items in the snapshot.
func ConfigsFromSnapshot(snapshot []go_hook.FilterResult) []*d8config_v1.DeckhouseConfig {
	configs := make([]*d8config_v1.DeckhouseConfig, 0)
	for _, item := range snapshot {
		cfg := item.(*d8config_v1.DeckhouseConfig)
		configs = append(configs, cfg)
	}
	return configs
}
