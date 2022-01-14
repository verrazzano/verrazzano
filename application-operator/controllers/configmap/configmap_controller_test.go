// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package configmap

import (
	"context"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"testing"
)

// TestReconcilerSetupWithManager test the creation of the ConfigMap reconciler.
// GIVEN a controller implementation
// WHEN the controller is created
// THEN verify no error is returned
func TestReconcilerSetupWithManager(t *testing.T) {
	assert := asserts.New(t)

	scheme := newScheme()
	vzapi.AddToScheme(scheme)
	client := fake.NewFakeClientWithScheme(scheme)
	reconciler := newReconciler(client)

	mocker := gomock.NewController(t)
	manager := mocks.NewMockManager(mocker)
	manager.EXPECT().GetConfig().Return(&rest.Config{}).AnyTimes()
	manager.EXPECT().GetScheme().Return(scheme).AnyTimes()
	manager.EXPECT().GetLogger().Return(log.NullLogger{}).AnyTimes()
	manager.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	manager.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()

	err := reconciler.SetupWithManager(manager)
	assert.NoError(err, "Expected no error when setting up reconciler")
	mocker.Finish()
}

// TestResetMutatingWebhookConfiguration tests resetting the workload resources to the default
// GIVEN the ConfigMap is being deleted
// WHEN the function is called with the webhook
// THEN the webhook is reset to the default list
func TestResetMutatingWebhookConfiguration(t *testing.T) {
	assert := asserts.New(t)

	scheme := newScheme()
	vzapi.AddToScheme(scheme)
	client := fake.NewFakeClientWithScheme(scheme)
	reconciler := newReconciler(client)

	localMWC := testMWC.DeepCopy()
	localMWC.Webhooks[0].Rules[0].Resources = append(defaultResourceList, testWorkload1, testWorkload2)

	reconciler.resetMutatingWebhookConfiguration(localMWC)
	assert.Equal(defaultResourceList, localMWC.Webhooks[0].Rules[0].Resources)
}

// TestUpdateMutatingWebhookConfiguration tests updating the workload resources to the ConfigMap values
// GIVEN the ConfigMap is being updated
// WHEN the function is called with the webhook and the ConfigMap
// THEN the webhook is updated to the new list
func TestUpdateMutatingWebhookConfiguration(t *testing.T) {
	assert := asserts.New(t)

	scheme := newScheme()
	vzapi.AddToScheme(scheme)
	client := fake.NewFakeClientWithScheme(scheme)
	reconciler := newReconciler(client)

	localMWC := testMWC.DeepCopy()

	reconciler.updateMutatingWebhookConfiguration(&testConfigMap, localMWC)
	assert.Equal(append(defaultResourceList, strings.ToLower(testWorkload1), strings.ToLower(testWorkload2)), localMWC.Webhooks[0].Rules[0].Resources)
}

// TestReconcileConfigMapCreateOrUpdate tests the reconciling of a ConfigMap when creating or updating
// GIVEN the ConfigMap is being updated
// WHEN the function is called with the webhook and the ConfigMap
// THEN the webhook is updated to the new list
func TestReconcileConfigMapCreateOrUpdate(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	localMWC := testMWC.DeepCopy()
	localCM := testConfigMap.DeepCopy()

	expectConfigMap(mock, localCM)
	expectMWC(mock, localMWC)

	_, err := reconciler.reconcileConfigMapCreateOrUpdate(context.TODO(), &testConfigMap, localMWC)
	assert.NoError(err, "Expected no error reconciling the create or update of the ConfigMap")
	assert.Equal(append(defaultResourceList, strings.ToLower(testWorkload1), strings.ToLower(testWorkload2)), localMWC.Webhooks[0].Rules[0].Resources)
}

