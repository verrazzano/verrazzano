// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNS     = "verrazzano"
	testCMName = "po-val"
	testVZName = "test-vz"
)

// TestVzContainsResource tests that the component name along with
// bool value true is returned if k8s object is referenced in the CR as
// override. Return false along with an empty string for other cases
func TestVzContainsResource(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	compContext := fakeComponentContext(mock, &testVZ)
	res0, ok0 := VzContainsResource(compContext, testConfigMap.Name, testConfigMap.Kind)

	asserts.True(ok0)
	asserts.NotEmpty(res0)
	asserts.Equal(res0, "prometheus-operator")

	anotherCM := testConfigMap
	anotherCM.Name = "MonfigCap"

	res1, ok1 := VzContainsResource(compContext, anotherCM.Name, anotherCM.Kind)
	mocker.Finish()
	asserts.False(ok1)
	asserts.Empty(res1)
}

// TestVzContainsResourceMonitoringDisabled tests that if MonitorChanges is set to false,
// then the component should be ignored and an empty string along with false bool value is
// returned.
func TestVzContainsResourceMonitoringDisabled(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	vz := testVZ
	*vz.Spec.Components.PrometheusOperator.MonitorChanges = false
	compContext := fakeComponentContext(mock, &vz)
	res0, ok0 := VzContainsResource(compContext, testConfigMap.Name, testConfigMap.Kind)

	mocker.Finish()
	asserts.False(ok0)
	asserts.Empty(res0)
}

// TestUpdateVerrazzanoForInstallOverrides tests that the call to update Verrazzano Status
// is made and doesn't return an error
func TestUpdateVerrazzanoForInstallOverrides(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	compContext := fakeComponentContext(mock, &testVZ)

	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vz *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			return nil
		})
	err := UpdateVerrazzanoForInstallOverrides(mock, compContext, "prometheus-operator")
	mocker.Finish()
	asserts.Nil(err)
}

// TestUpdateVerrazzanoForInstallOverrides tests that if Verrazzano hasn't initialized component status
// an error will be returned by the function
func TestUpdateVerrazzanoForInstallOverridesError(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	vz := testVZ
	vz.Status.Components = nil
	compContext := fakeComponentContext(mock, &vz)

	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).MaxTimes(0)
	err := UpdateVerrazzanoForInstallOverrides(mock, compContext, "prometheus-operator")
	mocker.Finish()
	asserts.NotNil(err)
}

var testConfigMap = corev1.ConfigMap{
	TypeMeta: metav1.TypeMeta{
		Kind: "ConfigMap",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      testCMName,
		Namespace: testNS,
	},
	Immutable:  nil,
	Data:       map[string]string{"override": "true"},
	BinaryData: nil,
}

// creates a component context for testing
func fakeComponentContext(mock *mocks.MockClient, vz *vzapi.Verrazzano) spi.ComponentContext {
	compContext := spi.NewFakeContext(mock, vz, nil, false)
	return compContext
}

var compStatusMap = makeVerrazzanoComponentStatusMap()
var testVZ = vzapi.Verrazzano{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "install.verrazzano.io/v1alpha1",
		Kind:       "Verrazzano",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      testVZName,
		Namespace: testNS,
	},
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{PrometheusOperator: &vzapi.PrometheusOperatorComponent{
			Enabled: True(),
			InstallOverrides: vzapi.InstallOverrides{
				MonitorChanges: True(),
				ValueOverrides: []vzapi.Overrides{
					{
						ConfigMapRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: testCMName,
							},
							Key:      "",
							Optional: nil,
						},
					},
				},
			},
		}},
	},
	Status: vzapi.VerrazzanoStatus{
		State: vzapi.VzStateReady,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondInstallComplete,
			},
		},
		Components: compStatusMap,
	},
}

// creates a component status map for testing
func makeVerrazzanoComponentStatusMap() vzapi.ComponentStatusMap {
	statusMap := make(vzapi.ComponentStatusMap)
	for _, comp := range registry.GetComponents() {
		if comp.IsOperatorInstallSupported() {
			statusMap[comp.Name()] = &vzapi.ComponentStatusDetails{
				Name: comp.Name(),
				Conditions: []vzapi.Condition{
					{
						Type:   vzapi.CondInstallComplete,
						Status: corev1.ConditionTrue,
					},
				},
				State: vzapi.CompStateReady,
			}
		}
	}
	return statusMap
}

// return address of a bool var with true value
func True() *bool {
	x := true
	return &x
}
