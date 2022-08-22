// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"context"
	"fmt"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profilesDir = "../../../../manifests/profiles"
const (
	profilesRelativePath = "../../../../manifests/profiles"
)

var crEnabled = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Keycloak: &vzapi.KeycloakComponent{
				Enabled: getBoolPtr(true),
			},
		},
	},
}

var mySQLSecret = v1.Secret{
	TypeMeta: metav1.TypeMeta{
		Kind: vzconst.SecretKind,
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      rootSecretName,
		Namespace: ComponentNamespace,
	},
	Immutable:  nil,
	Data:       nil,
	StringData: nil,
	Type:       "",
}

var pvc100Gi, _ = resource.ParseQuantity("100Gi")

const (
	minExpectedHelmOverridesCount = 3
	testBomFilePath               = "../../testdata/test_bom.json"
)

// TestAppendMySQLOverrides tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides
// WHEN I pass in an empty VZ CR
// THEN the correct overrides are returned
func TestAppendMySQLOverrides(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	vz := &vzapi.Verrazzano{}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, minExpectedHelmOverridesCount)
}

// TestAppendMySQLOverridesUpdate tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides
// WHEN the mysql secret exists in the keycloak namespace during install
// THEN the correct overrides are returned
func TestAppendMySQLOverridesUpdate(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	vz := &vzapi.Verrazzano{}
	secret := mySQLSecret
	secret.Data = map[string][]byte{}
	secret.Data[mySQLRootKey] = []byte("test-root-key")
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&secret).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, minExpectedHelmOverridesCount)
	assert.Equal(t, "test-root-key", bom.FindKV(kvs, helmRootPwd))

}

// TestAppendMySQLOverridesWithInstallArgs tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides
// WHEN I pass in an empty VZ CR with MySQL install args
// THEN the override key value pairs contain the install args
func TestAppendMySQLOverridesWithInstallArgs(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Keycloak: &vzapi.KeycloakComponent{
					MySQL: vzapi.MySQLComponent{
						MySQLInstallArgs: []vzapi.InstallArgs{
							{Name: "key", Value: "value"},
						},
					},
				},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1+minExpectedHelmOverridesCount)
	assert.Equal(t, "value", bom.FindKV(kvs, "key"))
	assert.NotEmpty(t, bom.FindKV(kvs, "image"))
}

// TestAppendMySQLOverridesDev tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides
// WHEN I pass in an VZ CR with the dev profile
// THEN the overrides contain the correct mysql persistence config
func TestAppendMySQLOverridesDev(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("dev"),
			DefaultVolumeSource: &v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1+minExpectedHelmOverridesCount)
	assert.NotEmpty(t, bom.FindKV(kvs, "image"))
}

// TestAppendMySQLOverridesDevWithPersistence tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides
// WHEN I pass in an VZ CR with the dev profile and persistence overrides
// THEN the overrides contain the correct mysql persistence config
func TestAppendMySQLOverridesDevWithPersistence(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("dev"),
			VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{{
				ObjectMeta: metav1.ObjectMeta{Name: "mysql"},
				Spec: v1.PersistentVolumeClaimSpec{
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							"storage": pvc100Gi,
						},
					},
				},
			}},
			Components: vzapi.ComponentSpec{
				Keycloak: &vzapi.KeycloakComponent{
					MySQL: vzapi.MySQLComponent{
						VolumeSource: &v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: "mysql"},
						},
					},
				},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1+minExpectedHelmOverridesCount)
	assert.Equal(t, "100Gi", bom.FindKV(kvs, "datadirVolumeClaimTemplate.resources.requests.storage"))
	assert.NotEmpty(t, bom.FindKV(kvs, "image"))
}

// TestAppendMySQLOverridesProd tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides
// WHEN I pass in an VZ CR with the prod profile
// THEN the overrides contain the correct mysql persistence config
func TestAppendMySQLOverridesProd(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("prod"),
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, minExpectedHelmOverridesCount)
	assert.NotEmpty(t, bom.FindKV(kvs, "image"))
}

// TestAppendMySQLOverridesProdWithOverrides tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides
// WHEN I pass in an VZ CR with the pred profile and a default volume source override
// THEN the overrides contain the correct mysql persistence config
func TestAppendMySQLOverridesProdWithOverrides(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("prod"),
			DefaultVolumeSource: &v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: "globalOverride",
				},
			},
			VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{{
				ObjectMeta: metav1.ObjectMeta{Name: "globalOverride"},
				Spec: v1.PersistentVolumeClaimSpec{
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							"storage": pvc100Gi,
						},
					},
				},
			}},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1+minExpectedHelmOverridesCount)
	assert.Equal(t, "100Gi", bom.FindKV(kvs, "datadirVolumeClaimTemplate.resources.requests.storage"))
	assert.NotEmpty(t, bom.FindKV(kvs, "image"))
}