// TestReconcileConfigMapDelete tests the reconciling of a ConfigMap when deleting
// GIVEN the ConfigMap is being deleted
// WHEN the function is called with the webhook and the ConfigMap
// THEN the webhook is changed to the default list
func TestReconcileConfigMapDelete(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	localMWC := testMWC.DeepCopy()
	localCM := testConfigMap.DeepCopy()

	expectConfigMap(mock, localCM)
	expectMWC(mock, localMWC)

	_, err := reconciler.reconcileConfigMapDelete(context.TODO(), &testConfigMap, localMWC)
	assert.NoError(err, "Expected no error reconciling the create or update of the ConfigMap")
	assert.Equal(defaultResourceList, localMWC.Webhooks[0].Rules[0].Resources)
}

// TestReconcile tests the reconciling of a ConfigMap
// GIVEN the ConfigMap is changed
// WHEN the function is called with a request
// THEN the webhook resource list is updated
func TestReconcile(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	reconciler := newReconciler(mock)

	localMWC := testMWC.DeepCopy()
	localCM := testConfigMap.DeepCopy()

	expectReconcileGet(mock, localMWC, localCM)
	expectConfigMap(mock, localCM)
	expectMWC(mock, localMWC)

	_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: testCMName}})
	assert.NoError(err, "Expected no error reconciling the create or update of the ConfigMap")
	assert.Equal(append(defaultResourceList, strings.ToLower(testWorkload1), strings.ToLower(testWorkload2)), localMWC.Webhooks[0].Rules[0].Resources)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	//_ = clientgoscheme.AddToScheme(scheme)
	k8sapps.AddToScheme(scheme)
	//	vzapi.AddToScheme(scheme)
	k8score.AddToScheme(scheme)
	//	certapiv1alpha2.AddToScheme(scheme)
	k8net.AddToScheme(scheme)
	return scheme
}

// newReconciler creates a new reconciler for testing
// c - The Kerberos client to inject into the reconciler
func newReconciler(c client.Client) Reconciler {
	log := ctrl.Log.WithName("test")
	scheme := newScheme()
	reconciler := Reconciler{
		Client: c,
		Log:    log,
		Scheme: scheme,
	}
	return reconciler
}

// expectConfigMap creates the mock calls for controllerutil.CreateOrUpdate for the ConfigMap
func expectConfigMap(mock *mocks.MockClient, configMap *k8score.ConfigMap) {
	mock.EXPECT().Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: testCMName}, gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, cm *k8score.ConfigMap) error {
			cm.Data = configMap.Data
			return nil
		})

	mock.EXPECT().Update(gomock.Any(), gomock.Not(gomock.Nil())).Return(nil)
}

// expectMWC creates the mock calls for controllerutil.CreateOrUpdate for the MutatingWebhookConfiguration
func expectMWC(mock *mocks.MockClient, configuration *admissionv1.MutatingWebhookConfiguration) {
	mock.EXPECT().Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: mutatingWebhookConfigName}, gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, mwc *admissionv1.MutatingWebhookConfiguration) error {
			mwc.Webhooks = configuration.Webhooks
			return nil
		})

	mock.EXPECT().Update(gomock.Any(), gomock.Not(gomock.Nil())).Return(nil)
}

// expectReconcileGet creates the mocks for the retrieval of the ConfigMap and the MutatingWebhookConfiguration
func expectReconcileGet(mock *mocks.MockClient, configuration *admissionv1.MutatingWebhookConfiguration, configMap *k8score.ConfigMap) {
	mock.EXPECT().Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: mutatingWebhookConfigName}, gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, mwc *admissionv1.MutatingWebhookConfiguration) error {
			mwc.ObjectMeta = configuration.ObjectMeta
			mwc.Webhooks = configuration.Webhooks
			return nil
		})
	mock.EXPECT().Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: testCMName}, gomock.Not(gomock.Nil())).DoAndReturn(
		func(ctx context.Context, key client.ObjectKey, cm *k8score.ConfigMap) error {
			cm.ObjectMeta = configMap.ObjectMeta
			cm.Data = configMap.Data
			return nil
		})
}
