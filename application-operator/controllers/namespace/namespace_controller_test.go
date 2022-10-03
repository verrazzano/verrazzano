// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package namespace

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var testScheme = newScheme()
var logger = vzlog.DefaultLogger()

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	return scheme
}

// newTestController - test helper to boostrap a NamespaceController for test purposes
func newTestController(c client.Client) (*NamespaceController, error) {
	mgr := fakeManager{
		Client: c,
		scheme: testScheme,
	}
	return NewNamespaceController(mgr, zap.S())
}

// TestReconcileNamespaceUpdate tests the Reconcile method for the following use case
// GIVEN a request to Reconcile a Namespace resource
// WHEN the namespace has the expected annotation
// THEN ensure that no error is returned and the result does not indicate a requeue
func TestReconcileNamespaceUpdate(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// Expect a call to get the namespace
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			ns.Name = "myns"
			ns.Annotations = map[string]string{
				constants.OCILoggingIDAnnotation: "myocid",
			}
			ns.Finalizers = []string{"someFinalizer"}
			return nil
		})

	// Expect a call to update the namespace that succeeds
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.UpdateOption) error {
			return nil
		})

	// Expect calls to restart Fluentd
	mockFluentdRestart(mock, asserts)

	addNamespaceLoggingFunc = func(_ context.Context, _ client.Client, _ string, _ string) (bool, error) {
		return true, nil
	}
	defer func() { addNamespaceLoggingFunc = addNamespaceLogging }()

	nc, err := newTestController(mock)
	asserts.NoError(err)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "myns"},
	}
	result, err := nc.Reconcile(context.TODO(), req)

	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(ctrl.Result{}, result)
}

// TestReconcileNamespaceNotFound tests the Reconcile method for the following use case
// GIVEN a request to Reconcile a Namespace resource
// WHEN the namespace can not be found
// THEN ensure that no error is returned and the result does not indicate a requeue
func TestReconcileNamespaceNotFound(t *testing.T) {
	runTestReconcileGetError(t, k8serrors.NewNotFound(schema.ParseGroupResource("Namespace"), "myns"), ctrl.Result{})
}

// TestReconcileNamespaceGetError tests the Reconcile method for the following use case
// GIVEN a request to Reconcile a Namespace resource
// WHEN the client Get() operation returns an error other than IsNotFound
// THEN ensure that the unexpected error is returned and the result does not indicate a requeue (controllerruntime does this)
func TestReconcileNamespaceGetError(t *testing.T) {
	err := fmt.Errorf("some other error getting namespace")
	runTestReconcileGetError(t, err, ctrl.Result{Requeue: true})
}

// runTestReconcileGetError - Common test code for the namespace Get() error cases
func runTestReconcileGetError(t *testing.T, returnErr error, expectedResult ctrl.Result) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// Expect a call to get the namespace
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			return returnErr
		})

	// Expect no call to update the namespace
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	nc, err := newTestController(mock)
	asserts.NoError(err)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "myns"},
	}
	result, err := nc.Reconcile(context.TODO(), req)

	mocker.Finish()
	asserts.Nil(err)
	asserts.Equal(expectedResult.Requeue, result.Requeue)
	if result.Requeue {
		asserts.Greater(result.RequeueAfter.Seconds(), time.Duration(0).Seconds())
	}
}

// TestReconcileNamespaceDeleted tests the Reconcile method for the following use case
// GIVEN a request to Reconcile a Namespace resource
// WHEN the namespace DeletionTimestamp has been set (namespace is in the process of being deleted)
// THEN ensure that the namespace finalizer is deleted and no error or requeue result are returned
func TestReconcileNamespaceDeleted(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// Expect a call to get the namespace
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			ns.Name = "myns"
			ns.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			ns.Annotations = map[string]string{
				constants.OCILoggingIDAnnotation: "myocid",
			}
			ns.Finalizers = []string{"someFinalizer", namespaceControllerFinalizer}
			return nil
		})

	// Expect calls to restart Fluentd
	mockFluentdRestart(mock, asserts)

	// Expect a call to update the namespace that succeeds
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.UpdateOption) error {
			asserts.NotContainsf(ns.Finalizers, namespaceControllerFinalizer, "Finalizer not removed: ", ns.Finalizers)
			return nil
		})

	removeNamespaceLoggingFunc = func(_ context.Context, _ client.Client, _ string) (bool, error) {
		return true, nil
	}
	defer func() { removeNamespaceLoggingFunc = removeNamespaceLogging }()

	nc, err := newTestController(mock)
	asserts.NoError(err)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "myns"},
	}
	result, err := nc.Reconcile(context.TODO(), req)

	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(ctrl.Result{}, result)
}

