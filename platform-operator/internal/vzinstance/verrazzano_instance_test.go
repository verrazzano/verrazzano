// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzinstance

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

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
	const apiURL = "api." + dnsDomain

	// Expect a call to get the verrazzano resource.
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ingressList *extv1beta1.IngressList) error {
			ingressList.Items = []extv1beta1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "cattle-system", Name: "rancher"},
					Spec: extv1beta1.IngressSpec{
						Rules: []extv1beta1.IngressRule{
							{Host: rancherURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "keycloak", Name: "keycloak"},
					Spec: extv1beta1.IngressSpec{
						Rules: []extv1beta1.IngressRule{
							{Host: keycloakURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: systemNamespace, Name: "vmi-system-es-ingest"},
					Spec: extv1beta1.IngressSpec{
						Rules: []extv1beta1.IngressRule{
							{Host: esURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: systemNamespace, Name: "vmi-system-prometheus"},
					Spec: extv1beta1.IngressSpec{
						Rules: []extv1beta1.IngressRule{
							{Host: promURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: systemNamespace, Name: "vmi-system-grafana"},
					Spec: extv1beta1.IngressSpec{
						Rules: []extv1beta1.IngressRule{
							{Host: grafanaURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: systemNamespace, Name: "vmi-system-kibana"},
					Spec: extv1beta1.IngressSpec{
						Rules: []extv1beta1.IngressRule{
							{Host: kibanaURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: systemNamespace, Name: "verrazzano-console-ingress"},
					Spec: extv1beta1.IngressSpec{
						Rules: []extv1beta1.IngressRule{
							{Host: consoleURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: systemNamespace, Name: "vmi-system-api"},
					Spec: extv1beta1.IngressSpec{
						Rules: []extv1beta1.IngressRule{
							{Host: apiURL},
						},
					},
				},
			}
			return nil
		})

	instanceInfo := GetInstanceInfo(mock)
	mocker.Finish()
	assert.NotNil(t, instanceInfo)
	assert.Equal(t, "https://"+consoleURL, *instanceInfo.ConsoleURL)
	assert.Equal(t, "https://"+rancherURL, *instanceInfo.RancherURL)
	assert.Equal(t, "https://"+keycloakURL, *instanceInfo.KeyCloakURL)
	assert.Equal(t, "https://"+esURL, *instanceInfo.ElasticURL)
	assert.Equal(t, "https://"+grafanaURL, *instanceInfo.GrafanaURL)
	assert.Equal(t, "https://"+kibanaURL, *instanceInfo.KibanaURL)
	assert.Equal(t, "https://"+promURL, *instanceInfo.PrometheusURL)
	assert.Equal(t, "https://"+apiURL, *instanceInfo.SystemURL)
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
	const apiURL = "api." + dnsDomain

	// Expect a call to get the verrazzano resource.
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ingressList *extv1beta1.IngressList) error {
			ingressList.Items = []extv1beta1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "cattle-system", Name: "rancher"},
					Spec: extv1beta1.IngressSpec{
						Rules: []extv1beta1.IngressRule{
							{Host: rancherURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "keycloak", Name: "keycloak"},
					Spec: extv1beta1.IngressSpec{
						Rules: []extv1beta1.IngressRule{
							{Host: keycloakURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: systemNamespace, Name: "vmi-system-prometheus"},
					Spec: extv1beta1.IngressSpec{
						Rules: []extv1beta1.IngressRule{
							{Host: promURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: systemNamespace, Name: "verrazzano-console-ingress"},
					Spec: extv1beta1.IngressSpec{
						Rules: []extv1beta1.IngressRule{
							{Host: consoleURL},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: systemNamespace, Name: "vmi-system-api"},
					Spec: extv1beta1.IngressSpec{
						Rules: []extv1beta1.IngressRule{
							{Host: apiURL},
						},
					},
				},
			}
			return nil
		})

	instanceInfo := GetInstanceInfo(mock)
	mocker.Finish()
	assert.NotNil(t, instanceInfo)
	assert.Equal(t, "https://"+consoleURL, *instanceInfo.ConsoleURL)
	assert.Equal(t, "https://"+rancherURL, *instanceInfo.RancherURL)
	assert.Equal(t, "https://"+keycloakURL, *instanceInfo.KeyCloakURL)
	assert.Nil(t, instanceInfo.ElasticURL)
	assert.Nil(t, instanceInfo.GrafanaURL)
	assert.Nil(t, instanceInfo.KibanaURL)
	assert.Equal(t, "https://"+promURL, *instanceInfo.PrometheusURL)
	assert.Equal(t, "https://"+apiURL, *instanceInfo.SystemURL)
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

	// Expect a call to get the verrazzano resource.
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ingressList *extv1beta1.IngressList) error {
			return fmt.Errorf("Test error")
		})

	info := GetInstanceInfo(mock)
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

	// Expect a call to get the verrazzano resource.
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ingressList *extv1beta1.IngressList) error {
			ingressList.Items = []extv1beta1.Ingress{}
			return nil
		})

	instanceInfo := GetInstanceInfo(mock)
	mocker.Finish()
	assert.Nil(t, instanceInfo)
}
