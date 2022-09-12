// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"testing"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	scheme2 "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profileDir = "../../../../manifests/profiles"

var pvc100Gi, _ = resource.ParseQuantity("100Gi")

// Test_FindStorageOverride tests the FindStorageOverride function
// GIVEN a call to FindStorageOverride
//  WHEN I call with a ComponentContext with different profiles and overrides
//  THEN the correct resource overrides or an error are returned
func Test_FindStorageOverride(t *testing.T) {

	tests := []struct {
		name             string
		description      string
		actualCR         vzapi.Verrazzano
		expectedOverride *ResourceRequestValues
		expectedErr      bool
	}{
		{
			name:        "TestProdNoOverrides",
			description: "Test storage override with empty CR",
			actualCR:    vzapi.Verrazzano{},
		},
		{
			name:        "TestProdEmptyDirOverride",
			description: "Test prod profile with empty dir storage override",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
				},
			},
			expectedOverride: &ResourceRequestValues{
				Storage: "",
			},
		},
		{
			name:        "TestProdPVCOverride",
			description: "Test prod profile with PVC storage override",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"}},
					VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "vmi"},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": pvc100Gi,
									},
								},
							},
						},
					},
				},
			},
			expectedOverride: &ResourceRequestValues{
				Storage: pvc100Gi.String(),
			},
		},
		{
			name:        "TestDevPVCOverride",
			description: "Test dev profile with PVC storage override",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Profile:             vzapi.Dev,
					DefaultVolumeSource: &corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"}},
					VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "vmi"},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": pvc100Gi,
									},
								},
							},
						},
					},
				},
			},
			expectedOverride: &ResourceRequestValues{
				Storage: pvc100Gi.String(),
			},
		},
		{
			name:        "TestDevUnsupportedVolumeSource",
			description: "Test dev profile with an unsupported default volume source",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Profile:             vzapi.Dev,
					DefaultVolumeSource: &corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{}},
					VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "vmi"},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": pvc100Gi,
									},
								},
							},
						},
					},
				},
			},
			expectedErr: true,
		},
		{
			name:        "TestDevMismatchedPVCClaimName",
			description: "Test dev profile with PVC default volume source and mismatched PVC claim name",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Profile:             vzapi.Dev,
					DefaultVolumeSource: &corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "foo"}},
					VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "vmi"},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": pvc100Gi,
									},
								},
							},
						},
					},
				},
			},
			expectedErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)

			fakeContext := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build(), &test.actualCR, nil, false, profileDir)

			override, err := FindStorageOverride(fakeContext.EffectiveCR())
			if test.expectedErr {
				a.Error(err)
			} else {
				a.NoError(err)
			}
			if test.expectedOverride != nil {
				if override == nil {
					a.FailNow("Expected returned override to not be nil")
				}
				a.Equal(*test.expectedOverride, *override)
			} else {
				a.Nil(override)
			}
		})
	}
}

// TestReassociateResources tests the VMO reassociateResources function
// GIVEN a VMO component
//  WHEN I call reassociateResources with a VMO service resource
//  THEN no error is returned and the VMO service contains expected Helm labels and annotations
func TestReassociateResources(t *testing.T) {
	// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme2.Scheme).WithObjects(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vmoComponentNamespace,
			Name:      vmoComponentName,
		},
	}).Build()
	err := ExportVMOHelmChart(spi.NewFakeContext(fakeClient, nil, nil, false))
	assert.NoError(t, err)
	err = ReassociateVMOResources(spi.NewFakeContext(fakeClient, nil, nil, false))
	assert.NoError(t, err)
	service := corev1.Service{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: vmoComponentNamespace, Name: vmoComponentName}, &service)
	assert.NoError(t, err)
	assert.Contains(t, service.Labels["app.kubernetes.io/managed-by"], "Helm")
	assert.Contains(t, service.Annotations["meta.helm.sh/release-name"], vmoComponentName)
	assert.Contains(t, service.Annotations["meta.helm.sh/release-namespace"], vmoComponentNamespace)
	assert.NotContains(t, service.Annotations["helm.sh/resource-policy"], "keep")
}

// TestExportVmoHelmChart tests the VMO exportVMOHelmChart function
// GIVEN a VMO component
//  WHEN I call exportVMOHelmChart with a VMO service resource
//  THEN no error is returned and the VMO service contains expected Helm labels and annotations
func TestExportVmoHelmChart(t *testing.T) {
	// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme2.Scheme).WithObjects(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vmoComponentNamespace,
			Name:      vmoComponentName,
		},
	}).Build()
	err := ExportVMOHelmChart(spi.NewFakeContext(fakeClient, nil, nil, false))
	assert.NoError(t, err)
	service := corev1.Service{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: vmoComponentNamespace, Name: vmoComponentName}, &service)
	assert.NoError(t, err)
	assert.Contains(t, service.Labels["app.kubernetes.io/managed-by"], "Helm")
	assert.Contains(t, service.Annotations["meta.helm.sh/release-name"], vmoComponentName)
	assert.Contains(t, service.Annotations["meta.helm.sh/release-namespace"], vmoComponentNamespace)
	assert.Contains(t, service.Annotations["helm.sh/resource-policy"], "keep")
}

func TestIsMultiNodeCluster(t *testing.T) {
	mkVZ := func(enabled bool) *vzapi.Verrazzano {
		return &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Elasticsearch: &vzapi.ElasticsearchComponent{
						Enabled: &enabled,
					},
				},
			},
		}
	}
	oneReplicaVZ := mkVZ(true)
	oneReplicaVZ.Spec.Components.Elasticsearch.ESInstallArgs = []vzapi.InstallArgs{
		{
			Name:  "nodes.master.replicas",
			Value: "1",
		},
	}
	multiNodeVZ := mkVZ(true)
	multiNodeVZ.Spec.Components.Elasticsearch.Nodes = []vzapi.OpenSearchNode{
		{
			Replicas: 3,
		},
	}
	var tests = []struct {
		name        string
		vz          *vzapi.Verrazzano
		isMultiNode bool
	}{
		{
			"not multinode when component nil",
			&vzapi.Verrazzano{},
			false,
		},
		{
			"not multinode when component disabled",
			mkVZ(false),
			false,
		},
		{
			"not multinode when 0 replicas",
			mkVZ(true),
			false,
		},
		{
			"not multinode when 1 replica",
			oneReplicaVZ,
			false,
		},
		{
			"multinode when 1+ replicas",
			multiNodeVZ,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := IsMultiNodeOpenSearch(tt.vz)
			assert.NoError(t, err)
			assert.Equal(t, tt.isMultiNode, m)
		})
	}
}

// Test_SetStorageSize tests the SetStorageSize function
func Test_SetStorageSize(t *testing.T) {
	// GIVEN an empty storage request
	// WHEN the storage size is set
	// THEN we expect the storage size to be the default value
	storageObject := &vmov1.Storage{}
	SetStorageSize(nil, storageObject)
	assert.Equal(t, defaultStorageSize, storageObject.Size)

	// GIVEN a populated storage request
	// WHEN the storage size is set
	// THEN we expect the storage size to be the value from the request
	const storageSize = "512Gi"
	storageRequest := &ResourceRequestValues{Storage: storageSize}
	SetStorageSize(storageRequest, storageObject)
	assert.Equal(t, storageSize, storageObject.Size)
}
