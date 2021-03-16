// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	wls "github.com/verrazzano/verrazzano/application-operator/apis/weblogic/v8"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testResourceName = "testName"
)

var mockFluentd *MockFluentdManager

// TestApply tests the association of a WLS domain and a logging scope
// GIVEN a logging scope and a WLS Domain which hasn't previously been associated with the scope
// WHEN apply is called with the scope and WLS Domain information
// THEN ensure that the WLS handler makes the expected invocations on the FLuentdManager to create the
//      FLUENTD resources for the WLS Domain
func TestApply(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	mockFluentd = NewMockFluentdManager(mocker)

	wlsHandler := wlsHandler{Log: ctrl.Log, Client: mockClient}
	existingFluentdCreateFunc := getFluentdManager
	getFluentdManager = getFluentdMock
	defer func() { getFluentdManager = existingFluentdCreateFunc }()

	resource := createTestResourceRelation()
	scope := createTestLoggingScope(true)

	wlsDomain := createWlsDomain(resource)
	fluentdPod := toFluentdPod(wlsDomain.Spec.ServerPod, resource, BuildWLSLogPath(resource.Name))

	mockClient.EXPECT().
		Get(context.Background(), types.NamespacedName{Name: testResourceName, Namespace: testNamespace}, &wlsDomain).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, domain *wls.Domain) error {
			domain.Spec.ServerPod = *createTestWlsServerPod()
			return nil
		})
	mockClient.EXPECT().
		Update(context.Background(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, domain *wls.Domain) error {
			// update the FLUENTD pod with the information that was returned from the FLUENTD call
			updateFluentdPodForApply(fluentdPod)
			// ensure that the below objects slices were updated to reflect the information returned by fluentd
			asserts.Equal(t, fluentdPod.Containers, domain.Spec.ServerPod.Containers)
			asserts.Equal(t, fluentdPod.Volumes, domain.Spec.ServerPod.Volumes)
			asserts.Equal(t, fluentdPod.VolumeMounts, domain.Spec.ServerPod.VolumeMounts)
			return nil
		})
	mockFluentd.EXPECT().
		Apply(scope, resource, fluentdPod).
		DoAndReturn(func(scope *vzapi.LoggingScope, resource vzapi.QualifiedResourceRelation, fluentdPod *FluentdPod) (bool, error) {
			updateFluentdPodForApply(fluentdPod)
			return true, nil
		})

	// invoke method being tested
	_, err := wlsHandler.Apply(context.Background(), resource, scope)
	asserts.Nil(t, err)

	mocker.Finish()
}

// TestRemove tests the disassociation of a WLS domain from a logging scope
// GIVEN A WLS Domain which was previously associated, but is no longer associated with a logging scope
// WHEN apply is called with the scope and WLS Domain information
// THEN ensure that the WLS handler makes the expected invocations on the FluentdManager to remove the
//      FLUENTD resources for the WLS Domain
func TestRemove(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	mockFluentd = NewMockFluentdManager(mocker)

	wlsHandler := wlsHandler{Log: ctrl.Log, Client: mockClient}
	existingFluentdCreateFunc := getFluentdManager
	getFluentdManager = getFluentdMock
	defer func() { getFluentdManager = existingFluentdCreateFunc }()

	resource := createTestResourceRelation()
	scope := createTestLoggingScope(true)

	wlsDomain := createWlsDomain(resource)
	fluentdPod := toFluentdPod(wlsDomain.Spec.ServerPod, resource, "")
	// populate pod
	updateFluentdPodForApply(fluentdPod)

	mockClient.EXPECT().
		Get(context.Background(), types.NamespacedName{Name: testResourceName, Namespace: testNamespace}, &wlsDomain).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, domain *wls.Domain) error {
			domain.Spec.ServerPod = *createTestWlsServerPod()
			updateServerPod(&domain.Spec.ServerPod)
			return nil
		})
	mockClient.EXPECT().
		Update(context.Background(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, domain *wls.Domain) error {
			// update the FLUENTD pod with the information that was returned from the FLUENTD call
			updateFluentdPodForApply(fluentdPod)
			// ensure that the below objects slices were updated to reflect the information returned by FLUENTD
			asserts.Len(t, domain.Spec.ServerPod.Containers, 0)
			asserts.Len(t, domain.Spec.ServerPod.Volumes, 0)
			asserts.Len(t, domain.Spec.ServerPod.VolumeMounts, 0)
			return nil
		})
	mockFluentd.EXPECT().
		Remove(scope, resource, fluentdPod).
		DoAndReturn(func(scope *vzapi.LoggingScope, resource vzapi.QualifiedResourceRelation, fluentdPod *FluentdPod) (bool, error) {
			updateFluentdPodForRemove(fluentdPod)
			return false, nil
		})

	// invoke method being tested
	deleteValidated, err := wlsHandler.Remove(context.Background(), resource, scope)
	asserts.Nil(t, err)
	asserts.False(t, deleteValidated)

	mocker.Finish()
}

// createTestResourceRelation creates a new test QualifiedResourceRelation
func createTestResourceRelation() vzapi.QualifiedResourceRelation {
	resource := vzapi.QualifiedResourceRelation{
		APIVersion: testAPIVersion,
		Kind:       wlsDomainKind,
		Namespace:  testNamespace,
		Name:       testResourceName,
		Role:       "",
	}

	return resource
}

// createTestWlsServerPod creates a new test WLS ServerPod
func createTestWlsServerPod() *wls.ServerPod {
	serverPod := wls.ServerPod{}

	return &serverPod
}

// getFluentdMock returns a mock FLUENTD instance
func getFluentdMock(ctx context.Context, log logr.Logger, client k8sclient.Client) FluentdManager {
	return mockFluentd
}

// updateFluentdPodForApply adds container, volume and volume mounts to a FluentdPod
func updateFluentdPodForApply(fluentdPod *FluentdPod) {
	fluentdPod.Containers = append(fluentdPod.Containers, v1.Container{})
	fluentdPod.Volumes = append(fluentdPod.Volumes, v1.Volume{})
	fluentdPod.VolumeMounts = append(fluentdPod.VolumeMounts, v1.VolumeMount{})
}

// updateFluentdPodForRemove removes container, volume and volume mounts from a FluentdPod
func updateFluentdPodForRemove(fluentdPod *FluentdPod) {
	fluentdPod.Containers = nil
	fluentdPod.Volumes = nil
	fluentdPod.VolumeMounts = nil
}

// updateServerPod updates a ServerPod with container, volume and volume mounts
func updateServerPod(serverPod *wls.ServerPod) {
	serverPod.Containers = append(serverPod.Containers, v1.Container{})
	serverPod.Volumes = append(serverPod.Volumes, v1.Volume{})
	serverPod.VolumeMounts = append(serverPod.VolumeMounts, v1.VolumeMount{})
}