// TestReconcileNamespaceDeletedErrorOnUpdate tests the Reconcile method for the following use case
// GIVEN a request to Reconcile a Namespace resource
// WHEN the namespace DeletionTimestamp has been set (namespace deleted) and the remove OCI logging integration returns an error
// THEN ensure than an error and a requeue are returned, and Update() is never called to remove the finalizer
func TestReconcileNamespaceDeletedErrorOnUpdate(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// Expect a call to get the namespace
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			ns.Name = "myns"
			ns.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			ns.Annotations = map[string]string{
				constants.OCILoggingIDAnnotation: "myocid",
			}
			ns.Finalizers = []string{"someFinalizer", namespaceControllerFinalizer}
			return nil
		})

	// Expect no call to update the namespace
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	nc, err := newTestController(mock)
	asserts.NoError(err)

	// Force a failure
	returnedErr := fmt.Errorf("error updating OCI Logging")
	removeNamespaceLoggingFunc = func(_ context.Context, _ client.Client, _ string) (bool, error) {
		return false, returnedErr
	}
	defer func() { removeNamespaceLoggingFunc = removeNamespaceLogging }()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "myns"},
	}
	result, _ := nc.Reconcile(context.TODO(), req)

	mocker.Finish()
	asserts.True(result.Requeue)
	asserts.Greater(result.RequeueAfter.Seconds(), time.Duration(0).Seconds())
}

// TestReconcileNamespaceDeletedNoFinalizer tests the Reconcile method for the following use case
// GIVEN a request to Reconcile a Namespace resource
// WHEN the namespace DeletionTimestamp has been set (namespace is in the process of being deleted)
// AND our finalizer doesn't exist (for example, we removed it on a previous reconcile)
// THEN we do not update the namespace or attempt to remove any logging config
func TestReconcileNamespaceDeletedNoFinalizer(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// Expect a call to get the namespace
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			ns.Name = "myns"
			ns.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			ns.Annotations = map[string]string{
				constants.OCILoggingIDAnnotation: "myocid",
			}
			ns.Finalizers = []string{"someFinalizer"}
			return nil
		})

	nc, err := newTestController(mock)
	asserts.NoError(err)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "myns"},
	}
	result, err := nc.Reconcile(context.TODO(), req)

	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(ctrl.Result{}, result)
}

// Test_removeFinalizer tests the removeFinalizer method for the following use case
// GIVEN a request to removeFinalizer for a Namespace resource
// WHEN the namespace has the NamespaceController finalizer present
// THEN the NamespaceController finalizer is removed from the Namespace resource and no requeue is indicated
func Test_removeFinalizer(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// Expect a call to update the namespace that succeeds
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.UpdateOption) error {
			return nil
		})

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "myns",
			Finalizers: []string{namespaceControllerFinalizer, "anotherFinalizer"},
		},
	}

	nc, err := newTestController(mock)
	asserts.NoError(err)

	result, err := nc.removeFinalizer(context.TODO(), ns, logger)

	mocker.Finish()
	asserts.NoError(err)
	asserts.NotContainsf(ns.Finalizers, namespaceControllerFinalizer, "Finalizer not removed: ", ns.Finalizers)
	asserts.Equal(ctrl.Result{}, result)
}

// Test_removeFinalizerNotPresent tests the removeFinalizer method for the following use case
// GIVEN a request to removeFinalizer for a Namespace resource
// WHEN the namespace does not have the NamespaceController finalizer present
// THEN the NamespaceController finalizer field unchanged and no requeue is indicated
func Test_removeFinalizerNotPresent(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// Expect a call to update the namespace that succeeds
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.UpdateOption) error {
			return nil
		})

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "myns",
			Finalizers: []string{"anotherFinalizer"},
		},
	}

	nc, err := newTestController(mock)
	asserts.NoError(err)

	result, err := nc.removeFinalizer(context.TODO(), ns, logger)

	mocker.Finish()
	asserts.NoError(err)
	asserts.Equalf(ns.Finalizers, []string{"anotherFinalizer"}, "Finalizers modified unexpectedly: %v", ns.Finalizers)
	asserts.Equal(ctrl.Result{}, result)
}

