// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"

	"github.com/verrazzano/verrazzano/operator/internal/util/helm"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/operator/mocks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// goodRunner is used to test helm success without actually running an OS exec command
type goodRunner struct {
}

// badRunner is used to test helm failure without actually running an OS exec command
type badRunner struct {
}

// Generate mocs for the Kerberos Client and StatusWriter interfaces for use in tests.
//go:generate mockgen -destination=../mocks/controller_mock.go -package=mocks -copyright_file=../hack/boilerplate.go.txt sigs.k8s.io/controller-runtime/pkg/client Client,StatusWriter

// TestUpgradeNoVersion tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN a verrazzano version is empty
// THEN ensure a condition with type UpgradeStarted is not added
func TestUpgradeNoVersion(t *testing.T) {
	namespace := "verrazzano"
	name := "test"

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.  Return resource with version
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Finalizers: []string{finalizerName}}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
				},
			}
			return nil
		})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUpgradeSameVersion tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN a verrazzano spec.version is the same as the status.version
// THEN ensure a condition with type UpgradeStarted is not added
func TestUpgradeSameVersion(t *testing.T) {
	namespace := "verrazzano"
	name := "test"

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.  Return resource with version
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "0.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				Version: "0.2.0",
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
				},
			}
			return nil
		})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUpgradeStarted tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN upgrade has not been started and spec.version doesn't match status.version
// THEN ensure a condition with type UpgradeStarted is added
func TestUpgradeStarted(t *testing.T) {
	namespace := "verrazzano"
	name := "test"

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.  Return resource with version
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "0.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
				},
			}
			return nil
		})

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 2, "Incorrect number of conditions")
			asserts.Equal(verrazzano.Status.Conditions[1].Type, vzapi.UpgradeStarted)
			return nil
		})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUpgradeTooManyFailures tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN the current upgrade failed more than the failure limet
// THEN ensure that upgrade is not started
func TestUpgradeTooManyFailures(t *testing.T) {
	namespace := "verrazzano"
	name := "test"

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.  Return resource with version
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "0.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
					{
						Type: vzapi.UpgradeFailed,
					},
					{
						Type: vzapi.UpgradeFailed,
					},
					{
						Type: vzapi.UpgradeFailed,
					},
				},
			}
			return nil
		})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUpgradeStartedWhenPrevFailures tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN the total upgrade failures exceed the limit, but the current upgrade is under the limit
// THEN ensure that upgrade is started
func TestUpgradeStartedWhenPrevFailures(t *testing.T) {
	namespace := "verrazzano"
	name := "test"

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.  Return resource with version
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "0.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
					{
						Type: vzapi.UpgradeFailed,
					},
					{
						Type: vzapi.UpgradeFailed,
					},
					{
						Type: vzapi.UpgradeComplete,
					},
					{
						Type: vzapi.UpgradeFailed,
					},
					{
						Type: vzapi.UpgradeFailed,
					},
				},
			}
			return nil
		})

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 7, "Incorrect number of conditions")
			asserts.Equal(verrazzano.Status.Conditions[6].Type, vzapi.UpgradeStarted)
			return nil
		})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUpgradeNotStartedWhenPrevFailures tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN the current upgrade failures exceeds the limit, but there was a previous upgrade success
// THEN ensure that upgrade is not started
func TestUpgradeNotStartedWhenPrevFailures(t *testing.T) {
	namespace := "verrazzano"
	name := "test"

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.  Return resource with version
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "0.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
					{
						Type: vzapi.UpgradeFailed,
					},
					{
						Type: vzapi.UpgradeFailed,
					},
					{
						Type: vzapi.UpgradeComplete,
					},
					{
						Type: vzapi.UpgradeFailed,
					},
					{
						Type: vzapi.UpgradeFailed,
					},
					{
						Type: vzapi.UpgradeFailed,
					},
				},
			}
			return nil
		})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUpgradeCompleted tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN spec.version doesn't match status.version
// THEN ensure a condition with type UpgradeCompleted is added
func TestUpgradeCompleted(t *testing.T) {
	namespace := "verrazzano"
	name := "test"

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.  Return resource with version
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "0.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
					{
						Type: vzapi.UpgradeStarted,
					},
				},
			}
			return nil
		})

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 3, "Incorrect number of conditions")
			asserts.Equal(verrazzano.Status.Conditions[2].Type, vzapi.UpgradeComplete, "Incorrect conditions")
			return nil
		})

	// Inject a fake cmd runner to the real helm is not called
	helm.SetCmdRunner(goodRunner{})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUpgradeHelmError tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN spec.version doesn't match status.version
// THEN ensure a condition with type UpgradeCompleted is added
func TestUpgradeHelmError(t *testing.T) {
	namespace := "verrazzano"
	name := "test"

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.  Return resource with version
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "0.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
					{
						Type: vzapi.UpgradeStarted,
					},
				},
			}
			return nil
		})

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 3, "Incorrect number of conditions")
			asserts.Equal(verrazzano.Status.Conditions[2].Type, vzapi.UpgradeFailed, "Incorrect condition")
			return nil
		})

	// Inject a bad cmd runner to the real helm is not called
	helm.SetCmdRunner(badRunner{})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestIsLastConditionNone tests the isLastCondition method for the following use case
// GIVEN an empty array of conditions
// WHEN isLastCondition is called
// THEN ensure that false
func TestIsLastConditionNone(t *testing.T) {
	asserts := assert.New(t)
	asserts.False(isLastCondition(vzapi.VerrazzanoStatus{}, vzapi.UpgradeComplete), "isLastCondition should have returned false")
}

// TestIsLastConditionFalse tests the isLastCondition method for the following use case
// GIVEN an array of conditions
// WHEN isLastCondition is called where the target last condition doesn't match the actual last condition
// THEN ensure that false is returned
func TestIsLastConditionFalse(t *testing.T) {
	asserts := assert.New(t)
	st := vzapi.VerrazzanoStatus{
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.UpgradeComplete,
			},
			{
				Type: vzapi.InstallFailed,
			},
		},
	}
	asserts.False(isLastCondition(st, vzapi.UpgradeComplete), "isLastCondition should have returned false")
}

// TestIsLastConditionTrue tests the isLastCondition method for the following use case
// GIVEN an array of conditions
// WHEN isLastCondition is called where the target last condition matches the actual last condition
// THEN ensure that true is returned
func TestIsLastConditionTrue(t *testing.T) {
	asserts := assert.New(t)
	st := vzapi.VerrazzanoStatus{
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.UpgradeComplete,
			},
			{
				Type: vzapi.InstallFailed,
			},
		},
	}
	asserts.True(isLastCondition(st, vzapi.InstallFailed), "isLastCondition should have returned true")
}

func (r goodRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}

func (r badRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte(""), []byte("failure"), errors.New("Helm Error")
}