// TestPreUpgradeProdProfile tests the preUpgrade function
// GIVEN a call to preUpgrade during upgrade of a prod profile cluster
// WHEN the PV, PVC and existing deployment exist
// THEN the PV is released and the deployment removed
func TestPreUpgradeProdProfile(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("dev"),
			VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{{
				ObjectMeta: metav1.ObjectMeta{Name: "mysql"},
				Spec: v1.PersistentVolumeClaimSpec{
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							"storage": pvc100Gi,
						},
					},
				},
			}},
			Components: vzapi.ComponentSpec{
				Keycloak: &vzapi.KeycloakComponent{
					MySQL: vzapi.MySQLComponent{
						VolumeSource: &v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: "mysql"},
						},
					},
				},
			},
		},
	}
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: DeploymentPersistentVolumeClaim}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, pvc *v1.PersistentVolumeClaim) error {
			pvc.Spec.VolumeName = "volumeName"
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: "volumeName"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, pv *v1.PersistentVolume) error {
			pv.Name = "volumeName"
			pv.Spec.PersistentVolumeReclaimPolicy = v1.PersistentVolumeReclaimDelete
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, pv *v1.PersistentVolume, opts ...client.UpdateOption) error {
			assert.Equal(t, 2, len(pv.Labels))
			assert.Equal(t, v1.PersistentVolumeReclaimRetain, pv.Spec.PersistentVolumeReclaimPolicy)
			return nil
		})
	mock.EXPECT().
		Delete(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, deployment *appsv1.Deployment, opts ...client.DeleteOption) error {
			assert.Equal(t, ComponentName, deployment.Name)
			return nil
		})
	mock.EXPECT().
		Delete(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, pvc *v1.PersistentVolumeClaim, opts ...client.DeleteOption) error {
			assert.Equal(t, ComponentName, pvc.Name)
			assert.Equal(t, ComponentNamespace, pvc.Namespace)
			return nil
		})
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *v1.PersistentVolumeList, opts ...client.ListOption) error {
			pv := v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "volumeName",
				},
				Spec: v1.PersistentVolumeSpec{
					ClaimRef: &v1.ObjectReference{
						Namespace: ComponentNamespace,
						Name:      DeploymentPersistentVolumeClaim,
					},
					PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimRetain,
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeReleased,
				},
			}
			list.Items = []v1.PersistentVolume{pv}
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, pv *v1.PersistentVolume, opts ...client.UpdateOption) error {
			assert.Nil(t, pv.Spec.ClaimRef)
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: statefulsetClaimName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, pvc *v1.PersistentVolumeClaim) error {
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, pvc *v1.PersistentVolumeClaim, opts ...client.UpdateOption) error {
			assert.Equal(t, statefulsetClaimName, pvc.Name)
			assert.Equal(t, "volumeName", pvc.Spec.VolumeName)
			return nil
		})

	ctx := spi.NewFakeContext(mock, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
	err := preUpgrade(ctx)
	assert.NoError(t, err)
}

// TestPreUpgradeDevProfile tests the preUpgrade function
// GIVEN a call to preUpgrade during upgrade of a dev profile cluster
// WHEN the pre upgrade executes
// THEN the pv/pvc steps are skipped
func TestPreUpgradeDevProfile(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("dev"),
			DefaultVolumeSource: &v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		},
	}
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	ctx := spi.NewFakeContext(mock, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
	err := preUpgrade(ctx)
	assert.NoError(t, err)
}

// TestPreUpgradeForStatefulSetMySQL tests the preUpgrade function
// GIVEN a call to preUpgrade during upgrade
// WHEN the resource is a statefulset rather than the old deployment
// THEN the PV reassignment steps are skipped
func TestPreUpgradeForStatefulSetMySQL(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("dev"),
			VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{{
				ObjectMeta: metav1.ObjectMeta{Name: "mysql"},
				Spec: v1.PersistentVolumeClaimSpec{
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							"storage": pvc100Gi,
						},
					},
				},
			}},
			Components: vzapi.ComponentSpec{
				Keycloak: &vzapi.KeycloakComponent{
					MySQL: vzapi.MySQLComponent{
						VolumeSource: &v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: "mysql"},
						},
					},
				},
			},
		},
	}
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: DeploymentPersistentVolumeClaim}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, pvc *v1.PersistentVolumeClaim) error {
			return errors.NewNotFound(schema.GroupResource{Group: "v1", Resource: "PersistentVolumeClaim"}, name.Name)
		})
	mock.EXPECT().
		Delete(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, deployment *appsv1.Deployment, opts ...client.DeleteOption) error {
			return errors.NewNotFound(schema.GroupResource{Group: "apps", Resource: "Deployment"}, deployment.Name)
		})
	mock.EXPECT().
		Delete(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, pvc *v1.PersistentVolumeClaim, opts ...client.DeleteOption) error {
			return errors.NewNotFound(schema.GroupResource{Group: "v1", Resource: "PersistentVolumeClaim"}, pvc.Name)
		})
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *v1.PersistentVolumeList, opts ...client.ListOption) error {
			list.Items = []v1.PersistentVolume{}
			return nil
		})

	ctx := spi.NewFakeContext(mock, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
	err := preUpgrade(ctx)
	assert.NoError(t, err)
}

