// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"testing"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// TestMetricsTraitDefaulter_Default tests adding a default MetricsTrait to an appconfig
// GIVEN a AppConfigDefaulter and an appconfig
// WHEN Default is called with an appconfig
// THEN Default should add a default MetricsTrait to the appconfig if supported
func TestMetricsTraitDefaulter_Default(t *testing.T) {
	testDefaulter(t, "hello-comp.yaml", "hello-conf.yaml", "hello-workload.yaml", true,
		0, 1)
	testDefaulter(t, "hello-comp.yaml", "hello-conf_withTrait.yaml", "hello-workload.yaml", true,
		1, 2)
	testDefaulter(t, "hello-comp.yaml", "hello-conf_withMetricsTrait.yaml", "hello-workload.yaml", true,
		2, 2)
	testDefaulter(t, "bobs-component-no-metrics.yaml", "bobs-conf-no-metrics.yaml", "", false,
		0, 0)
}

// TestMetricsTraitDefaulter_Cleanup tests cleaning up the default MetricsTrait on an appconfig
// GIVEN a AppConfigDefaulter and an appconfig
// WHEN Cleanup is called with an appconfig
// THEN Cleanup should run without error
func TestMetricsTraitDefaulter_Cleanup(t *testing.T) {
	testMetricsTraitDefaulterCleanup(t, "hello-conf.yaml", false)
	testMetricsTraitDefaulterCleanup(t, "hello-conf.yaml", true)
	testMetricsTraitDefaulterCleanup(t, "bobs-conf-no-metrics.yaml", false)
	testMetricsTraitDefaulterCleanup(t, "bobs-conf-no-metrics.yaml", true)
}

func testDefaulter(t *testing.T, componentPath, configPath, workloadPath string, workloadSupported bool, initTraitsSize, expectedTraitsSize int) {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
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
	defaulter := &MetricsTraitDefaulter{Client: mock}

	// Expect a call to get the component.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1.Component) error {
			err = json.Unmarshal(readYaml2Json(t, componentPath), component)
			if err != nil {
				t.Fatalf("Error in unmarshalling component %v", err)
			}
			return nil
		}).AnyTimes()

	err = defaulter.Default(appConfig, false, zap.S())
	if err != nil {
		t.Fatalf("Error in defaulter.Default %v", err)
	}
	assert.Equal(t, expectedTraitsSize, len(appConfig.Spec.Components[0].Traits))
	foundMetricsTrait := false
	for _, trait := range appConfig.Spec.Components[0].Traits {
		var rawTrait map[string]interface{}
		_ = json.Unmarshal(trait.Trait.Raw, &rawTrait)
		if rawTrait["apiVersion"] == apiVersion && rawTrait["kind"] == v1alpha1.MetricsTraitKind {
			foundMetricsTrait = true
		}
	}
	assert.Equal(t, foundMetricsTrait, workloadSupported)
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
	err = defaulter.Cleanup(appConfig, dryRun, zap.S())
	if err != nil {
		t.Fatalf("Error in defaulter.Default %v", err)
	}
}
