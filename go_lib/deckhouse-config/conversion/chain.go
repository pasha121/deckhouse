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

package conversion

import (
	"fmt"
	"sync"
)

// ModuleChain is a chain of conversions for module.
type ModuleChain struct {
	m sync.RWMutex

	moduleName string

	// version -> convertor
	conversions map[int]*Conversion

	latestVersion int
}

func NewModuleChain(moduleName string) *ModuleChain {
	return &ModuleChain{
		moduleName:  moduleName,
		conversions: make(map[int]*Conversion),
	}
}

func (c *ModuleChain) Add(conversion *Conversion) {
	c.m.Lock()
	defer c.m.Unlock()

	c.conversions[conversion.Source] = conversion

	// Update latest version.
	if c.latestVersion == 0 || conversion.Target > c.latestVersion {
		c.latestVersion = conversion.Target
	}
}

func (c *ModuleChain) ConvertToLatest(fromVersion int, values map[string]interface{}) (int, map[string]interface{}, error) {
	c.m.Lock()
	defer c.m.Unlock()

	maxTries := len(c.conversions)

	tries := 0
	currentVersion := fromVersion
	currentValues := values
	for {
		conv := c.conversions[currentVersion]
		if conv == nil {
			return 0, nil, fmt.Errorf("convert from %d: conversion chain interrupt: no conversion from %d", fromVersion, currentVersion)
		}
		newVer := conv.Target
		newValues, err := conv.Convert(currentValues)
		if err != nil {
			return 0, nil, fmt.Errorf("convert from %d: conversion chain error for %d: %v", fromVersion, currentVersion, err)
		}

		// Stop after converting to the latest version.
		if newVer == c.latestVersion {
			return newVer, newValues, nil
		}

		currentVersion = newVer
		currentValues = newValues

		// Prevent looped conversions.
		tries++
		if tries > maxTries {
			return 0, nil, fmt.Errorf("convert from %d: conversion chain too long or looped", fromVersion)
		}
	}
}

func (c *ModuleChain) Conversion(srcVersion int) *Conversion {
	c.m.RLock()
	defer c.m.RUnlock()

	return c.conversions[srcVersion]
}

func (c *ModuleChain) LatestVersion() int {
	return c.latestVersion
}

// Count returns a number of registered conversions for the module.
func (c *ModuleChain) Count() int {
	c.m.RLock()
	defer c.m.RUnlock()

	return len(c.conversions)
}

// HasVersion returns whether module has registered conversion for version.
func (c *ModuleChain) HasVersion(version int) bool {
	c.m.RLock()
	defer c.m.RUnlock()

	_, has := c.conversions[version]
	return has
}

// VersionList returns all versions for the module.
func (c *ModuleChain) VersionList() []int {
	c.m.RLock()
	defer c.m.RUnlock()
	versions := make([]int, 0)
	for ver := range c.conversions {
		versions = append(versions, ver)
	}
	return versions
}