// TestPostUpgrade tests the postUpgrade function
// GIVEN a call to postUpgrade during upgrade of a prod profile
// WHEN the PV for the stateful set exists
// THEN the PV is updated with the original reclaim policy
func TestPostUpgradeProdProfile(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("dev"),
			VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{{
				ObjectMeta: metav1.ObjectMeta{Name: "mysql"},
				Spec: v1.PersistentVolumeClaimSpec{
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							"storage": pvc100Gi,
						},
					},
				},
			}},
			Components: vzapi.ComponentSpec{
				Keycloak: &vzapi.KeycloakComponent{
					MySQL: vzapi.MySQLComponent{
						VolumeSource: &v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: "mysql"},
						},
					},
				},
			},
		},
	}
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *v1.PersistentVolumeList, opts ...client.ListOption) error {
			pv := v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "volumeName",
					Labels: map[string]string{vzconst.OldReclaimPolicyLabel: string(v1.PersistentVolumeReclaimDelete)},
				},
				Spec: v1.PersistentVolumeSpec{
					ClaimRef: &v1.ObjectReference{
						Namespace: ComponentNamespace,
						Name:      statefulsetClaimName,
					},
					PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimRetain,
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeBound,
				},
			}
			list.Items = []v1.PersistentVolume{pv}
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, pv *v1.PersistentVolume, opts ...client.UpdateOption) error {
			assert.Equal(t, v1.PersistentVolumeReclaimDelete, pv.Spec.PersistentVolumeReclaimPolicy)
			_, ok := pv.Labels[vzconst.OldReclaimPolicyLabel]
			assert.False(t, ok)
			return nil
		})

	ctx := spi.NewFakeContext(mock, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
	err := postUpgrade(ctx)
	assert.NoError(t, err)
}

// TestPostUpgrade tests the postUpgrade function
// GIVEN a call to postUpgrade during upgrade of a dev profile
// WHEN the code executes
// THEN the PV update is skipped
func TestPostUpgradeDevProfile(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("dev"),
			DefaultVolumeSource: &v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		},
	}
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	ctx := spi.NewFakeContext(mock, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
	err := postUpgrade(ctx)
	assert.NoError(t, err)
}

// TestAppendMySQLOverridesUpgradeDevProfile tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides during upgrade for a dev profile cluster
// WHEN I pass in a VZ CR
// THEN the correct overrides are returned
func TestAppendMySQLOverridesUpgradeDevProfile(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("dev"),
			DefaultVolumeSource: &v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		},
	}
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: rootSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = rootSecretName
			secret.Data = map[string][]byte{}
			secret.Data[mySQLRootKey] = []byte("test-root-key")
			return nil
		})
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *v1.PersistentVolumeList, opts ...client.ListOption) error {
			return nil
		}).AnyTimes()
	ctx := spi.NewFakeContext(mock, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1+minExpectedHelmOverridesCount)
	assert.Equal(t, "test-root-key", bom.FindKV(kvs, helmRootPwd))
	assert.NotEmpty(t, bom.FindKV(kvs, "image"))
}

// TestAppendMySQLOverridesUpgradeProdProfile tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides during upgrade for a prod profile cluster
// WHEN I pass in a VZ CR
// THEN the correct overrides are returned
func TestAppendMySQLOverridesUpgradeProdProfile(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("dev"),
			VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{{
				ObjectMeta: metav1.ObjectMeta{Name: "mysql"},
				Spec: v1.PersistentVolumeClaimSpec{
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							"storage": pvc100Gi,
						},
					},
				},
			}},
			Components: vzapi.ComponentSpec{
				Keycloak: &vzapi.KeycloakComponent{
					MySQL: vzapi.MySQLComponent{
						VolumeSource: &v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: "mysql"},
						},
					},
				},
			},
		},
	}
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: rootSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = rootSecretName
			secret.Data = map[string][]byte{}
			secret.Data[mySQLRootKey] = []byte("test-root-key")
			return nil
		})
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *v1.PersistentVolumeList, opts ...client.ListOption) error {
			return nil
		}).AnyTimes()
	ctx := spi.NewFakeContext(mock, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1+minExpectedHelmOverridesCount)
	assert.Equal(t, "test-root-key", bom.FindKV(kvs, helmRootPwd))
	assert.NotEmpty(t, bom.FindKV(kvs, "image"))
}

