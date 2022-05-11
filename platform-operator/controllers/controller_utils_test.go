package controllers

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

const (
	testNS     = "verrazzano"
	testCMName = "po-val"
	testVZName = "test-vz"
)

func TestVzContainsResource(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	compContext := fakeComponentContext(mock, &testVZ)
	res0, ok0 := VzContainsResource(compContext, &testConfigMap)

	asserts.True(ok0)
	asserts.NotEmpty(res0)
	asserts.Equal(res0, "prometheus-operator")

	anotherCM := testConfigMap
	anotherCM.Name = "MonfigCap"

	res1, ok1 := VzContainsResource(compContext, &anotherCM)

	asserts.False(ok1)
	asserts.Empty(res1)
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

func fakeComponentContext(mock *mocks.MockClient, vz *vzapi.Verrazzano) spi.ComponentContext {
	compContext := spi.NewFakeContext(mock, vz, false)
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
			HelmValueOverrides: vzapi.HelmValueOverrides{
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

func True() *bool {
	x := true
	return &x
}
