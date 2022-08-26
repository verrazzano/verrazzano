// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"testing"

	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	clusterIP             = "1.2.3.4"
	promOperComponentName = "prometheus-operator"
	testPrometheusName    = "test-prometheus"
)

var promTestScheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(promTestScheme)
	_ = promoperapi.AddToScheme(promTestScheme)
}

// TestUpdatePrometheusAnnotations tests the UpdatePrometheusAnnotations function
// GIVEN the Prometheus CR and keycloak-http service exist
// WHEN  the UpdatePrometheusAnnotations function is called
// THEN  the Prometheus CR pod annotations are updated with the keycloak IP
func TestUpdatePrometheusAnnotations(t *testing.T) {
	asserts := assert.New(t)

	// create the k8s mock populated with resources
	k8sMock := createK8sMock()

	ctx := spi.NewFakeContext(k8sMock, &vzapi.Verrazzano{}, nil, false)
	err := UpdatePrometheusAnnotations(ctx, constants.VerrazzanoMonitoringNamespace, promOperComponentName)
	asserts.NoError(err)

	prom := &promoperapi.Prometheus{}
	err = k8sMock.Get(context.TODO(), types.NamespacedName{Namespace: constants.VerrazzanoMonitoringNamespace, Name: testPrometheusName}, prom)
	asserts.NoError(err)
	asserts.Equal(clusterIP+"/32", prom.Spec.PodMetadata.Annotations["traffic.sidecar.istio.io/includeOutboundIPRanges"])
}

// TestUpdatePrometheusAnnotationsErrorConditions tests the UpdatePrometheusAnnotations function error conditions
func TestUpdatePrometheusAnnotationsErrorConditions(t *testing.T) {
	asserts := assert.New(t)

	// GIVEN the Prometheus CRDs have not been installed
	// WHEN  the UpdatePrometheusAnnotations function is called
	// THEN  no error is returned

	// create the k8s mock without the Prometheus types
	schemeMissingPromTypes := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(schemeMissingPromTypes)

	k8sMock := fake.NewClientBuilder().WithScheme(schemeMissingPromTypes).Build()

	ctx := spi.NewFakeContext(k8sMock, &vzapi.Verrazzano{}, nil, false)
	err := UpdatePrometheusAnnotations(ctx, constants.VerrazzanoMonitoringNamespace, promOperComponentName)
	asserts.NoError(err)

	// GIVEN the keycloak-http service has not been created
	// WHEN  the UpdatePrometheusAnnotations function is called
	// THEN  no error is returned

	// create the k8s mock without the keycloak-http service
	k8sMock = fake.NewClientBuilder().WithScheme(promTestScheme).Build()

	ctx = spi.NewFakeContext(k8sMock, &vzapi.Verrazzano{}, nil, false)
	err = UpdatePrometheusAnnotations(ctx, constants.VerrazzanoMonitoringNamespace, promOperComponentName)
	asserts.NoError(err)
}

// createK8sMock creates the k8s mock populated with test resources
func createK8sMock() client.Client {
	return fake.NewClientBuilder().WithScheme(promTestScheme).WithObjects(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.KeycloakNamespace,
				Name:      keycloakHTTPService,
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: clusterIP,
			},
		},
		&promoperapi.Prometheus{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.VerrazzanoMonitoringNamespace,
				Name:      testPrometheusName,
				Labels: map[string]string{
					constants.VerrazzanoComponentLabelKey: promOperComponentName,
				},
			},
		}).Build()
}
