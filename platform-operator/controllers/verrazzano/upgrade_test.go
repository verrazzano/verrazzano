// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"go.uber.org/zap"

	istiocomp "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	k8sapps "k8s.io/api/apps/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/helm"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// unitTestBomFIle is used for unit test
const unitTestBomFile = "../../verrazzano-bom.json"

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
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano
	labels := map[string]string{}

	config.SetDefaultBomFilePath(unitTestBomFile)
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
				State: vzapi.Ready,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
				},
			}
			verrazzano.Status.Components = makeVerrazzanoComponentStatusMap()
			return nil
		})

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, labels)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

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
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano
	labels := map[string]string{}

	config.SetDefaultBomFilePath(unitTestBomFile)
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
				State:   vzapi.Ready,
				Version: "0.2.0",
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
				},
			}
			verrazzano.Status.Components = makeVerrazzanoComponentStatusMap()
			return nil
		})

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, labels)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

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

// TestUpgradeInitComponents tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource when Status.Components is empty
// WHEN spec.version doesn't match status.version
// THEN ensure that the Status.components is populated
func TestUpgradeInitComponents(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

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
				State: vzapi.Ready,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
				},
			}
			return nil
		})

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, nil)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource to update components
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.NotZero(len(verrazzano.Status.Components), "Status.Components len should not be zero")
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

// TestUpgradeStarted tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN upgrade has not been started and spec.version doesn't match status.version
// THEN ensure a condition with type UpgradeStarted is added
func TestUpgradeStarted(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

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
				State: vzapi.Ready,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
				},
				Components: makeVerrazzanoComponentStatusMap(),
			}
			return nil
		})

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, nil)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

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
	asserts.Equal(time.Duration(1), result.RequeueAfter)
}

// TestUpgradeTooManyFailures tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN the current upgrade failed more than the failure limit
// THEN ensure that upgrade is not started
func TestUpgradeTooManyFailures(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

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
				Generation: 1,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "0.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State:      vzapi.Ready,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
				},
			}
			return nil
		})

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, nil)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

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
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

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
				Generation: 2,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "0.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State:      vzapi.Ready,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type: vzapi.UpgradeComplete,
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:2",
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:2",
					},
				},
			}
			return nil
		})

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, nil)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 8, "Incorrect number of conditions")
			asserts.Equal(verrazzano.Status.Conditions[7].Type, vzapi.UpgradeStarted)
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
	asserts.Equal(time.Duration(1), result.RequeueAfter)
}

// TestUpgradeNotStartedWhenPrevFailures tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN the current upgrade failures exceeds the limit, but there was a previous upgrade success
// THEN ensure that upgrade is not started
func TestUpgradeNotStartedWhenPrevFailures(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

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
				Generation: 2,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "0.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State:      vzapi.Ready,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.InstallComplete,
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type: vzapi.UpgradeComplete,
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:2",
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:2",
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:2",
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:2",
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:2",
					},
					{
						Type:    vzapi.UpgradeFailed,
						Message: "Upgrade failed generation:2",
					},
				},
			}
			return nil
		})

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, nil)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

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
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	fname, _ := filepath.Abs(unitTestBomFile)
	config.SetDefaultBomFilePath(fname)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{},
		}
	})
	defer registry.ResetGetComponentsFn()

	// Expect a call to get the Prometheus deployment and return a NotFound error.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "verrazzano-system", Name: "vmi-system-prometheus-0"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			return errors2.NewNotFound(schema.GroupResource{Group: "apps", Resource: "Deployment"}, name.Name)
		})

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
				State:      vzapi.Ready,
				Components: makeVerrazzanoComponentStatusMap(),
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

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, nil)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 3, "Incorrect number of conditions")
			asserts.Equal(vzapi.UpgradeComplete, verrazzano.Status.Conditions[2].Type, "Incorrect conditions")
			return nil
		})

	istiocomp.SetIstioUpgradeFunction(func(log *zap.SugaredLogger, imageOverrideString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
		return []byte(""), []byte(""), nil
	})
	defer istiocomp.SetDefaultIstioUpgradeFunction()
	istiocomp.SetRestartComponentsFunction(func(log *zap.SugaredLogger, err error, i istiocomp.IstioComponent, client client.Client) error {
		return nil
	})
	defer istiocomp.SetDefaultRestartComponentsFunction()

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

