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

package main

import (
	"context"
	"path/filepath"
	"testing"

	d8config "github.com/deckhouse/deckhouse/go_lib/deckhouse-config"
	module_manager "github.com/deckhouse/deckhouse/go_lib/deckhouse-config/module-manager"
	. "github.com/onsi/gomega"
	"github.com/slok/kubewebhook/v2/pkg/model"
	kwhvalidating "github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	_ "deckhouse-config-webhook/testdata/modules/001-module-one/conversion"
)

func runValidate(t *testing.T, rootDir string, manifest string) (*kwhvalidating.ValidatorResult, error) {
	g := NewWithT(t)

	// Init d8config with basic ModuleManager.
	globalHooksDir := filepath.Join(rootDir, "global-hooks")
	modulesDir := filepath.Join(rootDir, "modules")
	mm, err := module_manager.InitBasic(globalHooksDir, modulesDir)
	g.Expect(err).ShouldNot(HaveOccurred())
	d8config.InitService(mm)

	cfgValidator := NewDeckhouseConfigValidator(globalHooksDir, modulesDir)

	// String manifest to unstructured.
	var m map[string]interface{}
	_ = yaml.Unmarshal([]byte(manifest), &m)
	obj := &unstructured.Unstructured{
		Object: m,
	}

	review := &model.AdmissionReview{
		Name:      obj.GetName(),
		Operation: model.OperationCreate,
	}

	return cfgValidator.Validate(context.TODO(), review, obj)
}

const validCfgVer1 = `
apiVersion: deckhouse.io/v1
kind: DeckhouseConfig
metadata:
  name: module-one
spec:
  configVersion: 1
  configValues:
    paramNum: 100
`

func TestValidateValidObjectVer1(t *testing.T) {
	g := NewWithT(t)

	res, err := runValidate(t, "testdata", validCfgVer1)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(res.Valid).Should(BeTrue(), "should convert to version 2 and validate successfully, got %+v", res)
	g.Expect(res.Warnings).Should(HaveLen(0))
}

const validCfgVer2 = `
apiVersion: deckhouse.io/v1
kind: DeckhouseConfig
metadata:
  name: module-one
spec:
  configVersion: 2
  configValues:
    paramStr: "100"
`

func TestValidateValidObjectVer2(t *testing.T) {
	g := NewWithT(t)

	res, err := runValidate(t, "testdata", validCfgVer2)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(res.Valid).Should(BeTrue(), "should be valid, got %+v", res)
	g.Expect(res.Warnings).Should(HaveLen(0))
}

const unknownCfg = `
apiVersion: deckhouse.io/v1
kind: DeckhouseConfig
metadata:
  name: module-three
spec:
  configVersion: 1
  configValues:
    param1: someText
`

func TestValidateUnknownConfig(t *testing.T) {
	g := NewWithT(t)

	res, err := runValidate(t, "testdata", unknownCfg)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(res.Valid).Should(BeTrue())
	g.Expect(res.Warnings).Should(HaveLen(1))
	g.Expect(res.Warnings[0]).Should(ContainSubstring("unknown"))
}

const invalidCfg = `
apiVersion: deckhouse.io/v1
kind: DeckhouseConfig
metadata:
  name: module-one
spec:
  configVersion: 1
  configValues:
    paramNum: 100
    unknown-param: someText
`

func TestValidateConfigWithExcessParams(t *testing.T) {
	g := NewWithT(t)

	res, err := runValidate(t, "testdata", invalidCfg)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(res.Valid).Should(BeFalse(), "should convert to version 2 and fail on forbidden property, got %+v", res)
	g.Expect(res.Message).Should(ContainSubstring("not valid"))
	g.Expect(res.Message).Should(ContainSubstring("forbidden property"))
}

const validGlobalCfg = `
apiVersion: deckhouse.io/v1
kind: DeckhouseConfig
metadata:
  name: global
spec:
  configVersion: 1
  configValues:
    globalParam: someText
`

func TestValidateValidGlobalConfig(t *testing.T) {
	g := NewWithT(t)

	res, err := runValidate(t, "testdata", validGlobalCfg)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(res.Valid).Should(BeTrue())
	g.Expect(res.Warnings).Should(HaveLen(0))
}

const invalidGlobalCfg = `
apiVersion: deckhouse.io/v1
kind: DeckhouseConfig
metadata:
  name: global
spec:
  configVersion: 1
  configValues:
    globalBadParam: someText
`

func TestValidateInvalidGlobalConfig(t *testing.T) {
	g := NewWithT(t)

	res, err := runValidate(t, "testdata", invalidGlobalCfg)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(res.Valid).Should(BeFalse())
	g.Expect(res.Message).Should(ContainSubstring("not valid"))
}

const noVerGlobalCfg = `
apiVersion: deckhouse.io/v1
kind: DeckhouseConfig
metadata:
  name: global
spec:
  configValues:
    globalParam: someText
`

func TestValidateGlobalConfigWithoutVersion(t *testing.T) {
	g := NewWithT(t)

	res, err := runValidate(t, "testdata", noVerGlobalCfg)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(res.Valid).Should(BeFalse())
	g.Expect(res.Message).Should(ContainSubstring("required"))
}
