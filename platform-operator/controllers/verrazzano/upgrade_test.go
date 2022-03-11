// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/vzinstance"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	helm2 "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	"github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// unitTestBomFIle is used for unit test
const unitTestBomFile = "../../verrazzano-bom.json"

//ingress list constants
const dnsDomain = "myenv.testverrazzano.com"
const keycloakURL = "keycloak." + dnsDomain
const esURL = "elasticsearch." + dnsDomain
const promURL = "prometheus." + dnsDomain
const grafanaURL = "grafana." + dnsDomain
const kialiURL = "kiali." + dnsDomain
const kibanaURL = "kibana." + dnsDomain
const rancherURL = "rancher." + dnsDomain
const consoleURL = "verrazzano." + dnsDomain

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
				State: vzapi.VzStateReady,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
				},
			}
			verrazzano.Status.Components = makeVerrazzanoComponentStatusMap()
			return nil
		})

	// The mocks are added to accomodate the expected calls to List instance when component is Ready
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ingressList *networkingv1.IngressList) error {
			ingressList.Items = []networkingv1.Ingress{}
			return nil
		}).AnyTimes()
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.NotZero(len(verrazzano.Status.Components), "Status.Components len should not be zero")
			return nil
		}).AnyTimes()

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
				State:   vzapi.VzStateReady,
				Version: "1.2.0",
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
				},
			}
			verrazzano.Status.Components = makeVerrazzanoComponentStatusMap()
			return nil
		})

	// The mocks are added to accomodate the expected calls to List instance when component is Ready
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ingressList *networkingv1.IngressList) error {
			ingressList.Items = []networkingv1.Ingress{}
			return nil
		}).AnyTimes()
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.NotZero(len(verrazzano.Status.Components), "Status.Components len should not be zero")
			return nil
		}).AnyTimes()

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
				State: vzapi.VzStateReady,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
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
	asserts.NotEqual(time.Duration(0), result.RequeueAfter)
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
				State: vzapi.VzStateReady,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
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
			asserts.Equal(verrazzano.Status.Conditions[1].Type, vzapi.CondUpgradeStarted)
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
	asserts.True(result.RequeueAfter.Seconds() <= 3)
}