// Test_removeFinalizerErrorOnUpdate tests the removeFinalizer method for the following use case
// GIVEN a request to removeFinalizer for a Namespace resource
// WHEN the client returns an error on Update()
// THEN an error is returned
func Test_removeFinalizerErrorOnUpdate(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	nc, err := newTestController(mock)
	asserts.NoError(err)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "myns",
			Finalizers: []string{namespaceControllerFinalizer, "anotherFinalizer"},
		},
	}

	// Expect a call to update the namespace that fails
	expectedErr := fmt.Errorf("error updating namespace")
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.UpdateOption) error {
			return expectedErr
		})

	result, err := nc.removeFinalizer(context.TODO(), ns, logger)

	mocker.Finish()
	asserts.Error(err)
	asserts.Equalf(expectedErr, err, "Did not get expected error: %v", err)
	asserts.Equal(ctrl.Result{}, result)
}

// Test_reconcileNamespaceErrorOnUpdate tests the reconcileNamespace method for the following use case
// GIVEN a request to reconcileNamespace for a Namespace resource
// WHEN the client returns an error on Update()
// THEN an error and a requeue are returned
func Test_reconcileNamespaceErrorOnUpdate(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	nc, err := newTestController(mock)
	asserts.NoError(err)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "myns",
			Annotations: map[string]string{
				constants.OCILoggingIDAnnotation: "myocid",
			},
			Finalizers: []string{"anotherFinalizer"},
		},
	}

	expectedErr := fmt.Errorf("error updating namespace")

	// Expect a call to update the namespace that fails
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.UpdateOption) error {
			return expectedErr
		})

	err = nc.reconcileNamespace(context.TODO(), ns, logger)

	mocker.Finish()
	asserts.Error(err)
	asserts.Equalf(expectedErr, err, "Did not get expected error: %v", err)
}

// Test_reconcileNamespace tests the reconcileNamespace method for the following use case
// GIVEN a request to reconcileNamespace for a Namespace resource
// WHEN the namespace is configured for OCI Logging
// THEN no error or requeue are returned
func Test_reconcileNamespace(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	nc, err := newTestController(mock)
	asserts.NoError(err)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "myns",
			Annotations: map[string]string{
				constants.OCILoggingIDAnnotation: "myocid",
			},
			Finalizers: []string{"anotherFinalizer", namespaceControllerFinalizer},
		},
	}

	addNamespaceLoggingFunc = func(_ context.Context, _ client.Client, _ string, _ string) (bool, error) {
		return false, nil
	}
	defer func() { addNamespaceLoggingFunc = addNamespaceLogging }()

	err = nc.reconcileNamespace(context.TODO(), ns, logger)

	mocker.Finish()
	asserts.NoError(err)
}

// Test_reconcileNamespaceDelete tests the reconcileNamespaceDelete method for the following use case
// GIVEN a request to reconcileNamespaceDelete for a Namespace resource
// WHEN the namespace is configured for OCI Logging
// THEN no error is returned
//
// Largely a placeholder for now
func Test_reconcileNamespaceDelete(t *testing.T) {
	asserts := assert.New(t)

	nc, err := newTestController(fake.NewClientBuilder().WithScheme(testScheme).Build())
	asserts.NoErrorf(err, "Error creating test controller")
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "myns",
			Annotations: map[string]string{
				constants.OCILoggingIDAnnotation: "myocid",
			},
			Finalizers: []string{"anotherFinalizer", namespaceControllerFinalizer},
		},
	}

	removeNamespaceLoggingFunc = func(_ context.Context, _ client.Client, _ string) (bool, error) {
		return false, nil
	}
	defer func() { removeNamespaceLoggingFunc = removeNamespaceLogging }()

	err = nc.reconcileNamespaceDelete(context.TODO(), ns, logger)
	asserts.NoError(err)
}

// Test_reconcileOCILoggingRemoveOCILogging tests the reconcileOCILogging method for the following use case
// GIVEN a request to reconcileOCILogging for a Namespace resource
// WHEN the namespace is not configured for OCI Logging
// THEN no error is returned
func Test_reconcileOCILoggingRemoveOCILogging(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	nc, err := newTestController(mock)
	asserts.NoError(err)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "myns",
			Finalizers: []string{"anotherFinalizer"},
		},
	}

	removeCalled := false
	removeNamespaceLoggingFunc = func(_ context.Context, _ client.Client, _ string) (bool, error) {
		removeCalled = true
		return false, nil
	}
	defer func() { removeNamespaceLoggingFunc = removeNamespaceLogging }()

	err = nc.reconcileOCILogging(context.TODO(), ns, logger)

	mocker.Finish()
	asserts.NoError(err)
	asserts.True(removeCalled)
}

