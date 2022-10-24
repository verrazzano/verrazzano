// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package configmaps

import (
	"context"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/status"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestConfigMapReconciler tests Reconciler method for the following use case
// GIVEN a request to reconcile a ConfigMap
// WHEN the ConfigMap is referenced in the Verrazzano CR under a component and is also present the CR namespace
// THEN the ReconcilingGeneration of the target component is set to 1
func TestConfigMapReconciler(t *testing.T) {
	asserts := assert.New(t)
	cm := testConfigMap
	cm.Finalizers = append(cm.Finalizers, constants.OverridesFinalizer)
	cli := fake.NewClientBuilder().WithObjects(&testVZ, &cm).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request0 := newRequest(testNS, testCMName)
	reconciler := newConfigMapReconciler(cli)
	res0, err0 := reconciler.Reconcile(context.TODO(), request0)

	asserts.NoError(err0)
	asserts.Equal(false, res0.Requeue)

	vz := vzapi.Verrazzano{}
	err := cli.Get(context.TODO(), types.NamespacedName{Namespace: testNS, Name: testVZName}, &vz)
	asserts.NoError(err)
	asserts.Equal(int64(1), vz.Status.Components["prometheus-operator"].ReconcilingGeneration)
}

// TestAddFinalizer tests the Reconcile loop for the following use case
// GIVEN a request to reconcile a ConfigMap that qualifies as an override
// WHEN the ConfigMap is found without the overrides finalizer
// THEN the overrides finalizer is added and we requeue without an error
func TestAddFinalizer(t *testing.T) {
	asserts := assert.New(t)
	cli := fake.NewClientBuilder().WithObjects(&testVZ, &testConfigMap).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request0 := newRequest(testNS, testCMName)
	reconciler := newConfigMapReconciler(cli)
	res0, err0 := reconciler.Reconcile(context.TODO(), request0)

	asserts.NoError(err0)
	asserts.Equal(true, res0.Requeue)

	cm := corev1.ConfigMap{}
	err := cli.Get(context.TODO(), types.NamespacedName{Namespace: testNS, Name: testCMName}, &cm)
	asserts.NoError(err)
	asserts.True(controllerutil.ContainsFinalizer(&cm, constants.OverridesFinalizer))
}

// TestOtherFinalizers tests the Reconcile loop for the following use case
// GIVEN a request to reconcile a ConfigMap that qualifies as an override resource and is scheduled for deletion
// WHEN the ConfigMap is found with finalizers but the override finalizer is missing
// THEN without updating the Verrazzano CR a requeue request is returned without an error
func TestOtherFinalizers(t *testing.T) {
	asserts := assert.New(t)
	cm := testConfigMap
	cm.Finalizers = append(cm.Finalizers, "test")
	cm.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	cli := fake.NewClientBuilder().WithObjects(&testVZ, &cm).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request0 := newRequest(testNS, testCMName)
	reconciler := newConfigMapReconciler(cli)
	res0, err0 := reconciler.Reconcile(context.TODO(), request0)

	asserts.NoError(err0)
	asserts.Equal(true, res0.Requeue)

	vz := &vzapi.Verrazzano{}
	err1 := cli.Get(context.TODO(), types.NamespacedName{Namespace: testNS, Name: testVZName}, vz)
	asserts.NoError(err1)
	asserts.NotEqual(int64(1), vz.Status.Components["prometheus-operator"].ReconcilingGeneration)
}

// TestConfigMapNotFound tests the Reconcile method for the following use cases
// GIVEN requests to reconcile a ConfigMap
// WHEN the ConfigMap is not found in the cluster
// THEN Verrazzano is updated if it's listed as an override, otherwise the request is ignored
func TestConfigMapNotFound(t *testing.T) {
	tests := []struct {
		nsn types.NamespacedName
	}{
		{
			nsn: types.NamespacedName{Namespace: testNS, Name: testCMName},
		},
		{
			nsn: types.NamespacedName{Namespace: testNS, Name: "test"},
		},
	}

	for i, tt := range tests {
		asserts := assert.New(t)
		cli := fake.NewClientBuilder().WithObjects(&testVZ).WithScheme(newScheme()).Build()

		config.TestProfilesDir = "../../manifests/profiles"
		defer func() { config.TestProfilesDir = "" }()

		request0 := newRequest(tt.nsn.Namespace, tt.nsn.Name)
		reconciler := newConfigMapReconciler(cli)
		res0, err0 := reconciler.Reconcile(context.TODO(), request0)

		asserts.NoError(err0)
		asserts.Equal(false, res0.Requeue)

		vz := &vzapi.Verrazzano{}
		err1 := cli.Get(context.TODO(), types.NamespacedName{Namespace: testNS, Name: testVZName}, vz)
		asserts.NoError(err1)
		if i == 0 {
			asserts.Equal(int64(1), vz.Status.Components["prometheus-operator"].ReconcilingGeneration)
		} else {
			asserts.NotEqual(int64(1), vz.Status.Components["prometheus-operator"].ReconcilingGeneration)
		}
	}

}

// TestDeletion tests the Reconcile loop for the following use case
// GIVEN a request to reconcile a ConfigMap that qualifies as an override
// WHEN we find that it is scheduled for deletion and contains overrides finalizer
// THEN the override finalizer is removed from the ConfigMap and Verrazzano CR is updated and request is returned without an error
func TestDeletion(t *testing.T) {
	asserts := assert.New(t)
	cm := testConfigMap
	cm.Finalizers = append(cm.Finalizers, constants.OverridesFinalizer)
	cm.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	cli := fake.NewClientBuilder().WithObjects(&testVZ, &cm).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request0 := newRequest(testNS, testCMName)
	reconciler := newConfigMapReconciler(cli)
	res0, err0 := reconciler.Reconcile(context.TODO(), request0)

	asserts.NoError(err0)
	asserts.Equal(false, res0.Requeue)

	cm1 := &corev1.ConfigMap{}
	err1 := cli.Get(context.TODO(), types.NamespacedName{Namespace: testNS, Name: testCMName}, cm1)
	asserts.True(errors.IsNotFound(err1))

	vz := &vzapi.Verrazzano{}
	err2 := cli.Get(context.TODO(), types.NamespacedName{Namespace: testNS, Name: testVZName}, vz)
	asserts.NoError(err2)
	asserts.Equal(int64(1), vz.Status.Components["prometheus-operator"].ReconcilingGeneration)
}

// TestConfigMapRequeue the Reconciler method for the following use case
// GIVEN a request to reconcile a ConfigMap that qualifies as an override
// WHEN the status of the Verrazzano CR is found without the Component Status details
// THEN a requeue request is returned with an error
func TestConfigMapRequeue(t *testing.T) {
	asserts := assert.New(t)
	vz := testVZ
	vz.Status.Components = nil
	asserts.Nil(vz.Status.Components)
	cm := testConfigMap
	cm.Finalizers = append(cm.Finalizers, constants.OverridesFinalizer)
	cli := fake.NewClientBuilder().WithObjects(&vz, &cm).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request0 := newRequest(testNS, testCMName)
	reconciler := newConfigMapReconciler(cli)
	res0, err0 := reconciler.Reconcile(context.TODO(), request0)

	asserts.Error(err0)
	asserts.Contains(err0.Error(), "Components not initialized")
	asserts.Equal(true, res0.Requeue)
}

// TestConfigMapCall tests the reconcileInstallOverrideConfigMap for the following use case
// GIVEN a request to reconcile a ConfigMap
// WHEN the request namespace matches the Verrazzano CR namespace
// THEN expect a call to get the ConfigMap
func TestConfigMapCall(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	expectGetConfigMapExists(mock, &testConfigMap, testNS, testCMName)

	request := newRequest(testNS, testCMName)
	reconciler := newConfigMapReconciler(mock)
	result, err := reconciler.reconcileInstallOverrideConfigMap(context.TODO(), request, &testVZ)
	asserts.NoError(err)
	mocker.Finish()
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestOtherNS tests the reconcileInstallOverrideConfigMap for the following use case
// GIVEN a request to reconcile a ConfigMap
// WHEN the request namespace does not match with the CR namespace
// THEN the request is ignored
func TestOtherNS(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Do not expect a call to get the ConfigMap if it's a different namespace
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).MaxTimes(0)

	request := newRequest("test0", "test1")
	reconciler := newConfigMapReconciler(mock)
	result, err := reconciler.reconcileInstallOverrideConfigMap(context.TODO(), request, &testVZ)
	asserts.NoError(err)
	mocker.Finish()
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)

}

// mock client request to get the configmap
func expectGetConfigMapExists(mock *mocks.MockClient, cmToUse *corev1.ConfigMap, namespace string, name string) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, cm *corev1.ConfigMap) error {
			return nil
		})
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	return scheme
}

// newRequest creates a new reconciler request for testing
// namespace - The namespace to use in the request
// name - The name to use in the request
func newRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name}}
}

// newConfigMapReconciler creates a new reconciler for testing
func newConfigMapReconciler(c client.Client) VerrazzanoConfigMapsReconciler {
	vzLog := vzlog.DefaultLogger()
	scheme := newScheme()
	reconciler := VerrazzanoConfigMapsReconciler{
		Client:        c,
		Scheme:        scheme,
		log:           vzLog,
		StatusUpdater: &vzstatus.FakeVerrazzanoStatusUpdater{Client: c},
	}
	return reconciler
}