// TestDeleteDuringUpgrade tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile a verrazzano resource after upgrade is started
// WHEN upgrade has started and deletion timestamp is not zero
// THEN ensure a condition with type Uninstall Started is added
func TestDeleteDuringUpgrade(t *testing.T) {
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
				Name:              name.Name,
				Namespace:         name.Namespace,
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
				Finalizers:        []string{finalizerName},
			}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "0.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State: vzapi.VzStateUpgrading,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
					{
						Type: vzapi.CondUpgradeStarted,
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

	// Expect a call to get the uninstall Job - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: getInstallNamespace(), Name: buildUninstallJobName(name)}, gomock.Not(gomock.Nil())).
		Return(errors2.NewNotFound(schema.GroupResource{Group: namespace, Resource: "Job"}, buildUninstallJobName(name)))

	// Expect a call to create the uninstall Job - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, job *batchv1.Job, opts ...client.CreateOption) error {
			asserts.Equalf(getInstallNamespace(), job.Namespace, "Job namespace did not match")
			asserts.Equalf(buildUninstallJobName(name), job.Name, "Job name did not match")
			return nil
		})

	// Expect a call to update the job - return success
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Expect a status update on the job
	mockStatus.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

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
	asserts.Equal(time.Duration(2)*time.Second, result.RequeueAfter)
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
				State:      vzapi.VzStateReady,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
					{
						Type:    vzapi.CondUpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type:    vzapi.CondUpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type:    vzapi.CondUpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type: vzapi.CondUpgradeComplete,
					},
					{
						Type:    vzapi.CondUpgradeFailed,
						Message: "Upgrade failed generation:2",
					},
					{
						Type:    vzapi.CondUpgradeFailed,
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
			asserts.Equal(verrazzano.Status.Conditions[7].Type, vzapi.CondUpgradeStarted)
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
	asserts.True(result.RequeueAfter.Seconds() <= 3)
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

	// Add mocks necessary for the system component restart
	mock.AddRestartMocks()

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
				State:      vzapi.VzStateReady,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
					{
						Type: vzapi.CondUpgradeStarted,
					},
				},
			}
			return nil
		}).AnyTimes()

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
			asserts.Equal(vzapi.CondUpgradeComplete, verrazzano.Status.Conditions[2].Type, "Incorrect conditions")
			return nil
		}).AnyTimes()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconcileLoop(reconciler, request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUpgradeCompletedMultipleReconcile tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN spec.version doesn't match status.version and reconcile is called multiple times
// THEN ensure a condition with type UpgradeCompleted is added
func TestUpgradeCompletedMultipleReconcile(t *testing.T) {
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
			fakeComponent{
				HelmComponent: helm2.HelmComponent{
					ReleaseName: "fake",
				},
			},
		}
	})
	defer registry.ResetGetComponentsFn()

	// Add mocks necessary for the system component restart
	mock.AddRestartMocks()

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
				State:      vzapi.VzStateReady,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
					{
						Type: vzapi.CondUpgradeStarted,
					},
				},
			}
			return nil
		}).AnyTimes()

	// Expect 2 calls to get the service account
	expectGetServiceAccountExists(mock, name, nil)

	// Expect 2 calls to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Expect calls to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 3, "Incorrect number of conditions")
			asserts.Equal(vzapi.CondUpgradeComplete, verrazzano.Status.Conditions[2].Type, "Incorrect conditions")
			return nil
		}).AnyTimes()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconcileLoop(reconciler, request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
	asserts.Len(upgradeTrackerMap, 0, "Expect upgradeTrackerMap to be empty")
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

	// Add mocks necessary for the system component restart
	mock.AddRestartMocks()

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
				State:      vzapi.VzStateReady,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
					{
						Type: vzapi.CondUpgradeStarted,
					},
				},
			}
			return nil
		}).AnyTimes()

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
			asserts.Equal(verrazzano.Status.Conditions[2].Type, vzapi.CondUpgradeComplete, "Incorrect conditions")
			return fmt.Errorf("Unexpected status error")
		}).AnyTimes()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconcileLoop(reconciler, request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
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
				Generation: 1,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "0.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State:      vzapi.VzStateReady,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
					{
						Type: vzapi.CondUpgradeStarted,
					},
				},
			}
			return nil
		}).AnyTimes()

	// expect a call to list any pending upgrade secrets for the component
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, secretList *v1.SecretList, opts ...client.ListOption) error {
			return nil
		}).AnyTimes()

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, nil)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconcileLoop(reconciler, request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.GreaterOrEqual(result.RequeueAfter.Seconds(), time.Duration(30).Seconds())
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
		State: vzapi.VzStateUpgrading,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondUpgradeStarted,
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
	result, err := reconcileUpgradeLoop(reconciler, &vz)

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
	// Need to use real component name since upgrade loops through registry
	componentName := oam.ComponentName

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
		State: vzapi.VzStateUpgrading,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondUpgradeStarted,
			},
		},
		Components: makeVerrazzanoComponentStatusMap(),
	}

	initStartingStates(&vz, componentName)

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
	mockComp.EXPECT().IsInstalled(gomock.Any()).Return(true, nil).AnyTimes()
	mockComp.EXPECT().PreUpgrade(gomock.Any()).Return(nil).Times(1)
	mockComp.EXPECT().Upgrade(gomock.Any()).Return(nil).Times(1)
	mockComp.EXPECT().PostUpgrade(gomock.Any()).Return(nil).Times(1)
	mockComp.EXPECT().Name().Return(componentName).AnyTimes()
	mockComp.EXPECT().IsReady(gomock.Any()).Return(true).AnyTimes()

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Add mocks necessary for the system component restart
	mock.AddRestartMocks()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 2, "Incorrect number of conditions")
			asserts.Equal(verrazzano.Status.Conditions[1].Type, vzapi.CondUpgradeComplete, "Incorrect condition")
			assert.Equal(t, vzapi.VzStateReady, verrazzano.Status.State)
			return nil
		}).Times(1)

	// Reconcile upgrade until state is done.  Put guard to prevent infinite loop
	reconciler := newVerrazzanoReconciler(mock)
	numComponentStates := 7
	var err error
	var result ctrl.Result
	for i := 0; i < numComponentStates; i++ {
		result, err = reconciler.reconcileUpgrade(vzlog.DefaultLogger(), &vz)
		if err != nil || !result.Requeue {
			break
		}
	}

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
}