// Test_reconcileOCILoggingRemoveOCILoggingError tests the reconcileOCILogging method for the following use case
// GIVEN a request to reconcileOCILogging for a Namespace resource
// WHEN the namespace is not configured for OCI Logging and we fail removing the OCI Logging config from the configmap
// THEN an error is returned
func Test_reconcileOCILoggingRemoveOCILoggingError(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	nc, err := newTestController(mock)
	asserts.NoError(err)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "myns",
			Finalizers: []string{"anotherFinalizer"},
		},
	}

	expectedErr := fmt.Errorf("error removing OCI logging")
	removeNamespaceLoggingFunc = func(_ context.Context, _ client.Client, _ string) (bool, error) {
		return false, expectedErr
	}
	defer func() { removeNamespaceLoggingFunc = removeNamespaceLogging }()

	err = nc.reconcileOCILogging(context.TODO(), ns, logger)

	mocker.Finish()
	asserts.Error(err)
	asserts.Equal(expectedErr, err)
}

// Test_reconcileOCILoggingAddOCILoggingUpdated tests the reconcileOCILogging method for the following use case
// GIVEN a request to reconcileOCILogging for a Namespace resource
// WHEN the namespace is configured for OCI Logging and processed for the first time
// THEN the AddOCILogging function is called, the namespace finalizer is added, and no error is returned
func Test_reconcileOCILoggingAddOCILoggingUpdated(t *testing.T) {
	runAddOCILoggingTest(t, true)
}

// Test_reconcileOCILoggingAddOCILoggingUpdated tests the reconcileOCILogging method for the following use case
// GIVEN a request to reconcileOCILogging for a Namespace resource
// WHEN the AddOCILogging returns false (no changes)
// THEN the AddOCILogging function is called, the namespace finalizer is added, and no error is returned
func Test_reconcileOCILoggingAddOCILoggingNoOp(t *testing.T) {
	runAddOCILoggingTest(t, false)
}

// runAddOCILoggingTest - shared helper for the AddOCILogging tests
func runAddOCILoggingTest(t *testing.T, addLoggingResult bool) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	nc, err := newTestController(mock)
	asserts.NoError(err)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "myns",
			Annotations: map[string]string{
				constants.OCILoggingIDAnnotation: "myocid",
			},
			Finalizers: []string{"anotherFinalizer"},
		},
	}

	addCalled := false
	addNamespaceLoggingFunc = func(_ context.Context, _ client.Client, _ string, _ string) (bool, error) {
		addCalled = true
		return addLoggingResult, nil
	}
	defer func() { addNamespaceLoggingFunc = addNamespaceLogging }()

	// Expect a call to update the namespace annotations that succeeds
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.UpdateOption) error {
			return nil
		})

	// if the result from adding logging returns true (meaning the Fluentd configmap was updated), then
	// mock expections for restarting Fluentd
	if addLoggingResult {
		mockFluentdRestart(mock, asserts)
	}

	err = nc.reconcileOCILogging(context.TODO(), ns, logger)

	mocker.Finish()
	asserts.NoError(err)
	asserts.Contains(ns.Finalizers, namespaceControllerFinalizer)
	asserts.Len(ns.Finalizers, 2)
	asserts.Truef(addCalled, "Add OCI Logging fn not called")
}

// Test_reconcileOCILoggingFinalizerAlreadyAdded tests the reconcileOCILogging method for the following use case
// GIVEN a request to reconcileOCILogging for a Namespace resource
// WHEN the NamespaceController finalizer is already present
// THEN the AddOCILogging function is called, Update() is not called, the namespace finalizer set is unchanged, and no error is returned
func Test_reconcileOCILoggingFinalizerAlreadyAdded(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	nc, err := newTestController(mock)
	asserts.NoError(err)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "myns",
			Annotations: map[string]string{
				constants.OCILoggingIDAnnotation: "myocid",
			},
			Finalizers: []string{"anotherFinalizer", namespaceControllerFinalizer},
		},
	}

	addCalled := false
	addNamespaceLoggingFunc = func(_ context.Context, _ client.Client, _ string, _ string) (bool, error) {
		addCalled = true
		return false, nil
	}
	defer func() { addNamespaceLoggingFunc = addNamespaceLogging }()

	// Expect no calls to update the namespace
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

	err = nc.reconcileOCILogging(context.TODO(), ns, logger)

	mocker.Finish()
	asserts.NoError(err)
	asserts.Contains(ns.Finalizers, namespaceControllerFinalizer)
	asserts.Len(ns.Finalizers, 2)
	asserts.Truef(addCalled, "Add OCI Logging fn not called")
}

