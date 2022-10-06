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
	"fmt"
	"strconv"
	"strings"

	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhvalidating "github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	d8config "github.com/deckhouse/deckhouse/go_lib/deckhouse-config"
	"github.com/deckhouse/deckhouse/go_lib/deckhouse-config/conversion"
	d8config_v1 "github.com/deckhouse/deckhouse/go_lib/deckhouse-config/v1"
)

type DeckhouseConfigValidator struct {
	modulesDir     string
	globalHooksDir string
	modulesMap     map[string]struct{}
}

func NewDeckhouseConfigValidator(globalHooksDir string, modulesDir string) *DeckhouseConfigValidator {
	return &DeckhouseConfigValidator{
		globalHooksDir: globalHooksDir,
		modulesDir:     modulesDir,
	}
}

func (c *DeckhouseConfigValidator) Validate(_ context.Context, review *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhvalidating.ValidatorResult, error) {
	if review.Operation == kwhmodel.OperationDelete && review.Name == "global" {
		return rejectResult("deleting DeckhouseConfig/global is not allowed")
	}

	untypedCfg, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("expect DeckhouseConfig as unstructured, got %T", obj)
	}

	if untypedCfg.GetKind() != "DeckhouseConfig" {
		return nil, fmt.Errorf("expect DeckhouseConfig, got %s", untypedCfg.GetKind())
	}

	var cfg d8config_v1.DeckhouseConfig
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(untypedCfg.UnstructuredContent(), &cfg)
	if err != nil {
		return nil, err
	}

	if !d8config.Service().PossibleNames().Has(cfg.Name) {
		return allowResult(fmt.Sprintf("module name '%s' is unknown for deckhouse", cfg.Name))
	}

	// Reject objects without version.
	if cfg.Spec.ConfigVersion == 0 {
		return rejectResult("spec.configVersion is required")
	}

	// Reject new objects with unknown version.
	if review.Operation == kwhmodel.OperationCreate {
		chain := conversion.Registry().Chain(cfg.GetName())
		if chain != nil && !chain.IsValidVersion(cfg.Spec.ConfigVersion) {
			supportedVersions := concatIntList(chain.VersionList())
			return rejectResult(fmt.Sprintf("spec.configVersion has invalid value. Supported versions: %s", supportedVersions))
		}
	}

	err = d8config.Service().ConfigValidator().ValidateConfig(&cfg)
	if err != nil {
		return rejectResult(fmt.Sprintf("validate: %v", err))
	}

	return allowResult("")
}

func allowResult(warnMsg string) (*kwhvalidating.ValidatorResult, error) {
	var warnings []string
	if warnMsg != "" {
		warnings = []string{warnMsg}
	}
	return &kwhvalidating.ValidatorResult{
		Valid:    true,
		Warnings: warnings,
	}, nil
}

func rejectResult(msg string) (*kwhvalidating.ValidatorResult, error) {
	return &kwhvalidating.ValidatorResult{
		Valid:   false,
		Message: msg,
	}, nil
}

func concatIntList(items []int) string {
	var buf strings.Builder
	for i, item := range items {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(strconv.Itoa(item))
	}
	return buf.String()
}