// TestUpgradeComponentWithBlockingStatus tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an upgrade
// WHEN the component fails to upgrade since a status other than "deployed" exists
// THEN the offending secret is deleted so the upgrade can proceed
func TestUpgradeComponentWithBlockingStatus(t *testing.T) {
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
		State: vzapi.VzStateUpgrading,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondUpgradeStarted,
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
	mockComp.EXPECT().IsInstalled(gomock.Any()).Return(true, nil).AnyTimes()
	mockComp.EXPECT().PreUpgrade(gomock.Any()).Return(nil).Times(1)
	mockComp.EXPECT().Upgrade(gomock.Any()).Return(fmt.Errorf("Upgrade in progress")).AnyTimes()
	mockComp.EXPECT().Name().Return("testcomp").Times(1).AnyTimes()

	// expect a call to list any secrets with a status other than "deployed" for the component
	statuses := []string{"unknown", "uninstalled", "superseded", "failed", "uninstalling", "pending-install", "pending-upgrade", "pending-rollback"}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(statuses))))
	status := statuses[int(n.Int64())]
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, secretList *v1.SecretList, opts ...client.ListOption) error {
			secretList.Items = []v1.Secret{{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"name": "testcomp", "status": status},
				},
			}}
			return nil
		}).AnyTimes()

	// expect a call to delete the secret
	mock.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Reconcile upgrade
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconcileUpgradeLoop(reconciler, &vz)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
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
		State: vzapi.VzStateUpgrading,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondUpgradeStarted,
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
	mockEnabledComp.EXPECT().IsInstalled(gomock.Any()).Return(true, nil).AnyTimes()
	mockEnabledComp.EXPECT().PreUpgrade(gomock.Any()).Return(nil).Times(1)
	mockEnabledComp.EXPECT().Upgrade(gomock.Any()).Return(nil).Times(1)
	mockEnabledComp.EXPECT().PostUpgrade(gomock.Any()).Return(nil).Times(1)
	mockEnabledComp.EXPECT().IsReady(gomock.Any()).Return(true).AnyTimes()

	// Set disabled mock component expectations
	mockDisabledComp.EXPECT().Name().Return("DisabledComponent").Times(1).AnyTimes()
	mockDisabledComp.EXPECT().IsInstalled(gomock.Any()).Return(false, nil).AnyTimes()
	mockDisabledComp.EXPECT().PreUpgrade(gomock.Any()).Return(nil).Times(0)
	mockDisabledComp.EXPECT().Upgrade(gomock.Any()).Return(nil).Times(0)
	mockDisabledComp.EXPECT().PostUpgrade(gomock.Any()).Return(nil).Times(0)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Add mocks necessary for the system component restart
	mock.AddRestartMocks()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 2, "Incorrect number of conditions")
			asserts.Equal(verrazzano.Status.Conditions[1].Type, vzapi.CondUpgradeComplete, "Incorrect condition")
			assert.Equal(t, vzapi.VzStateReady, verrazzano.Status.State)
			return nil
		}).Times(1)

	// Reconcile upgrade
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconcileUpgradeLoop(reconciler, &vz)

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
				State: vzapi.VzStateFailed,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondUpgradeFailed,
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
			asserts.Equal(verrazzano.Status.State, vzapi.VzStateReady, "Incorrect State")
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
				State: vzapi.VzStateFailed,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondUpgradeFailed,
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
	asserts.False(isLastCondition(vzapi.VerrazzanoStatus{}, vzapi.CondUpgradeComplete), "isLastCondition should have returned false")
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
				Type: vzapi.CondUpgradeComplete,
			},
			{
				Type: vzapi.CondInstallFailed,
			},
		},
	}
	asserts.False(isLastCondition(st, vzapi.CondUpgradeComplete), "isLastCondition should have returned false")
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
				Type: vzapi.CondUpgradeComplete,
			},
			{
				Type: vzapi.CondInstallFailed,
			},
		},
	}
	asserts.True(isLastCondition(st, vzapi.CondInstallFailed), "isLastCondition should have returned true")
}