// Test_reconcileOCILoggingAddOCILoggingAddFailed tests the reconcileOCILogging method for the following use case
// GIVEN a request to reconcileOCILogging for a Namespace resource
// WHEN the OCI Logging annotation and NamespaceController finalizer are present and the AddOCILogging helper returns an error
// THEN the AddOCILogging function is called, Update() is not called, the namespace finalizer set is unchanged, and an error is returned
func Test_reconcileOCILoggingAddOCILoggingAddFailed(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	nc, err := newTestController(mock)
	asserts.NoError(err)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "myns",
			Annotations: map[string]string{
				constants.OCILoggingIDAnnotation: "myocid",
			},
			Finalizers: []string{"anotherFinalizer", namespaceControllerFinalizer},
		},
	}

	expectedErr := fmt.Errorf("error adding OCI Logging configuration")
	addNamespaceLoggingFunc = func(_ context.Context, _ client.Client, _ string, _ string) (bool, error) {
		return false, expectedErr
	}
	defer func() { addNamespaceLoggingFunc = addNamespaceLogging }()

	// Expect a call to update the namespace annotations that succeeds
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

	err = nc.reconcileOCILogging(context.TODO(), ns, logger)

	mocker.Finish()
	asserts.Error(err)
	asserts.Equal(expectedErr, err)
	asserts.Contains(ns.Finalizers, namespaceControllerFinalizer)
	asserts.Len(ns.Finalizers, 2)
}

// mockFluentdRestart - Mock expections for Fluentd daemonset restart
func mockFluentdRestart(mock *mocks.MockClient, asserts *assert.Assertions) {
	// Expect a call to get the Fleuntd Daemonset and another to update it with a restart time annotation
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ds *appsv1.DaemonSet) error {
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ds *appsv1.DaemonSet, opts ...client.UpdateOption) error {
			asserts.Contains(ds.Spec.Template.ObjectMeta.Annotations, vzconst.VerrazzanoRestartAnnotation)
			return nil
		})
}

// TestReconcileKubeSystem tests to make sure we do not reconcile
// Any resource that belong to the kube-system namespace
func TestReconcileKubeSystem(t *testing.T) {
	asserts := assert.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	nc, err := newTestController(mock)
	asserts.NoError(err)

	// create a request and reconcile it
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: vzconst.KubeSystem},
	}
	result, err := nc.Reconcile(context.TODO(), req)

	// Validate the results
	mocker.Finish()
	asserts.Nil(err)
	asserts.True(result.IsZero())
}

// Fake manager for unit testing
type fakeManager struct {
	client.Client
	scheme *runtime.Scheme
}

func (f fakeManager) Start(ctx context.Context) error {
	return nil
}

func (f fakeManager) GetControllerOptions() v1alpha1.ControllerConfigurationSpec {
	return v1alpha1.ControllerConfigurationSpec{}
}

func (f fakeManager) Add(_ manager.Runnable) error {
	return nil
}

func (f fakeManager) Elected() <-chan struct{} {
	return nil
}

func (f fakeManager) SetFields(_ interface{}) error {
	return nil
}

func (f fakeManager) AddMetricsExtraHandler(_ string, _ http.Handler) error {
	return nil
}

func (f fakeManager) AddHealthzCheck(_ string, _ healthz.Checker) error {
	return nil
}

func (f fakeManager) AddReadyzCheck(_ string, _ healthz.Checker) error {
	return nil
}

func (f fakeManager) GetConfig() *rest.Config {
	return nil
}

func (f fakeManager) GetScheme() *runtime.Scheme {
	return f.scheme
}

func (f fakeManager) GetClient() client.Client {
	return f.Client
}

func (f fakeManager) GetFieldIndexer() client.FieldIndexer {
	return nil
}

func (f fakeManager) GetCache() cache.Cache {
	return nil
}

func (f fakeManager) GetEventRecorderFor(_ string) record.EventRecorder {
	return nil
}

func (f fakeManager) GetRESTMapper() meta.RESTMapper {
	return nil
}

func (f fakeManager) GetAPIReader() client.Reader {
	return nil
}

func (f fakeManager) GetWebhookServer() *webhook.Server {
	return nil
}

func (f fakeManager) GetLogger() logr.Logger {
	return log.Log
}

var _ ctrl.Manager = fakeManager{}
