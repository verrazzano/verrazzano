// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"github.com/golang/mock/gomock"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime2 "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	containerizedWorkload := oamv1.ContainerizedWorkload{TypeMeta: metav1.TypeMeta{Kind: "ContainerizedWorkload", APIVersion: "core.oam.dev/v1alpha2"}}
	testDefaulter(t, "hello-conf.yaml", &containerizedWorkload, 0, 1, true)
	testDefaulter(t, "hello-conf_withTrait.yaml", &containerizedWorkload, 1, 2, true)
	testDefaulter(t, "hello-conf_withMetricsTrait.yaml", &containerizedWorkload, 2, 2, true)

	configMapWorkload := v1.ConfigMap{TypeMeta: metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"}}
	testDefaulter(t, "hello-conf.yaml", &configMapWorkload, 0, 0, false)
}

// TestMetricsTraitDefaulter_Cleanup tests cleaning up the default MetricsTrait on an appconfig
// GIVEN a AppConfigDefaulter and an appconfig
//  WHEN Cleanup is called with an appconfig
//  THEN Cleanup should run without error
func TestMetricsTraitDefaulter_Cleanup(t *testing.T) {
	testMetricsTraitDefaulterCleanup(t, "hello-conf.yaml", false)
	testMetricsTraitDefaulterCleanup(t, "hello-conf.yaml", true)
}

func testDefaulter(t *testing.T, configPath string, workloadObject runtime2.Object, initTraitsSize, expectedTraitsSize int, expectedMetricsTrait bool) {
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
	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: metricsTraitType}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, td *oamv1.TraitDefinition) error {
			td.Spec.AppliesToWorkloads = []string{
				"core.oam.dev/v1alpha2.ContainerizedWorkload",
				"oam.verrazzano.io/v1alpha1.VerrazzanoCoherenceWorkload",
				"oam.verrazzano.io/v1alpha1.VerrazzanoHelidonWorkload",
				"oam.verrazzano.io/v1alpha1.VerrazzanoWebLogicWorkload",
			}
			return nil
		})
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "default", Name: "hello-component"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, comp *oamv1.Component) error {
			b, err := json.Marshal(workloadObject)
			if err != nil {
				t.Fatalf("Error in json.Marshal %v", err)
			}
			comp.Spec.Workload = runtime.RawExtension{Raw: b, Object: workloadObject}
			return nil
		})
	defaulter := &MetricsTraitDefaulter{cli}
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
	assert.Equal(t, expectedMetricsTrait, foundMetricsTrait)
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
	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	defaulter := &MetricsTraitDefaulter{Client: cli}
	err = defaulter.Cleanup(appConfig, dryRun)
	if err != nil {
		t.Fatalf("Error in defaulter.Default %v", err)
	}
}