// TestIsMySQLReady tests the isMySQLReady function
// GIVEN a call to isMySQLReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsMySQLReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"app": ComponentName},
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": ComponentName},
				},
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas:   1,
				Replicas:        1,
				UpdatedReplicas: 1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      fmt.Sprintf("%s-router", ComponentName),
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
				Name:        "mysql-router-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName + "-0",
				Labels: map[string]string{
					"controller-revision-hash": ComponentName + "-f97fd59d8",
					"pod-template-hash":        "95d8c5d96",
					"app":                      ComponentName,
				},
			},
		},
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName + "-router-jfkdlf",
				Labels: map[string]string{
					"controller-revision-hash": ComponentName + "-f97fd59d8",
					"pod-template-hash":        "95d8c5d96",
					"app":                      ComponentName,
				},
			},
		},
		&appsv1.ControllerRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ComponentName + "-f97fd59d8",
				Namespace: ComponentNamespace,
			},
			Revision: 1,
		},
	).Build()
	servers := []byte(`{"serverInstances": "1"}`)
	routers := []byte(`{"routerInstances": "1"}`)

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("prod"),
			Components: vzapi.ComponentSpec{
				Keycloak: &vzapi.KeycloakComponent{
					MySQL: vzapi.MySQLComponent{
						InstallOverrides: vzapi.InstallOverrides{
							ValueOverrides: []vzapi.Overrides{
								{
									Values: &apiextensionsv1.JSON{
										Raw: servers,
									},
								},
								{
									Values: &apiextensionsv1.JSON{
										Raw: routers,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	assert.True(t, isMySQLReady(spi.NewFakeContext(fakeClient, vz, false)))
}

// TestIsMySQLNotReady tests the isMySQLReady function
// GIVEN a call to isMySQLReady
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
func TestIsMySQLNotReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"app": ComponentName},
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": ComponentName},
				},
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas:   1,
				Replicas:        1,
				UpdatedReplicas: 1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      fmt.Sprintf("%s-router", ComponentName),
				Labels:    map[string]string{"app": ComponentName},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": ComponentName},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        "mysql-router-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName + "-0",
				Labels: map[string]string{
					"controller-revision-hash": ComponentName + "-f97fd59d8",
					"pod-template-hash":        "95d8c5d96",
					"app":                      ComponentName,
				},
			},
		},
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName + "-router-jfkdlf",
				Labels: map[string]string{
					"controller-revision-hash": ComponentName + "-f97fd59d8",
					"pod-template-hash":        "95d8c5d96",
					"app":                      ComponentName,
				},
			},
		},
		&appsv1.ControllerRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ComponentName + "-f97fd59d8",
				Namespace: ComponentNamespace,
			},
			Revision: 1,
		},
	).Build()
	servers := []byte(`{"serverInstances": "1"}`)
	routers := []byte(`{"routerInstances": "1"}`)

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("prod"),
			Components: vzapi.ComponentSpec{
				Keycloak: &vzapi.KeycloakComponent{
					MySQL: vzapi.MySQLComponent{
						InstallOverrides: vzapi.InstallOverrides{
							ValueOverrides: []vzapi.Overrides{
								{
									Values: &apiextensionsv1.JSON{
										Raw: servers,
									},
								},
								{
									Values: &apiextensionsv1.JSON{
										Raw: routers,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	assert.False(t, isMySQLReady(spi.NewFakeContext(fakeClient, vz, false)))
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component is nil
//  THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilKeycloak tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component is nil
//  THEN true is returned
func TestIsEnabledNilKeycloak(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component enabled is nil
//  THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component is explicitly enabled
//  THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component is explicitly disabled
//  THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledManagedClusterProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak enabled flag is nil and managed cluster profile
//  THEN false is returned
func TestIsEnabledManagedClusterProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	cr.Spec.Profile = vzapi.ManagedCluster
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledProdProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak enabled flag is nil and prod profile
//  THEN false is returned
func TestIsEnabledProdProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	cr.Spec.Profile = vzapi.Prod
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledDevProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak enabled flag is nil and dev profile
//  THEN false is returned
func TestIsEnabledDevProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	cr.Spec.Profile = vzapi.Dev
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

func getBoolPtr(b bool) *bool {
	return &b
}
