// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqloperator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestGetInstallOverrides tests the GetInstallOverrides function
// GIVEN a call to GetInstallOverrides
// WHEN there is a valid MySQL Operator configuration
// THEN the correct Helm overrides are returned
func TestGetInstallOverrides(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				MySQLOperator: &vzapi.MySQLOperatorComponent{
					InstallOverrides: vzapi.InstallOverrides{
						ValueOverrides: []vzapi.Overrides{
							{Values: &apiextensionsv1.JSON{
								Raw: []byte("{\"key1\": \"value1\"}")},
							},
						},
					},
				},
			},
		},
	}

	comp := NewComponent()
	overrides := comp.GetOverrides(vz).([]vzapi.Overrides)
	assert.Equal(t, []byte("{\"key1\": \"value1\"}"), overrides[0].Values.Raw)

	vz = &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				MySQLOperator: nil,
			},
		},
	}

	overrides = comp.GetOverrides(vz).([]vzapi.Overrides)
	assert.Equal(t, []vzapi.Overrides{}, overrides)

	beta1vz := &v1beta1.Verrazzano{
		Spec: v1beta1.VerrazzanoSpec{
			Components: v1beta1.ComponentSpec{
				MySQLOperator: &v1beta1.MySQLOperatorComponent{
					InstallOverrides: v1beta1.InstallOverrides{
						ValueOverrides: []v1beta1.Overrides{
							{Values: &apiextensionsv1.JSON{
								Raw: []byte("{\"key1\": \"value1\"}")},
							},
						},
					},
				},
			},
		},
	}

	overrides2 := comp.GetOverrides(beta1vz).([]v1beta1.Overrides)
	assert.Equal(t, []byte("{\"key1\": \"value1\"}"), overrides2[0].Values.Raw)
}

// TestIsMySQLOperatorReady tests the isReady function
// GIVEN a call to isReady
// WHEN the deployment object has enough replicas available
// THEN true is returned
func TestIsMySQLOperatorReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"app": ComponentName},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": ComponentName},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        ComponentName + "-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"app":               ComponentName,
				},
			},
		},
	).Build()

	mysqlOperator := NewComponent().(mysqlOperatorComponent)
	assert.True(t, mysqlOperator.isReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

// TestIsMySQLOperatorNotReady tests the isReady function
// GIVEN a call to isReady
// WHEN the deployment object does NOT have enough replicas available
// THEN false is returned
func TestIsMySQLOperatorNotReady(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentNamespace,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   0,
		},
	}).Build()
	mysqlOperator := NewComponent().(mysqlOperatorComponent)
	assert.False(t, mysqlOperator.isReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

// TestIsInstalled tests the isInstalled function
// GIVEN a call to isInstalled
// WHEN the deployment object exists
// THEN true is returned
func TestIsInstalled(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"app": ComponentName},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": ComponentName},
				},
			},
		},
	).Build()

	mysqlOperator := NewComponent().(mysqlOperatorComponent)
	assert.True(t, mysqlOperator.isInstalled(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

// TestIsInstalledFalse tests the isInstalled function
// GIVEN a call to isInstalled
// WHEN the deployment object does not exist
// THEN false is returned
func TestIsInstalledFalse(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects().Build()
	mysqlOperator := NewComponent().(mysqlOperatorComponent)
	assert.False(t, mysqlOperator.isInstalled(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

// TestAppendOverrides tests the AppendOverrides function
// GIVEN a call to AppendOverrides
// WHEN the verrazzano-container-registry secret exists in the mysql-operator namespace
// THEN the correct Helm overrides are returned
func TestAppendOverrides(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      constants.GlobalImagePullSecName,
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(secret).Build()

	kvs, err := AppendOverrides(spi.NewFakeContext(fakeClient, nil, nil, false), "", "", "", []bom.KeyValue{{Key: "key1", Value: "value1"}})
	assert.Nil(t, err)
	assert.Len(t, kvs, 2)
	assert.Equal(t, bom.KeyValue{Key: "key1", Value: "value1"}, kvs[0])
	assert.Equal(t, bom.KeyValue{Key: "image.pullSecrets.enabled", Value: "true"}, kvs[1])
}

// TestAppendOverridesNoSecret tests the AppendOverrides function
// GIVEN a call to AppendOverrides
// WHEN the verrazzano-container-registry secret does not exist in the mysql-operator namespace
// THEN the correct Helm overrides are returned
func TestAppendOverridesNoSecret(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	kvs, err := AppendOverrides(spi.NewFakeContext(fakeClient, nil, nil, false), "", "", "", []bom.KeyValue{{Key: "key1", Value: "value1"}})
	assert.Nil(t, err)
	assert.Len(t, kvs, 1)
	assert.Equal(t, bom.KeyValue{Key: "key1", Value: "value1"}, kvs[0])
}