// TestUpgradeCompletedStatusReturnsError tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN the update of the VZ resource status fails and returns an error
// THEN ensure an error is returned and a requeue is requested
func TestUpgradeCompletedStatusReturnsError(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	fname, _ := filepath.Abs(unitTestBomFile)
	config.SetDefaultBomFilePath(fname)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{},
		}
	})
	defer registry.ResetGetComponentsFn()

	// Expect a call to get the Prometheus deployment and return a NotFound error.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "verrazzano-system", Name: "vmi-system-prometheus-0"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			return errors2.NewNotFound(schema.GroupResource{Group: "apps", Resource: "Deployment"}, name.Name)
		})

	istiocomp.SetIstioUpgradeFunction(func(log *zap.SugaredLogger, imageOverrideString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
		return []byte(""), []byte(""), nil
	})
	defer istiocomp.SetDefaultIstioUpgradeFunction()
	istiocomp.SetRestartComponentsFunction(func(log *zap.SugaredLogger, err error, i istiocomp.IstioComponent, client client.Client) error {
		return nil
	})
	defer istiocomp.SetDefaultRestartComponentsFunction()

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
				State:      vzapi.Ready,
				Components: makeVerrazzanoComponentStatusMap(),
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

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, nil)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 3, "Incorrect number of conditions")
			asserts.Equal(verrazzano.Status.Conditions[2].Type, vzapi.UpgradeComplete, "Incorrect conditions")
			return fmt.Errorf("Unexpected status error")
		})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.Error(err)
	asserts.Equal(true, result.Requeue)
}

// TestUpgradeHelmError tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN spec.version doesn't match status.version
// THEN ensure a condition with type UpgradeCompleted is added
func TestUpgradeHelmError(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

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
				State:      vzapi.Ready,
				Components: makeVerrazzanoComponentStatusMap(),
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

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, nil)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

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

// TestRetryUpgrade tests the retryUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after a failed upgrade
// WHEN when the restart-version annotation and the observed-restart-version annotation don't match and
// WHEN spec.version doesn't match status.version
// THEN ensure the annotations are updated and the reconciler requeues with the Ready StateType
func TestRetryUpgrade(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Expect a call to get the verrazzano resource.  Return resource with version and the restart-version annotation
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Finalizers: []string{finalizerName},
				Annotations: map[string]string{
					constants.RestartVersionAnnotation: "a",
				}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "0.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State: vzapi.Failed,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.UpgradeFailed,
					},
				},
				Components: makeVerrazzanoComponentStatusMap(),
			}
			return nil
		})

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, nil)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Expect a call to update annotations and ensure annotations are accurate
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano) error {
			asserts.Equal(verrazzano.ObjectMeta.Annotations[constants.RestartVersionAnnotation], "a", "Incorrect restart version")
			asserts.Equal(verrazzano.ObjectMeta.Annotations[constants.ObservedRestartVersionAnnotation], "a", "Incorrect observed restart version")
			return nil
		})

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 1, "Incorrect number of conditions")
			asserts.Equal(verrazzano.Status.State, vzapi.Ready, "Incorrect State")
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
	asserts.Equal(time.Duration(1), result.RequeueAfter)
}

// TestDontRetryUpgrade tests the retryUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after a failed upgrade
// WHEN when the restart-version annotation and the observed-restart-version annotation match and
// THEN ensure that
func TestDontRetryUpgrade(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Expect a call to get the verrazzano resource.  Return resource with version and the restart-version annotation
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Finalizers: []string{finalizerName},
				Annotations: map[string]string{
					constants.RestartVersionAnnotation:         "b",
					constants.ObservedRestartVersionAnnotation: "b",
				}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "0.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State: vzapi.Failed,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.UpgradeFailed,
					},
				},
				Components: makeVerrazzanoComponentStatusMap(),
			}
			return nil
		})

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, nil)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.True(result.IsZero())
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

func (r goodRunner) Run(_ *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}

func (r badRunner) Run(_ *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte(""), []byte("failure"), errors.New("Helm Error")
}
