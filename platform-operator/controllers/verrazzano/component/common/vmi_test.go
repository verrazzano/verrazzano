// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzbeta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	scheme2 "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profileDir = "../../../../manifests/profiles"

var pvc100Gi, _ = resource.ParseQuantity("100Gi")
var pvc0Gi, _ = resource.ParseQuantity("0Gi")

// TestVMI tests the Multiple VMI functions
func TestVMI(t *testing.T) {
	b := true
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
					Components:          vzapi.ComponentSpec{Istio: &vzapi.IstioComponent{Enabled: &b, InjectionEnabled: &b}, DNS: &vzapi.DNSComponent{Wildcard: &vzapi.Wildcard{Domain: APIGroupRancherManagement}, InstallOverrides: vzapi.InstallOverrides{MonitorChanges: &b}}},
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
			name:        "TestDevNoSettingOverride",
			description: "Test dev profile with no storage setting",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Profile:             vzapi.Dev,
					DefaultVolumeSource: &corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"}},
					VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "vmi"},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.ResourceRequirements{},
							},
						},
					},
				},
			},
			expectedOverride: &ResourceRequestValues{
				Storage: pvc0Gi.String(),
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
			cli := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects().Build()
			fakeContext := spi.NewFakeContext(cli, &test.actualCR, nil, false, profileDir)
			a.False(IsVMISecretReady(fakeContext))
			a.False(IsGrafanaAdminSecretReady(fakeContext))
			a.Nil(CreateAndLabelVMINamespaces(fakeContext))
			a.Error(CreateOrUpdateVMI(fakeContext, nil))
			fakeContext.EffectiveCR().Spec.Components.DNS = &vzapi.DNSComponent{Wildcard: &vzapi.Wildcard{Domain: "verrazzano"}}
			fakeContext.EffectiveCR().Spec.Components.Grafana = &vzapi.GrafanaComponent{Database: &vzapi.DatabaseInfo{Name: "name", Host: "host"}}
			fakeContext.EffectiveCR().Spec.Components.DNS.OCI = &vzapi.OCI{DNSZoneName: "ind"}
			fakeContext.EffectiveCR().Spec.Components.DNS.Wildcard = nil
			a.Error(CreateOrUpdateVMI(fakeContext, nil))
			a.NoError(EnsureBackupSecret(cli))
			a.NoError(EnsureGrafanaAdminSecret(cli))
			a.NoError(EnsureVMISecret(cli))
			a.NotNil(NewVMI())
			a.NotNil(EnsureGrafanaDatabaseSecret(fakeContext))
			a.True(IsVMISecretReady(fakeContext))
			a.True(IsGrafanaAdminSecretReady(fakeContext))
			err := CompareStorageOverrides(fakeContext.ActualCR(), fakeContext.EffectiveCR(), "")
			if test.expectedErr {
				a.Error(err)
			} else {
				a.NoError(err)
			}
			override, err := FindStorageOverride(fakeContext.EffectiveCR())
			if test.expectedErr {
				a.Error(err)
			} else {
				a.NoError(err)
			}
			a.Equal(test.expectedOverride, override)
		})
	}
}

