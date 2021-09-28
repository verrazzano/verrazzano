// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus

import (
	"context"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	k8snet "k8s.io/api/networking/v1beta1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	crfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// Test_FixupPrometheusDeployment_WithPrometheusAndKeycloak tests updating Prometheus
// configuration when Keycloak is installed.
// GIVEN Prometheus and Keycloak are deployed
// WHEN FixupPrometheusDeployment is called
// THEN Prometheus configuration should be updated to avoid the Istio sidecar for calls to Keycloak
func Test_FixupPrometheusDeployment_WithPrometheusAndKeycloak(t *testing.T) {
	assert := asserts.New(t)
	ctx := context.TODO()
	log := zap.S()
	cli := crfake.NewFakeClientWithScheme(newScheme())

	promObj := newPrometheusDeployment()
	assert.NoError(cli.Create(ctx, promObj))
	kcObj := newKeycloakStatefulSet()
	assert.NoError(cli.Create(ctx, kcObj))
	kcSvc := newKeycloakHTTPService()
	assert.NoError(cli.Create(ctx, kcSvc))

	assert.NoError(FixupPrometheusDeployment(log, cli))

	promKey := client.ObjectKey{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-prometheus-0"}
	assert.NoError(cli.Get(ctx, promKey, promObj))
	assert.Equal("10.11.12.13/32", promObj.Spec.Template.Annotations["traffic.sidecar.istio.io/includeOutboundIPRanges"])
}

// Test_FixupPrometheusDeployment_WithPrometheusWithoutKeycloak tests updating Prometheus
// configuration when Keycloak is not installed.
// GIVEN Prometheus is deployed but Keycloak is not deployed
// WHEN FixupPrometheusDeployment is called
// THEN Prometheus configuration should be updated to avoid the Istio sidecar for all calls
func Test_FixupPrometheusDeployment_WithPrometheusWithoutKeycloak(t *testing.T) {
	assert := asserts.New(t)
	ctx := context.TODO()
	log := zap.S()
	cli := crfake.NewFakeClientWithScheme(newScheme())

	promObj := newPrometheusDeployment()
	assert.NoError(cli.Create(ctx, promObj))

	assert.NoError(FixupPrometheusDeployment(log, cli))

	promKey := client.ObjectKey{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-prometheus-0"}
	assert.NoError(cli.Get(ctx, promKey, promObj))
	assert.Equal("0.0.0.0/0", promObj.Spec.Template.Annotations["traffic.sidecar.istio.io/excludeOutboundIPRanges"])
}

// Test_FixupPrometheusDeployment_WithoutDeployment tests that nothing is done when
// Prometheus is not deployed in the cluster.
// GIVEN no Prometheus is deployed
// WHEN FixupPrometheusDeployment is called
// THEN No change error should be returned
func Test_FixupPrometheusDeployment_WithoutDeployment(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S()
	cli := crfake.NewFakeClientWithScheme(newScheme())
	assert.NoError(FixupPrometheusDeployment(log, cli))
}

// newPrometheusDeployment creates a new Prometheus Deployment
func newPrometheusDeployment() *k8sapps.Deployment {
	return &k8sapps.Deployment{
		ObjectMeta: k8smeta.ObjectMeta{
			Name:      "vmi-system-prometheus-0",
			Namespace: constants.VerrazzanoSystemNamespace,
		},
		Spec:   k8sapps.DeploymentSpec{},
		Status: k8sapps.DeploymentStatus{},
	}
}

// newKeycloakStatefulSet creates a new Keycloak StatefultSet
func newKeycloakStatefulSet() *k8sapps.StatefulSet {
	return &k8sapps.StatefulSet{
		ObjectMeta: k8smeta.ObjectMeta{
			Name:      "keycloak",
			Namespace: "keycloak",
		},
		Spec:   k8sapps.StatefulSetSpec{},
		Status: k8sapps.StatefulSetStatus{},
	}
}

// newKeycloakHTTPService creates a new Keycloak HTTP Service
func newKeycloakHTTPService() *k8score.Service {
	return &k8score.Service{
		ObjectMeta: k8smeta.ObjectMeta{
			Name:      "keycloak-http",
			Namespace: "keycloak",
		},
		Spec: k8score.ServiceSpec{
			ClusterIP: "10.11.12.13",
		},
		Status: k8score.ServiceStatus{},
	}
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	k8score.AddToScheme(scheme)
	k8sapps.AddToScheme(scheme)
	k8snet.AddToScheme(scheme)
	vzapi.AddToScheme(scheme)
	return scheme
}