func (r goodRunner) Run(_ *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}

func (r badRunner) Run(_ *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte(""), []byte("failure"), errors.New("Helm Error")
}

// TestInstanceRestoreWithEmptyStatus tests the reconcileUpdate method for the following use case
// WHEN instance is restored via backup and restore instance status is not updated
// WHEN components are already installed.
// When verrazzano instance status is empty
// THEN update the instance urls appropriately
func TestInstanceRestoreWithEmptyStatus(t *testing.T) {
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

	// Expect a call to get the verrazzano resource.
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
				State: vzapi.VzStateReady,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
				},
			}
			verrazzano.Status.Components = makeVerrazzanoComponentStatusMap()
			return nil
		})

	// The mocks are added to accomodate the expected calls to List instance when component is Ready
	// The ingress list is from where urls are populated at Status.Instance
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ingressList *networkingv1.IngressList) error {
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
					ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-es-ingest"},
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
					ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-kiali"},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: kialiURL},
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
		}).AnyTimes()
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.NotZero(len(verrazzano.Status.Components), "Status.Components len should not be zero")
			return nil
		}).AnyTimes()

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

	// validating instance urls are updated
	// Status is empty in this case
	enabled := true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Console: &vzapi.ConsoleComponent{
					MonitoringComponent: vzapi.MonitoringComponent{
						Enabled: &enabled,
					},
				},
			},
		},
	}
	instanceInfo := vzinstance.GetInstanceInfo(spi.NewFakeContext(mock, vz, false))
	assert.NotNil(t, instanceInfo)
	assert.Equal(t, "https://"+consoleURL, *instanceInfo.ConsoleURL)
	assert.Equal(t, "https://"+rancherURL, *instanceInfo.RancherURL)
	assert.Equal(t, "https://"+keycloakURL, *instanceInfo.KeyCloakURL)
	assert.Equal(t, "https://"+esURL, *instanceInfo.ElasticURL)
	assert.Equal(t, "https://"+grafanaURL, *instanceInfo.GrafanaURL)
	assert.Equal(t, "https://"+kialiURL, *instanceInfo.KialiURL)
	assert.Equal(t, "https://"+kibanaURL, *instanceInfo.KibanaURL)
	assert.Equal(t, "https://"+promURL, *instanceInfo.PrometheusURL)
}

