// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"testing"
	"time"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
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

// TestUpdate tests the reconcile func with updated generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the smaller LastReconciledGeneration than verrazzano CR in the request
// THEN ensure a condition with type InstallStarted
func TestUpdate(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	lastReconciledGeneration := int64(2)
	var verrazzanoToUse vzapi.Verrazzano
	labels := map[string]string{}

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)
	var vz *vzapi.Verrazzano
	fakeComp := fakeComponent{}
	fakeComp.ReleaseName = "my-component"
	fakeComp.SupportsOperatorInstall = true
	fakeCompUpdated := false
	fakeComp.installFunc = func(ctx spi.ComponentContext) error {
		fakeCompUpdated = true
		return nil
	}
	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComp,
		}
	})
	defer registry.ResetGetComponentsFn()
	compStatusMap := makeVerrazzanoComponentStatusMap()
	for _, status := range compStatusMap {
		status.LastReconciledGeneration = lastReconciledGeneration
	}
	// Expect a call to get the verrazzano resource.  Return resource with version
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			vz = verrazzano
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Generation: lastReconciledGeneration + 1,
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
			verrazzano.Status.Components = compStatusMap
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
	asserts.Equal(vzapi.VzStateInstalling, vz.Status.State)
	asserts.True(fakeCompUpdated)
	asserts.True(result.Requeue)
}

// TestNoUpdateSameGeneration tests the reconcile func with same generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the same LastReconciledGeneration as verrazzano CR
// THEN ensure a condition with type InstallStarted is not added
func TestNoUpdateSameGeneration(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	lastReconciledGeneration := int64(2)
	var verrazzanoToUse vzapi.Verrazzano
	labels := map[string]string{}

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)
	var vz *vzapi.Verrazzano
	compStatusMap := makeVerrazzanoComponentStatusMap()
	for _, status := range compStatusMap {
		status.LastReconciledGeneration = lastReconciledGeneration
	}

	// Expect a call to get the verrazzano resource.  Return resource with version
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			vz = verrazzano
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Generation: lastReconciledGeneration,
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
			verrazzano.Status.Components = compStatusMap
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
	asserts.Equal(vzapi.VzStateReady, vz.Status.State)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUpdateWithUpgrade tests the reconcile func with updated generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the smaller LastReconciledGeneration than verrazzano CR in the request
// THEN ensure a condition with type UpgradeStarted
func TestUpdateWithUpgrade(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	lastReconciledGeneration := int64(2)
	var verrazzanoToUse vzapi.Verrazzano
	labels := map[string]string{}

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)
	var vz *vzapi.Verrazzano
	compStatusMap := makeVerrazzanoComponentStatusMap()
	for _, status := range compStatusMap {
		status.LastReconciledGeneration = lastReconciledGeneration
	}
	// Expect a call to get the verrazzano resource.  Return resource with version
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			vz = verrazzano
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Generation: lastReconciledGeneration + 1,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "1.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State:   vzapi.VzStateReady,
				Version: "1.1.0",
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
				},
			}
			verrazzano.Status.Components = compStatusMap
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
	asserts.Equal(vzapi.VzStateUpgrading, vz.Status.State)
	asserts.True(result.Requeue)
}