// TestStorageOverrideBeta1 tests the StorageOverrides functions for V1Beta1
// case mentioned to description
// expected error with multiple cases
func TestStorageOverrideBeta1(t *testing.T) {
	tests := []struct {
		name             string
		description      string
		actualCR         vzbeta1.Verrazzano
		expectedOverride *ResourceRequestValues
		expectedErr      bool
	}{
		{
			name:        "TestProdNoOverrides",
			description: "Test storage override with empty CR",
			actualCR:    vzbeta1.Verrazzano{},
		},
		{
			name:        "TestProdEmptyDirOverride",
			description: "Test prod profile with empty dir storage override",
			actualCR: vzbeta1.Verrazzano{
				Spec: vzbeta1.VerrazzanoSpec{
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
			actualCR: vzbeta1.Verrazzano{
				Spec: vzbeta1.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"}},
					VolumeClaimSpecTemplates: []vzbeta1.VolumeClaimSpecTemplate{
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
			actualCR: vzbeta1.Verrazzano{
				Spec: vzbeta1.VerrazzanoSpec{
					Profile:             vzbeta1.Dev,
					DefaultVolumeSource: &corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"}},
					VolumeClaimSpecTemplates: []vzbeta1.VolumeClaimSpecTemplate{
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
			name:        "TestDevNoSettingOverride",
			description: "Test dev profile with no storage setting",
			actualCR: vzbeta1.Verrazzano{
				Spec: vzbeta1.VerrazzanoSpec{
					Profile:             vzbeta1.Dev,
					DefaultVolumeSource: &corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"}},
					VolumeClaimSpecTemplates: []vzbeta1.VolumeClaimSpecTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "vmi"},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.ResourceRequirements{},
							},
						},
					},
				},
			},
			expectedOverride: &ResourceRequestValues{
				Storage: pvc0Gi.String(),
			},
		},
		{
			name:        "TestDevUnsupportedVolumeSource",
			description: "Test dev profile with an unsupported default volume source",
			actualCR: vzbeta1.Verrazzano{
				Spec: vzbeta1.VerrazzanoSpec{
					Profile:             vzbeta1.Dev,
					DefaultVolumeSource: &corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{}},
					VolumeClaimSpecTemplates: []vzbeta1.VolumeClaimSpecTemplate{
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
			actualCR: vzbeta1.Verrazzano{
				Spec: vzbeta1.VerrazzanoSpec{
					Profile:             vzbeta1.Dev,
					DefaultVolumeSource: &corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "foo"}},
					VolumeClaimSpecTemplates: []vzbeta1.VolumeClaimSpecTemplate{
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
			fakeContext := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build(), nil, &test.actualCR, false, profileDir)
			err := CompareStorageOverridesV1Beta1(fakeContext.ActualCRV1Beta1(), fakeContext.EffectiveCRV1Beta1(), "")
			if test.expectedErr {
				a.Error(err)
			} else {
				a.NoError(err)
			}
			override, err := FindStorageOverrideV1Beta1(fakeContext.EffectiveCRV1Beta1())
			if test.expectedErr {
				a.Error(err)
			} else {
				a.NoError(err)
			}
			a.Equal(test.expectedOverride, override)
		})
	}
}

// TestReassociateResources tests the VMO reassociateResources function
// GIVEN a VMO component
//
//	WHEN I call reassociateResources with a VMO service resource
//	THEN no error is returned and the VMO service contains expected Helm labels and annotations
func TestReassociateResources(t *testing.T) {
	// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme2.Scheme).WithObjects(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: VMOComponentNamespace,
			Name:      VMOComponentName,
		},
	}).Build()
	err := ExportVMOHelmChart(spi.NewFakeContext(fakeClient, nil, nil, false))
	assert.NoError(t, err)
	err = ReassociateVMOResources(spi.NewFakeContext(fakeClient, nil, nil, false))
	assert.NoError(t, err)
	service := corev1.Service{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: VMOComponentNamespace, Name: VMOComponentName}, &service)
	assert.NoError(t, err)
	assert.Contains(t, service.Labels["app.kubernetes.io/managed-by"], "Helm")
	assert.Contains(t, service.Annotations["meta.helm.sh/release-name"], VMOComponentName)
	assert.Contains(t, service.Annotations["meta.helm.sh/release-namespace"], VMOComponentNamespace)
	assert.NotContains(t, service.Annotations["helm.sh/resource-policy"], "keep")
}

// TestExportVmoHelmChart tests the VMO exportVMOHelmChart function
// GIVEN a VMO component
//
//	WHEN I call exportVMOHelmChart with a VMO service resource
//	THEN no error is returned and the VMO service contains expected Helm labels and annotations
func TestExportVmoHelmChart(t *testing.T) {
	// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme2.Scheme).WithObjects(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: VMOComponentNamespace,
			Name:      VMOComponentName,
		},
	}).Build()
	err := ExportVMOHelmChart(spi.NewFakeContext(fakeClient, nil, nil, false))
	assert.NoError(t, err)
	service := corev1.Service{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: VMOComponentNamespace, Name: VMOComponentName}, &service)
	assert.NoError(t, err)
	assert.Contains(t, service.Labels["app.kubernetes.io/managed-by"], "Helm")
	assert.Contains(t, service.Annotations["meta.helm.sh/release-name"], VMOComponentName)
	assert.Contains(t, service.Annotations["meta.helm.sh/release-namespace"], VMOComponentNamespace)
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
			Replicas: Int32Ptr(3),
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

// TestGetVMOManagedDeployments tests GetVMOManagedDeployments
// GIVEN both VMO managed and non-VMO managed deployments exist
// WHEN GetVMOManagedDeployments is called
// THEN only the VMO managed deployment(s) are returned
func TestGetVMOManagedDeployments(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	vmoMgd := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vmoMgd", Namespace: "somens", Labels: vmoManagedLabels},
	}
	nonVMOMgd := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nonVMOMgd", Namespace: "somens"},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&vmoMgd, &nonVMOMgd).
		Build()

	list, err := GetVMOManagedDeployments(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
	assert.Len(t, list.Items, 1)
	assert.Equal(t, vmoMgd.Name, list.Items[0].Name)
	assert.Equal(t, vmoMgd.Namespace, list.Items[0].Namespace)
}