// TestInstanceRestoreWithPopulatedStatus tests the reconcileUpdate method for the following use case
// WHEN instance is restored via backup and restore instance status is not updated
// WHEN components are already installed.
// When verrazzano instance status is not empty
// THEN update the instance urls appropriately
func TestInstanceRestoreWithPopulatedStatus(t *testing.T) {
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

	// Expect a call to get the verrazzano resource.
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
				State: vzapi.VzStateReady,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
				},
			}
			verrazzano.Status.Components = makeVerrazzanoComponentStatusMap()
			return nil
		})

	// The mocks are added to accomodate the expected calls to List instance when component is Ready
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ingressList *networkingv1.IngressList) error {
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
					ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-es-ingest"},
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
					ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-kiali"},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: kialiURL},
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
		}).AnyTimes()
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.NotZero(len(verrazzano.Status.Components), "Status.Components len should not be zero")
			return nil
		}).AnyTimes()

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

	// validating instance urls are updated
	fakeInstanceInfo := vzapi.InstanceInfo{}
	consolefqdn := fmt.Sprintf("https://%s", consoleURL)
	fakeInstanceInfo.ConsoleURL = &consolefqdn

	rancherfqdn := fmt.Sprintf("https://%s", rancherURL)
	fakeInstanceInfo.ConsoleURL = &rancherfqdn

	keycloakfqdn := fmt.Sprintf("https://%s", keycloakURL)
	fakeInstanceInfo.ConsoleURL = &keycloakfqdn

	esfqdn := fmt.Sprintf("https://%s", esURL)
	fakeInstanceInfo.ConsoleURL = &esfqdn

	grafanafqdn := fmt.Sprintf("https://%s", grafanaURL)
	fakeInstanceInfo.ConsoleURL = &grafanafqdn

	kibanafqdn := fmt.Sprintf("https://%s", kibanaURL)
	fakeInstanceInfo.ConsoleURL = &kibanafqdn

	kialifqdn := fmt.Sprintf("https://%s", kialiURL)
	fakeInstanceInfo.ConsoleURL = &kialifqdn

	promfqdn := fmt.Sprintf("https://%s", promURL)
	fakeInstanceInfo.ConsoleURL = &promfqdn

	enabled := true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Console: &vzapi.ConsoleComponent{
					MonitoringComponent: vzapi.MonitoringComponent{
						Enabled: &enabled,
					},
				},
			},
		},
	}
	vz.Status.VerrazzanoInstance = &fakeInstanceInfo

	instanceInfo := vzinstance.GetInstanceInfo(spi.NewFakeContext(mock, vz, false))
	assert.NotNil(t, instanceInfo)
	assert.Equal(t, "https://"+consoleURL, *instanceInfo.ConsoleURL)
	assert.Equal(t, "https://"+rancherURL, *instanceInfo.RancherURL)
	assert.Equal(t, "https://"+keycloakURL, *instanceInfo.KeyCloakURL)
	assert.Equal(t, "https://"+esURL, *instanceInfo.ElasticURL)
	assert.Equal(t, "https://"+grafanaURL, *instanceInfo.GrafanaURL)
	assert.Equal(t, "https://"+kialiURL, *instanceInfo.KialiURL)
	assert.Equal(t, "https://"+kibanaURL, *instanceInfo.KibanaURL)
	assert.Equal(t, "https://"+promURL, *instanceInfo.PrometheusURL)
}

// initStartingStates inits the starting state for verrazzano and component upgrade
func initStartingStates(cr *vzapi.Verrazzano, compName string) {
	initStates(cr, vzStateStart, compName, compStateInit)
}

// initStates inits the specified state for verrazzano and component upgrade
func initStates(cr *vzapi.Verrazzano, vzState VerrazzanoUpgradeState, compName string, compState ComponentUpgradeState) {
	tracker := getUpgradeTracker(cr)
	tracker.vzState = vzState
	upgradeContext := tracker.getComponentUpgradeContext(compName)
	upgradeContext.state = compState
}

// reconcileUpgradeLoop
func reconcileUpgradeLoop(reconciler Reconciler, cr *vzapi.Verrazzano) (ctrl.Result, error) {
	numComponentStates := 7
	var err error
	var result ctrl.Result
	for i := 0; i < numComponentStates; i++ {
		result, err = reconciler.reconcileUpgrade(vzlog.DefaultLogger(), cr)
		if err != nil || !result.Requeue {
			break
		}
	}
	return result, err
}

// reconcileLoop
func reconcileLoop(reconciler Reconciler, request ctrl.Request) (ctrl.Result, error) {
	numComponentStates := 7
	var err error
	var result ctrl.Result
	for i := 0; i < numComponentStates; i++ {
		result, err = reconciler.Reconcile(request)
		if err != nil || !result.Requeue {
			break
		}
	}
	return result, err
}
