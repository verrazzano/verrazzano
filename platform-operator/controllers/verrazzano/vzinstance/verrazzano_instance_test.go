// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzinstance

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

// TestGetInstanceInfo tests the GetInstanceInfo method
// GIVEN a request to GetInstanceInfo
// WHEN all system ingresses are present
// THEN the an instance info struct is returned with the expected URLs
func TestGetInstanceInfo(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	const dnsDomain = "myenv.testverrazzano.com"

	const keycloakURL = "keycloak." + dnsDomain
	const esURL = "elasticsearch." + dnsDomain
	const promURL = "prometheus." + dnsDomain
	const grafanaURL = "grafana." + dnsDomain
	const kibanaURL = "kibana." + dnsDomain
	const rancherURL = "rancher." + dnsDomain
	const consoleURL = "verrazzano." + dnsDomain

	// Expect a call to get the Verrazzano resource.
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ingressList *networkingv1.IngressList, opts ...client.ListOption) error {
			ingressList.Items = []networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "cattle-system", Name: "rancher"},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: rancherURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "keycloak", Name: "keycloak"},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: keycloakURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-os-ingest"},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: esURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-prometheus"},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: promURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-grafana"},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: grafanaURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-kibana"},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: kibanaURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: consoleURL},
						},
					},
				},
			}
			return nil
		})

	enabled := true
	vz := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				Console: &v1alpha1.ConsoleComponent{
					Enabled: &enabled,
				},
			},
		},
	}

	instanceInfo := GetInstanceInfo(spi.NewFakeContext(mock, vz, nil, false))
	mocker.Finish()
	assert.NotNil(t, instanceInfo)
	assert.Equal(t, "https://"+consoleURL, *instanceInfo.ConsoleURL)
	assert.Equal(t, "https://"+rancherURL, *instanceInfo.RancherURL)
	assert.Equal(t, "https://"+keycloakURL, *instanceInfo.KeyCloakURL)
	assert.Equal(t, "https://"+esURL, *instanceInfo.ElasticURL)
	assert.Equal(t, "https://"+grafanaURL, *instanceInfo.GrafanaURL)
	assert.Equal(t, "https://"+kibanaURL, *instanceInfo.KibanaURL)
	assert.Equal(t, "https://"+promURL, *instanceInfo.PrometheusURL)
}

// TestGetInstanceInfoManagedCluster tests GetInstanceInfo method
// GIVEN a request to GetInstanceInfo
// WHEN all some system ingresses are missing (e.g., Managed Cluster configuration)
// THEN the an instance info struct is returned with the expected URLs, and nil where ingresses are missing
func TestGetInstanceInfoManagedCluster(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	const dnsDomain = "myenv.testverrazzano.com"

	const keycloakURL = "keycloak." + dnsDomain
	const promURL = "prometheus." + dnsDomain
	const rancherURL = "rancher." + dnsDomain
	const consoleURL = "verrazzano." + dnsDomain

	// Expect a call to get the Verrazzano resource.
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ingressList *networkingv1.IngressList, opts ...client.ListOption) error {
			ingressList.Items = []networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "cattle-system", Name: "rancher"},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: rancherURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "keycloak", Name: "keycloak"},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: keycloakURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-prometheus"},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: promURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: consoleURL},
						},
					},
				},
			}
			return nil
		})

	instanceInfo := GetInstanceInfo(spi.NewFakeContext(mock, &v1alpha1.Verrazzano{}, nil, false))
	mocker.Finish()
	assert.NotNil(t, instanceInfo)
	assert.Equal(t, "https://"+consoleURL, *instanceInfo.ConsoleURL)
	assert.Equal(t, "https://"+rancherURL, *instanceInfo.RancherURL)
	assert.Equal(t, "https://"+keycloakURL, *instanceInfo.KeyCloakURL)
	assert.Nil(t, instanceInfo.ElasticURL)
	assert.Nil(t, instanceInfo.GrafanaURL)
	assert.Nil(t, instanceInfo.KibanaURL)
	assert.Equal(t, "https://"+promURL, *instanceInfo.PrometheusURL)
}

// TestGetInstanceInfoManagedCluster tests GetInstanceInfo method
// GIVEN a request to GetInstanceInfo
// WHEN an error is returned when listing the ingress resources
// THEN nil is returned
func TestGetInstanceInfoGetError(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the Verrazzano resource.
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ingressList *networkingv1.IngressList, opts ...client.ListOption) error {
			return fmt.Errorf("test error")
		})

	info := GetInstanceInfo(spi.NewFakeContext(mock, &v1alpha1.Verrazzano{}, nil, false))
	mocker.Finish()
	assert.Nil(t, info)
}

// TestGetInstanceInfoNoIngresses tests GetInstanceInfo method
// GIVEN a request to GetInstanceInfo
// WHEN all system ingresses are missing
// THEN the an instance info struct is returned with the expected URLs, and nil where ingresses are missing
func TestGetInstanceInfoNoIngresses(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the Verrazzano resource.
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ingressList *networkingv1.IngressList, opts ...client.ListOption) error {
			ingressList.Items = []networkingv1.Ingress{}
			return nil
		})

	enabled := false
	vz := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				Console: &v1alpha1.ConsoleComponent{
					Enabled: &enabled,
				},
			},
		},
	}

	instanceInfo := GetInstanceInfo(spi.NewFakeContext(mock, vz, nil, false))
	mocker.Finish()
	assert.Nil(t, instanceInfo)
}