// TestGetVMOManagedIngresses tests GetVMOManagedIngresses
// GIVEN both VMO managed and non-VMO managed ingresses exist
// WHEN GetVMOManagedIngresses is called
// THEN only the VMO managed ingress(es) are returned
func TestGetVMOManagedIngresses(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = netv1.AddToScheme(scheme)

	vmoMgd := netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vmoMgd", Namespace: "somens", Labels: vmoManagedLabels},
	}
	nonVMOMgd := netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "otherobj", Namespace: "somens"},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&vmoMgd, &nonVMOMgd).
		Build()

	list, err := GetVMOManagedIngresses(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
	assert.Len(t, list.Items, 1)
	assert.Equal(t, vmoMgd.Name, list.Items[0].Name)
	assert.Equal(t, nonVMOMgd.Namespace, list.Items[0].Namespace)
}

// TestGetVMOManagedServices tests GetVMOManagedServices
// GIVEN both VMO managed and non-VMO managed services exist
// WHEN GetVMOManagedServices is called
// THEN only the VMO managed service(s) are returned
func TestGetVMOManagedServices(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	vmoMgd := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vmoMgd", Namespace: "somens", Labels: vmoManagedLabels},
	}
	nonVMOMgd := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "otherobj", Namespace: "somens"},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&vmoMgd, &nonVMOMgd).
		Build()

	list, err := GetVMOManagedServices(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
	assert.Len(t, list.Items, 1)
	assert.Equal(t, vmoMgd.Name, list.Items[0].Name)
	assert.Equal(t, nonVMOMgd.Namespace, list.Items[0].Namespace)
}

// TestGetVMOManagedStatefulsets tests GetVMOManagedStatefulsets
// GIVEN both VMO managed and non-VMO managed statefulsets exist
// WHEN GetVMOManagedStatefulsets is called
// THEN only the VMO managed statefulset(s) are returned
func TestGetVMOManagedStatefulsets(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	vmoMgd := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vmoMgd", Namespace: "somens", Labels: vmoManagedLabels},
	}
	nonVMOMgd := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "otherobj", Namespace: "somens"},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&vmoMgd, &nonVMOMgd).
		Build()

	list, err := GetVMOManagedStatefulsets(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
	assert.Len(t, list.Items, 1)
	assert.Equal(t, vmoMgd.Name, list.Items[0].Name)
	assert.Equal(t, nonVMOMgd.Namespace, list.Items[0].Namespace)
}

// TestDeleteVMI tests DeleteVMI
// GIVEN a system VMI exists in the cluster
//
//	WHEN DeleteVMI is called
//	THEN the VMI is deleted
//
// GIVEN a system VMI does NOT in the cluster
//
//	WHEN DeleteVMI is called
//	THEN no error is returned
func TestDeleteVMI(t *testing.T) {
	systemVMI := vmov1.VerrazzanoMonitoringInstance{
		ObjectMeta: metav1.ObjectMeta{Name: VMIName, Namespace: constants.VerrazzanoSystemNamespace},
	}
	otherVMISystemNS := vmov1.VerrazzanoMonitoringInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "otherVMI", Namespace: constants.VerrazzanoSystemNamespace},
	}
	otherVMIOtherNS := vmov1.VerrazzanoMonitoringInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "otherVMI", Namespace: "otherNS"},
	}

	tests := []struct {
		name          string
		vmis          []client.Object
		expectDeleted []vmov1.VerrazzanoMonitoringInstance
	}{
		{"Only System VMI exists",
			[]client.Object{&systemVMI},
			[]vmov1.VerrazzanoMonitoringInstance{systemVMI}},
		{"Only non-system VMIs exist",
			[]client.Object{&otherVMISystemNS, &otherVMIOtherNS},
			[]vmov1.VerrazzanoMonitoringInstance{}},
		{"System VMI and non-system VMIs exist",
			[]client.Object{&systemVMI, &otherVMISystemNS, &otherVMIOtherNS},
			[]vmov1.VerrazzanoMonitoringInstance{systemVMI}},
		{"No VMIs exist", []client.Object{}, []vmov1.VerrazzanoMonitoringInstance{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
			// for the Component interface hook
			scheme := runtime.NewScheme()
			_ = vmov1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.vmis...).Build()
			err := DeleteVMI(spi.NewFakeContext(fakeClient, nil, nil, false))
			for _, expectDeleted := range tt.expectDeleted {
				obj := vmov1.VerrazzanoMonitoringInstance{}
				err := fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: expectDeleted.GetNamespace(), Name: expectDeleted.GetName()}, &obj)
				assert.True(t, errors.IsNotFound(err))
			}
			assert.NoError(t, err)
		})
	}
}
