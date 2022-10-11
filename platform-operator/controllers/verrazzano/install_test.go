// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testBomFile = "../../verrazzano-bom.json"
const fakeCompReleaseName = "verrazzano-authproxy"

// TestUpdate tests the reconcile func with updated generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the smaller LastReconciledGeneration than verrazzano CR in the request
// THEN ensure a condition with type InstallStarted
func TestUpdate(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "TestUpdate"
	lastReconciledGeneration := int64(2)
	reconcilingGen := int64(0)
	asserts, vz, result, fakeCompUpdated, err := testUpdate(t,
		lastReconciledGeneration+1, reconcilingGen, lastReconciledGeneration,
		"1.3.0", "1.3.0", namespace, name, "true")
	defer reset()
	asserts.NoError(err)
	asserts.Equal(vzapi.VzStateReconciling, vz.Status.State)
	asserts.True(*fakeCompUpdated)
	asserts.Equal(vz.Generation, vz.Status.Components[fakeCompReleaseName].LastReconciledGeneration)
	asserts.Equal(vzapi.CondInstallStarted, vz.Status.Components[fakeCompReleaseName].Conditions[0].Type)
	asserts.Equal(vzapi.CondInstallComplete, vz.Status.Components[fakeCompReleaseName].Conditions[1].Type)
	asserts.False(result.Requeue)
}

// TestNoUpdateSameGeneration tests the reconcile func with same generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the same LastReconciledGeneration as verrazzano CR
// THEN ensure a condition with type InstallStarted is not added
func TestNoUpdateSameGeneration(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "TestSameGeneration"
	lastReconciledGeneration := int64(2)
	reconcilingGen := int64(0)
	asserts, vz, result, fakeCompUpdated, err := testUpdate(t, lastReconciledGeneration, reconcilingGen, lastReconciledGeneration,
		"1.3.1", "1.3.1", namespace, name, "true")
	defer reset()
	asserts.NoError(err)
	asserts.Equal(vzapi.VzStateReady, vz.Status.State)
	asserts.Nil(fakeCompUpdated)
	asserts.False(result.Requeue)
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
	reconcilingGen := int64(0)
	asserts, vz, result, fakeCompUpdated, err := testUpdate(t, lastReconciledGeneration+1, reconcilingGen, lastReconciledGeneration,
		"1.3.0", "1.2.0", namespace, name, "true")
	defer reset()
	asserts.NoError(err)
	asserts.Equal(vzapi.VzStateUpgrading, vz.Status.State)
	asserts.Nil(fakeCompUpdated)
	asserts.True(result.Requeue)
}

// TestUpdateOnUpdate tests the reconcile func with updated generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the smaller LastReconciledGeneration than verrazzano CR in the request
// THEN ensure a condition with type InstallStarted
func TestUpdateOnUpdate(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	lastReconciledGeneration := int64(2)
	reconcilingGen := int64(3)
	asserts, vz, result, fakeCompUpdated, err := testUpdate(t,
		reconcilingGen+1, reconcilingGen, lastReconciledGeneration,
		"1.3.3", "1.3.3", namespace, name, "true")
	defer reset()
	asserts.NoError(err)
	asserts.Equal(vzapi.VzStateReconciling, vz.Status.State)
	asserts.True(*fakeCompUpdated)
	asserts.Equal(vz.Generation, vz.Status.Components[fakeCompReleaseName].LastReconciledGeneration)
	asserts.Equal(vzapi.CondInstallStarted, vz.Status.Components[fakeCompReleaseName].Conditions[0].Type)
	asserts.Equal(vzapi.CondInstallComplete, vz.Status.Components[fakeCompReleaseName].Conditions[1].Type)
	asserts.False(result.Requeue)
}

// TestUpdateFalseMonitorChanges tests the reconcile func with updated generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the smaller LastReconciledGeneration but MonitorOverrides returns false
// THEN ensure a condition with type InstallStarted is not added
func TestUpdateFalseMonitorChanges(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "TestUpdate"
	lastReconciledGeneration := int64(2)
	reconcilingGen := int64(0)
	asserts, vz, result, fakeCompUpdated, err := testUpdate(t,
		lastReconciledGeneration+1, reconcilingGen, lastReconciledGeneration,
		"1.3.0", "1.3.0", namespace, name, "false")
	defer reset()
	asserts.NoError(err)
	asserts.Equal(vzapi.VzStateReady, vz.Status.State)
	asserts.Nil(fakeCompUpdated)
	asserts.False(result.Requeue)
}

func reset() {
	registry.ResetGetComponentsFn()
	config.SetDefaultBomFilePath("")
	helm.SetDefaultChartStatusFunction()
	config.SetDefaultBomFilePath("")
	helm.SetDefaultChartStatusFunction()
	config.TestProfilesDir = ""
}

func testUpdate(t *testing.T,
	//mocker *gomock.Controller, mock *mocks.MockClient,
	vzCrGen, reconcilingGen, lastReconciledGeneration int64,
	//mockStatus *mocks.MockStatusWriter,
	specVer, statusVer, namespace, name, monitorChanges string) (*assert.Assertions, *vzapi.Verrazzano, ctrl.Result, *bool, error) {
	asserts := assert.New(t)

	config.SetDefaultBomFilePath(testBomFile)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	fakeComp := fakeComponent{}
	fakeComp.ReleaseName = fakeCompReleaseName
	fakeComp.SupportsOperatorInstall = true
	fakeComp.monitorChanges = monitorChanges
	var fakeCompUpdated *bool
	fakeComp.installFunc = func(ctx spi.ComponentContext) error {
		update := true
		fakeCompUpdated = &update
		return nil
	}
	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComp,
		}
	})
	compStatusMap := makeVerrazzanoComponentStatusMap()
	for _, status := range compStatusMap {
		status.ReconcilingGeneration = reconcilingGen
		status.LastReconciledGeneration = lastReconciledGeneration
	}
	var vz *vzapi.Verrazzano
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
				Generation: vzCrGen,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: specVer}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State:   vzapi.VzStateReady,
				Version: statusVer,
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
		DoAndReturn(func(ctx context.Context, ingressList *networkingv1.IngressList, options ...client.UpdateOption) error {
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
	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	var verrazzanoToUse vzapi.Verrazzano
	labels := map[string]string{}
	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, labels)
	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)
	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	config.TestProfilesDir = "../../manifests/profiles"
	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)
	mocker.Finish()
	return asserts, vz, result, fakeCompUpdated, err
}
