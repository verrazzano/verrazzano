// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
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

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

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
				Version: "1.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State:   vzapi.Ready,
				Version: "1.2.0",
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

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

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

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

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
				Version: "1.1.0"}
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

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

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

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

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

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

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

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

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

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{
				upgradeFunc: func(ctx spi.ComponentContext) error {
					return fmt.Errorf("Error running upgrade")
				},
			},
		}
	})

	defer registry.ResetGetComponentsFn()

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

// TestUpgradeIsCompInstalledFailure tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an upgrade
// WHEN when the comp.IsInstalled() function returns an error
// THEN an error is returned and the VZ status is not updated
func TestUpgradeIsCompInstalledFailure(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	vz := vzapi.Verrazzano{}
	vz.TypeMeta = metav1.TypeMeta{
		APIVersion: "install.verrazzano.io/v1alpha1",
		Kind:       "Verrazzano"}
	vz.ObjectMeta = metav1.ObjectMeta{
		Namespace:  namespace,
		Name:       name,
		Finalizers: []string{finalizerName},
	}
	vz.Spec = vzapi.VerrazzanoSpec{
		Version: "0.2.0"}
	vz.Status = vzapi.VerrazzanoStatus{
		State: vzapi.Upgrading,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.UpgradeStarted,
			},
		},
		Components: makeVerrazzanoComponentStatusMap(),
	}

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{
				isInstalledFunc: func(ctx spi.ComponentContext) (bool, error) {
					return false, fmt.Errorf("Error running isInstalled")
				},
			},
		}
	})
	defer registry.ResetGetComponentsFn()

	// Expect a call to update annotations and ensure annotations are accurate
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano) error {
			return nil
		}).Times(0)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			return nil
		}).Times(0)

	// Reconcile upgrade
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.reconcileUpgrade(zap.S(), &vz)

	// Validate the results
	mocker.Finish()
	asserts.Error(err)
	asserts.Equal(true, result.Requeue)
}

// TestUpgradeComponent tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an upgrade
// WHEN the component upgrades normally
// THEN no error is returned and the correct spi.Component upgrade methods have been returned
func TestUpgradeComponent(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	vz := vzapi.Verrazzano{}
	vz.TypeMeta = metav1.TypeMeta{
		APIVersion: "install.verrazzano.io/v1alpha1",
		Kind:       "Verrazzano"}
	vz.ObjectMeta = metav1.ObjectMeta{
		Namespace:  namespace,
		Name:       name,
		Finalizers: []string{finalizerName},
	}
	vz.Spec = vzapi.VerrazzanoSpec{
		Version: "0.2.0"}
	vz.Status = vzapi.VerrazzanoStatus{
		State: vzapi.Upgrading,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.UpgradeStarted,
			},
		},
		Components: makeVerrazzanoComponentStatusMap(),
	}

	mockComp := mocks.NewMockComponent(mocker)

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			mockComp,
		}
	})
	defer registry.ResetGetComponentsFn()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Set mock component expectations
	mockComp.EXPECT().IsInstalled(gomock.Any()).Return(true, nil)
	mockComp.EXPECT().PreUpgrade(gomock.Any()).Return(nil).Times(1)
	mockComp.EXPECT().Upgrade(gomock.Any()).Return(nil).Times(1)
	mockComp.EXPECT().PostUpgrade(gomock.Any()).Return(nil).Times(1)
	mockComp.EXPECT().Name().Return("testcomp").Times(1)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 2, "Incorrect number of conditions")
			asserts.Equal(verrazzano.Status.Conditions[1].Type, vzapi.UpgradeComplete, "Incorrect condition")
			assert.Equal(t, vzapi.Ready, verrazzano.Status.State)
			return nil
		}).Times(1)

	// Reconcile upgrade
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.reconcileUpgrade(zap.S(), &vz)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
}

// TestUpgradeMultipleComponentsOneDisabled tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an upgrade
// WHEN where one component is enabled and another is disabled
// THEN the upgrade completes normally and the correct spi.Component upgrade methods have not been invoked for the disabled component
func TestUpgradeMultipleComponentsOneDisabled(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	vz := vzapi.Verrazzano{}
	vz.TypeMeta = metav1.TypeMeta{
		APIVersion: "install.verrazzano.io/v1alpha1",
		Kind:       "Verrazzano"}
	vz.ObjectMeta = metav1.ObjectMeta{
		Namespace:  namespace,
		Name:       name,
		Finalizers: []string{finalizerName},
	}
	vz.Spec = vzapi.VerrazzanoSpec{
		Version: "0.2.0"}
	vz.Status = vzapi.VerrazzanoStatus{
		State: vzapi.Upgrading,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.UpgradeStarted,
			},
		},
		Components: makeVerrazzanoComponentStatusMap(),
	}

	mockEnabledComp := mocks.NewMockComponent(mocker)
	mockDisabledComp := mocks.NewMockComponent(mocker)

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			mockEnabledComp,
			mockDisabledComp,
		}
	})
	defer registry.ResetGetComponentsFn()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Set enabled mock component expectations
	mockEnabledComp.EXPECT().Name().Return("EnabledComponent").AnyTimes()
	mockEnabledComp.EXPECT().IsInstalled(gomock.Any()).Return(true, nil)
	mockEnabledComp.EXPECT().PreUpgrade(gomock.Any()).Return(nil).Times(1)
	mockEnabledComp.EXPECT().Upgrade(gomock.Any()).Return(nil).Times(1)
	mockEnabledComp.EXPECT().PostUpgrade(gomock.Any()).Return(nil).Times(1)

	// Set disabled mock component expectations
	mockDisabledComp.EXPECT().Name().Return("DisabledComponent").Times(1)
	mockDisabledComp.EXPECT().IsInstalled(gomock.Any()).Return(false, nil)
	mockDisabledComp.EXPECT().PreUpgrade(gomock.Any()).Return(nil).Times(0)
	mockDisabledComp.EXPECT().Upgrade(gomock.Any()).Return(nil).Times(0)
	mockDisabledComp.EXPECT().PostUpgrade(gomock.Any()).Return(nil).Times(0)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 2, "Incorrect number of conditions")
			asserts.Equal(verrazzano.Status.Conditions[1].Type, vzapi.UpgradeComplete, "Incorrect condition")
			assert.Equal(t, vzapi.Ready, verrazzano.Status.State)
			return nil
		}).Times(1)

	// Reconcile upgrade
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.reconcileUpgrade(zap.S(), &vz)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
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

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

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
					constants.UpgradeRetryVersion: "a",
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
			asserts.Equal(verrazzano.ObjectMeta.Annotations[constants.UpgradeRetryVersion], "a", "Incorrect restart version")
			asserts.Equal(verrazzano.ObjectMeta.Annotations[constants.ObservedUpgradeRetryVersion], "a", "Incorrect observed restart version")
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

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

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
					constants.UpgradeRetryVersion:         "b",
					constants.ObservedUpgradeRetryVersion: "b",
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
