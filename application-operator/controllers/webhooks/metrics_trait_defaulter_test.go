// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"encoding/json"
	"testing"

	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// TestMetricsTraitDefaulter_Default tests adding a default MetricsTrait to an appconfig
// GIVEN a AppConfigDefaulter and an appconfig
//  WHEN Default is called with an appconfig
//  THEN Default should add a default MetricsTrait to the appconfig
func TestMetricsTraitDefaulter_Default(t *testing.T) {
	testDefaulter(t, "hello-comp.yaml", "hello-conf.yaml",
		0, 1)
	testDefaulter(t, "hello-comp.yaml", "hello-conf_withTrait.yaml",
		1, 2)
	testDefaulter(t, "hello-comp.yaml", "hello-conf_withMetricsTrait.yaml",
		2, 2)
}

// TestMetricsTraitDefaulter_Cleanup tests cleaning up the default MetricsTrait on an appconfig
// GIVEN a AppConfigDefaulter and an appconfig
//  WHEN Cleanup is called with an appconfig
//  THEN Cleanup should run without error
func TestMetricsTraitDefaulter_Cleanup(t *testing.T) {
	testMetricsTraitDefaulterCleanup(t, "hello-conf.yaml", false)
	testMetricsTraitDefaulterCleanup(t, "hello-conf.yaml", true)
}

func testDefaulter(t *testing.T, componentPath, configPath string, initTraitsSize, expectedTraitsSize int) {
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: readYaml2Json(t, configPath)}
	decoder := decoder()
	appConfig := &oamv1.ApplicationConfiguration{}
	err := decoder.Decode(req, appConfig)
	if err != nil {
		t.Fatalf("Error in decoder.Decode %v", err)
	}
	assert.Equal(t, 1, len(appConfig.Spec.Components))
	assert.Equal(t, initTraitsSize, len(appConfig.Spec.Components[0].Traits))
	defaulter := &MetricsTraitDefaulter{}
	err = defaulter.Default(appConfig, false)
	if err != nil {
		t.Fatalf("Error in defaulter.Default %v", err)
	}
	assert.Equal(t, expectedTraitsSize, len(appConfig.Spec.Components[0].Traits))
	foundMetricsTrait := false
	for _, trait := range appConfig.Spec.Components[0].Traits {
		var rawTrait map[string]interface{}
		json.Unmarshal(trait.Trait.Raw, &rawTrait)
		if rawTrait["apiVersion"] == apiVersion && rawTrait["kind"] == v1alpha1.MetricsTraitKind {
			foundMetricsTrait = true
		}
	}
	assert.True(t, foundMetricsTrait)
}

func testMetricsTraitDefaulterCleanup(t *testing.T, configPath string, dryRun bool) {
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: readYaml2Json(t, configPath)}
	decoder := decoder()
	appConfig := &oamv1.ApplicationConfiguration{}
	err := decoder.Decode(req, appConfig)
	if err != nil {
		t.Fatalf("Error in decoder.Decode %v", err)
	}
	assert.Equal(t, 1, len(appConfig.Spec.Components))
	defaulter := &MetricsTraitDefaulter{}
	err = defaulter.Cleanup(appConfig, dryRun)
	if err != nil {
		t.Fatalf("Error in defaulter.Default %v", err)
	}
}
