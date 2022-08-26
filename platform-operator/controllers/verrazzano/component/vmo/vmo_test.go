// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmo

import (
	"context"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testBomFilePath = "../../testdata/test_bom.json"

// TestIsVMOReady tests the isVMOReady function
// GIVEN a call to isVMOReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsVMOReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"k8s-app": ComponentName},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"k8s-app": ComponentName},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"k8s-app":           ComponentName,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        ComponentName + "-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
	).Build()
	assert.True(t, isVMOReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

// TestIsVMONotReady tests the isVMOReady function
// GIVEN a call to isVMOReady
//  WHEN the deployment object does not have enough replicas available
//  THEN true is returned
func TestIsVMONotReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentName,
			Labels:    map[string]string{"k8s-app": ComponentName},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 0,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	}).Build()
	assert.False(t, isVMOReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

// TestAppendVMOOverrides tests the appendVMOOverrides function
// GIVEN a call to appendVMOOverrides
//  WHEN I call with no extra kvs
//  THEN the correct KeyValue objects are returned and no error occurs
func TestAppendVMOOverrides(t *testing.T) {
	a := assert.New(t)
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ingress-nginx",
			Name:      "ingress-controller-ingress-nginx-controller",
		},
		Spec: corev1.ServiceSpec{
			ExternalIPs: []string{
				"nn.nn.nn.nn",
			},
		},
	}).Build()

	kvs, err := appendVMOOverrides(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false), "", "", "", []bom.KeyValue{})

	a.NoError(err)
	a.Len(kvs, 4)
	a.Contains(kvs, bom.KeyValue{
		Key:   "monitoringOperator.prometheusInitImage",
		Value: "ghcr.io/oracle/oraclelinux:7-slim",
	})
	a.Contains(kvs, bom.KeyValue{
		Key:   "monitoringOperator.esInitImage",
		Value: "ghcr.io/oracle/oraclelinux:7.8",
	})
	a.Contains(kvs, bom.KeyValue{
		Key:   "config.dnsSuffix",
		Value: "nn.nn.nn.nn.nip.io",
	})
	a.Contains(kvs, bom.KeyValue{
		Key:   "config.envName",
		Value: "default",
	})
}

// TestAppendVMOOverridesNoNGINX tests the appendVmoOverrides function
// GIVEN a call to appendVmoOverrides
//  WHEN I call with no extra kvs and NGINX is disabled
//  THEN the correct KeyValue objects are returned and no error occurs
func TestAppendVmoOverridesNoNGINX(t *testing.T) {
	a := assert.New(t)
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	enabled := false
	kvs, err := appendVMOOverrides(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Enabled: &enabled,
				},
			},
		},
	}, nil, false), "", "", "", []bom.KeyValue{})

	a.NoError(err)
	a.Len(kvs, 2)
	a.Contains(kvs, bom.KeyValue{
		Key:   "monitoringOperator.prometheusInitImage",
		Value: "ghcr.io/oracle/oraclelinux:7-slim",
	})
	a.Contains(kvs, bom.KeyValue{
		Key:   "monitoringOperator.esInitImage",
		Value: "ghcr.io/oracle/oraclelinux:7.8",
	})
}

// TestAppendVmoOverridesOidcAuthDisabled tests the appendVmoOverrides function
// GIVEN a call to appendVmoOverrides
//  WHEN the Auth Proxy component is disabled
//  THEN the key/value slice contains a helm override to disable OIDC auth in the VMO
func TestAppendVmoOverridesOidcAuthDisabled(t *testing.T) {
	a := assert.New(t)
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ingress-nginx",
			Name:      "ingress-controller-ingress-nginx-controller",
		},
		Spec: corev1.ServiceSpec{
			ExternalIPs: []string{
				"nn.nn.nn.nn",
			},
		},
	}).Build()

	var falseValue = false
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				AuthProxy: &vzapi.AuthProxyComponent{
					Enabled: &falseValue,
				},
			},
		},
	}
	kvs, err := appendVMOOverrides(spi.NewFakeContext(fakeClient, vz, nil, false), "", "", "", []bom.KeyValue{})

	a.NoError(err)
	a.Contains(kvs, bom.KeyValue{
		Key:   "monitoringOperator.oidcAuthEnabled",
		Value: "false",
	})
}

// erroringFakeClient wraps a k8s client and returns an error when Update is called
type erroringFakeClient struct {
	client.Client
}

// Update always returns an error - used to simulate an error updating a resource
func (e *erroringFakeClient) Update(_ context.Context, _ client.Object, _ ...client.UpdateOption) error {
	return errors.NewConflict(schema.GroupResource{}, "", nil)
}

// TestRetainPrometheusPersistentVolume tests the retainPrometheusPersistentVolume function
func TestRetainPrometheusPersistentVolume(t *testing.T) {
	a := assert.New(t)

	const (
		volumeName    = "pvc-5ab58a05-71f9-4f09-8911-a5c029f6305f"
		reclaimPolicy = corev1.PersistentVolumeReclaimDelete
	)

	// GIVEN a vmi-system-prometheus pvc and associated persistent volume
	//  WHEN we call retainPrometheusPersistentVolume
	//  THEN the persistent volume reclaim policy is set to "retain"
	//   AND the persistent volume has the expected labels
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      constants.VMISystemPrometheusVolumeClaim,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: volumeName,
			},
		},
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeReclaimPolicy: reclaimPolicy,
			},
		}).Build()

	err := retainPrometheusPersistentVolume(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false))
	a.NoError(err)

	pv := &corev1.PersistentVolume{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: volumeName}, pv)
	a.NoError(err)

	// validate that the expected labels are set and the reclaim policy is set to "retain"
	a.Equal(constants.PrometheusStorageLabelValue, pv.Labels[constants.StorageForLabel])
	a.Equal(string(reclaimPolicy), pv.Labels[constants.OldReclaimPolicyLabel])
	a.Equal(corev1.PersistentVolumeReclaimRetain, pv.Spec.PersistentVolumeReclaimPolicy)

	// GIVEN no vmi-system-prometheus pvc
	//  WHEN we call retainPrometheusPersistentVolume
	//  THEN no resources are changed and no error occurs
	fakeClient = fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	err = retainPrometheusPersistentVolume(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false))
	a.NoError(err)

	// GIVEN a vmi-system-prometheus pvc and no associated persistent volume
	//  WHEN we call retainPrometheusPersistentVolume
	//  THEN an error is returned
	fakeClient = fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      constants.VMISystemPrometheusVolumeClaim,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: volumeName,
			},
		}).Build()

	err = retainPrometheusPersistentVolume(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false))
	a.ErrorContains(err, "Failed fetching persistent volume")

	// GIVEN a vmi-system-prometheus pvc and associated persistent volume
	//  WHEN we call retainPrometheusPersistentVolume and an error occurs updating the persistent volume
	//  THEN an error is returned
	fakeClient = fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      constants.VMISystemPrometheusVolumeClaim,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: volumeName,
			},
		},
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeReclaimPolicy: reclaimPolicy,
			},
		}).Build()

	erroringClient := &erroringFakeClient{Client: fakeClient}
	err = retainPrometheusPersistentVolume(spi.NewFakeContext(erroringClient, &vzapi.Verrazzano{}, nil, false))
	a.ErrorContains(err, "Failed updating persistent volume")
}
