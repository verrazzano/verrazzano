// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	profilesDir  = "../../../../manifests/profiles"
	notDepFound  = "not-deployment-found"
	notPVCDelete = "not-pvc-deleted"
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
		Name:      rootSec,
		Namespace: ComponentNamespace,
	},
	Immutable:  nil,
	Data:       nil,
	StringData: nil,
	Type:       "",
}

var pvc100Gi, _ = resource.ParseQuantity("100Gi")

const (
	minExpectedHelmOverridesCount = 4
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
	ctx := spi.NewFakeContext(fakeClient, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 4+minExpectedHelmOverridesCount)
	assert.NotEmpty(t, bom.FindKV(kvs, "initdbScripts.create-db\\.sh"))
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
	ctx := spi.NewFakeContext(fakeClient, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 4+minExpectedHelmOverridesCount)
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
	ctx := spi.NewFakeContext(fakeClient, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 5+minExpectedHelmOverridesCount)
	assert.Equal(t, "value", bom.FindKV(kvs, "key"))
	assert.NotEmpty(t, bom.FindKV(kvs, "initdbScripts.create-db\\.sh"))
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
	ctx := spi.NewFakeContext(fakeClient, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 4+minExpectedHelmOverridesCount)
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
	ctx := spi.NewFakeContext(fakeClient, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 5+minExpectedHelmOverridesCount)
	assert.Equal(t, "100Gi", bom.FindKV(kvs, "datadirVolumeClaimTemplate.resources.requests.storage"))
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
	ctx := spi.NewFakeContext(fakeClient, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 4+minExpectedHelmOverridesCount)
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
	ctx := spi.NewFakeContext(fakeClient, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 5+minExpectedHelmOverridesCount)
	assert.Equal(t, "100Gi", bom.FindKV(kvs, "datadirVolumeClaimTemplate.resources.requests.storage"))
}

// TestPreUpgradeProdProfile tests the preUpgrade function
// GIVEN a call to preUpgrade during upgrade of a prod profile cluster
// WHEN the PV, PVC and existing deployment exist
// THEN the PV is released and the deployment removed
func TestPreUpgradeProdProfile(t *testing.T) {
	initUnitTesting()
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

	// isLegacyDatabaseUpgrade
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *v1.Secret) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			dep.Data = map[string][]byte{notDepFound: []byte("false")}
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *appsv1.Deployment) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *v1.Secret) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, secret *v1.Secret, opts ...client.UpdateOption) error {
			assert.Equal(t, dbMigrationSecret, secret.Name)
			return nil
		})
	// handleLegacyDatabasePreUpgrade

	// isDatabaseMigrationStageCompleted
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *v1.Secret) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			dep.Data = map[string][]byte{notPVCDelete: []byte("false")}
			return nil
		})
	// get PVC
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: DeploymentPersistentVolumeClaim}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, pvc *v1.PersistentVolumeClaim) error {
			pvc.Spec.VolumeName = "pv-name"
			return nil
		})
	// RetainPersistentVolume
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: "pv-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, pv *v1.PersistentVolume) error {
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, pv *v1.PersistentVolume, opts ...client.UpdateOption) error {
			assert.Equal(t, 2, len(pv.Labels))
			assert.Equal(t, v1.PersistentVolumeReclaimRetain, pv.Spec.PersistentVolumeReclaimPolicy)
			return nil
		})
	// deleting the deployment
	mock.EXPECT().
		Delete(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, deployment *appsv1.Deployment, opts ...client.DeleteOption) error {
			assert.Equal(t, ComponentName, deployment.Name)
			return nil
		})
	// DeleteExistingVolumeClaim
	mock.EXPECT().
		Delete(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, pvc *v1.PersistentVolumeClaim, opts ...client.DeleteOption) error {
			assert.Equal(t, ComponentName, pvc.Name)
			assert.Equal(t, ComponentNamespace, pvc.Namespace)
			return nil
		})
	// updateDBMigrationInProgressSecret(ctx, pvcDeletedStage)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = name.Name
			secret.Namespace = name.Namespace
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, secret *v1.Secret, opts ...client.UpdateOption) error {
			assert.Equal(t, dbMigrationSecret, secret.Name)
			return nil
		})
	// UpdateExistingVolumeClaims
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = name.Name
			secret.Namespace = name.Namespace
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
	// createPVCFromPV
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: legacyDBDumpClaim}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, pvc *v1.PersistentVolumeClaim) error {
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, pvc *v1.PersistentVolumeClaim, opts ...client.UpdateOption) error {
			assert.Equal(t, legacyDBDumpClaim, pvc.Name)
			assert.Equal(t, "volumeName", pvc.Spec.VolumeName)
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = name.Name
			secret.Namespace = name.Namespace
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, secret *v1.Secret, opts ...client.UpdateOption) error {
			assert.Equal(t, dbMigrationSecret, secret.Name)
			return nil
		})
	// Expect a calls to get and create the load job
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbLoadJobName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, job *batchv1.Job) error {
			return errors.NewNotFound(schema.GroupResource{Group: "batchv1", Resource: "Job"}, name.Name)
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: secretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = secretName
			secret.Data = map[string][]byte{}
			secret.Data["mysql-root-password"] = []byte("test-root-key")
			return nil
		})
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, job *batchv1.Job, opts ...client.CreateOption) error {
			return nil
		})
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *v1.PodList, opts ...client.ListOption) error {
			pod := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: dbLoadJobName,
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			}
			list.Items = []v1.Pod{pod}
			return nil
		})

	ctx := spi.NewFakeContext(mock, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
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

	// isLegacyDatabaseUpgrade
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *v1.Secret) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			dep.Data = map[string][]byte{notDepFound: []byte("false")}
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *appsv1.Deployment) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *v1.Secret) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, secret *v1.Secret, opts ...client.UpdateOption) error {
			assert.Equal(t, dbMigrationSecret, secret.Name)
			return nil
		})
	// handleLegacyDatabasePreUpgrade

	// isDatabaseMigrationStageCompleted
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *v1.Secret) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			dep.Data = map[string][]byte{notPVCDelete: []byte("false")}
			return nil
		})
	// get PVC
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: DeploymentPersistentVolumeClaim}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, pvc *v1.PersistentVolumeClaim) error {
			return errors.NewNotFound(schema.GroupResource{Group: "v1", Resource: "PersistenceVolumeClaim"}, name.Name)
		})

	ctx := spi.NewFakeContext(mock, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
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

	// isLegacyDatabaseUpgrade
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *v1.Secret) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			dep.Data = map[string][]byte{notDepFound: []byte("false")}
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *appsv1.Deployment) error {
			return errors.NewNotFound(schema.GroupResource{Group: "appsv1", Resource: "Deployment"}, name.Name)
		})

	ctx := spi.NewFakeContext(mock, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
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

	// cleanupDbMigrationJob
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbLoadJobName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, job *batchv1.Job) error {
			job.Name = dbLoadJobName
			return nil
		})
	mock.EXPECT().
		Delete(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, job *batchv1.Job, opts ...client.DeleteOption) error {
			assert.Equal(t, dbLoadJobName, job.Name)
			return nil
		})
	// deleteDbMigrationSecret
	mock.EXPECT().
		Delete(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, secret *v1.Secret, opts ...client.DeleteOption) error {
			assert.Equal(t, dbMigrationSecret, secret.Name)
			return nil
		})
	// ResetVolumeReclaimPolicy
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
						Name:      legacyDBDumpClaim,
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

	ctx := spi.NewFakeContext(mock, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
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

	// cleanupDbMigrationJob
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbLoadJobName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, job *batchv1.Job) error {
			return errors.NewNotFound(schema.GroupResource{Group: "batch", Resource: "Job"}, name.Name)
		})
	// deleteDbMigrationSecret
	mock.EXPECT().
		Delete(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *v1.Secret, opts ...client.DeleteOption) error {
			assert.Equal(t, dbMigrationSecret, secret.Name)
			return nil
		})
	// ResetVolumeReclaimPolicy
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *v1.PersistentVolumeList, opts ...client.ListOption) error {
			return errors.NewNotFound(schema.GroupResource{Group: "v1", Resource: "PersistentVolumeList"}, "")
		})

	ctx := spi.NewFakeContext(mock, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
	err := postUpgrade(ctx)
	assert.NoError(t, err)
}

// TestAppendMySQLOverridesUpgradeLegacyDevProfile tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides during upgrade for a dev profile cluster
// WHEN I pass in a VZ CR
// THEN the correct overrides are returned
func TestAppendMySQLOverridesUpgradeLegacyDevProfile(t *testing.T) {
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

	// isLegacyDatabaseUpgrade
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *v1.Secret) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			dep.Data = map[string][]byte{notDepFound: []byte("false")}
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *appsv1.Deployment) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *v1.Secret) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, secret *v1.Secret, opts ...client.UpdateOption) error {
			assert.Equal(t, dbMigrationSecret, secret.Name)
			return nil
		})

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: secretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = secretName
			secret.Data = map[string][]byte{}
			secret.Data["mysql-root-password"] = []byte("test-root-key")
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: secretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = secretName
			secret.Data = map[string][]byte{}
			secret.Data["mysql-password"] = []byte("test-user-key")
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *v1.Secret) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			dep.Data = map[string][]byte{notPVCDelete: []byte("false")}
			return nil
		})
	// get PVC
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: DeploymentPersistentVolumeClaim}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, pvc *v1.PersistentVolumeClaim) error {
			return errors.NewNotFound(schema.GroupResource{Group: "v1", Resource: "PersistentVolumeClaim"}, name.Name)
		})

	ctx := spi.NewFakeContext(mock, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 4+minExpectedHelmOverridesCount)
	assert.Equal(t, "test-root-key", bom.FindKV(kvs, helmRootPwd))
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
	// isLegacyDatabaseUpgrade
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *v1.Secret) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			dep.Data = map[string][]byte{notDepFound: []byte("false")}
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *appsv1.Deployment) error {
			return errors.NewNotFound(schema.GroupResource{Group: "appsv1", Resource: "Deployment"}, name.Name)
		})

	ctx := spi.NewFakeContext(mock, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 2)
}

// TestAppendMySQLOverridesUpgradeLegacyProdProfile tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides during upgrade for a prod profile cluster
// WHEN I pass in a VZ CR
// THEN the correct overrides are returned
func TestAppendMySQLOverridesUpgradeLegacyProdProfile(t *testing.T) {
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
	// isLegacyDatabaseUpgrade
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *v1.Secret) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			dep.Data = map[string][]byte{notDepFound: []byte("false")}
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *appsv1.Deployment) error {
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *v1.Secret) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, secret *v1.Secret, opts ...client.UpdateOption) error {
			assert.Equal(t, dbMigrationSecret, secret.Name)
			return nil
		})
	// appendLegacyUpgradeBaseValues
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: secretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Data = map[string][]byte{rootPasswordKey: []byte("test-root-key")}
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: secretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Data = map[string][]byte{secretKey: []byte("test-user-key")}
			return nil
		})
	// get PVC
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbMigrationSecret}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dep *v1.Secret) error {
			dep.Name = name.Name
			dep.Namespace = name.Namespace
			dep.Data = map[string][]byte{notPVCDelete: []byte("false")}
			return nil
		})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: DeploymentPersistentVolumeClaim}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, pvc *v1.PersistentVolumeClaim) error {
			return nil
		})
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *v1.PersistentVolumeList, opts ...client.ListOption) error {
			return nil
		}).AnyTimes()
	ctx := spi.NewFakeContext(mock, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 3+minExpectedHelmOverridesCount)
	assert.Equal(t, "test-root-key", bom.FindKV(kvs, helmRootPwd))
}

// TestClusterResourceDefaultRegistry tests the getRegistrySettings function
// GIVEN a call to getRegistrySettings
// WHEN there are no registry overrides
// THEN the default value is returned for the imageRepository helm value
func TestClusterResourceDefaultRegistry(t *testing.T) {
	const defaultRegistry = "ghcr.io/verrazzano"

	bomFile, err := bom.NewBom(testBomFilePath)
	assert.NoError(t, err)
	kvPair, err := getRegistrySettings(&bomFile)
	assert.NoError(t, err)

	assert.Equal(t, imageRepositoryKey, kvPair.Key)
	assert.Equal(t, defaultRegistry, kvPair.Value)
}

// TestClusterResourcePrivateRegistryOverride tests the getRegistrySettings function
// GIVEN a call to getRegistrySettings
// WHEN there are custom private registry overrides
// THEN the imageRepository helm value points to the correct registry/repo
func TestClusterResourcePrivateRegistryOverride(t *testing.T) {
	const registry = "myregistry.io"
	os.Setenv(vzconst.RegistryOverrideEnvVar, registry)
	defer func() { os.Unsetenv(vzconst.RegistryOverrideEnvVar) }()

	const repoPath = "someuser/basepath"
	os.Setenv(vzconst.ImageRepoOverrideEnvVar, repoPath)
	defer func() { os.Unsetenv(vzconst.ImageRepoOverrideEnvVar) }()

	bomFile, err := bom.NewBom(testBomFilePath)
	assert.NoError(t, err)
	kvPair, err := getRegistrySettings(&bomFile)
	assert.NoError(t, err)

	scRepo, err := bomFile.GetSubcomponent(bomSubComponentName)
	assert.NoError(t, err)

	assert.Equal(t, imageRepositoryKey, kvPair.Key)
	assert.Equal(t, fmt.Sprintf("%s/%s/%s", registry, repoPath, scRepo.Repository), kvPair.Value)
}

// TestIsMySQLReady tests the isMySQLReady function
// GIVEN a call to isMySQLReady
// WHEN the deployment object has enough replicas available
// AND the InnoDBCluster is online
// THEN true is returned
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
		newInnoDBCluster(innoDBClusterStatusOnline),
	).Build()
	servers := []byte(`{"serverInstances": 1}`)
	routers := []byte(`{"routerInstances": 1}`)

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
	mysql := NewComponent().(mysqlComponent)
	assert.True(t, mysql.isMySQLReady(spi.NewFakeContext(fakeClient, vz, nil, false)))
}

// TestIsMySQLNotReady tests the isMySQLReady function
// GIVEN a call to isMySQLReady
// WHEN the deployment object does NOT have enough replicas available
// THEN false is returned
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
	servers := []byte(`{"serverInstances": 1}`)
	routers := []byte(`{"routerInstances": 1}`)

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
	mysql := NewComponent().(mysqlComponent)
	assert.False(t, mysql.isMySQLReady(spi.NewFakeContext(fakeClient, vz, nil, false)))
}

// TestIsMySQLReadyInnoDBClusterNotOnline tests the isMySQLReady function
// GIVEN a call to isMySQLReady
// WHEN the deployment object has enough replicas available
// AND the InnoDBCluster is NOT online
// THEN false is returned
func TestIsMySQLReadyInnoDBClusterNotOnline(t *testing.T) {
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
		newInnoDBCluster("PARTIAL"),
	).Build()
	servers := []byte(`{"serverInstances": 1}`)
	routers := []byte(`{"routerInstances": 1}`)

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
	mysql := NewComponent().(mysqlComponent)
	assert.False(t, mysql.isMySQLReady(spi.NewFakeContext(fakeClient, vz, nil, false)))
}

// TestPostUpgradeCleanup tests the PostUpgradeCleanup function
// GIVEN a call to PostUpgradeCleanup
// WHEN The legacy DB upgrade PVC exists
// THEN It is deleted and no error is returned
func TestPostUpgradeCleanup(t *testing.T) {
	asserts := assert.New(t)
	legacyPVC := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: legacyDBDumpClaim, Namespace: ComponentNamespace},
	}
	otherPVC := &v1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"}}

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(legacyPVC, otherPVC).Build()

	asserts.NoError(PostUpgradeCleanup(vzlog.DefaultLogger(), fakeClient))

	pvc := &v1.PersistentVolumeClaim{}
	notFoundErr := fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(legacyPVC), pvc)
	asserts.Error(notFoundErr)
	asserts.True(errors.IsNotFound(notFoundErr))

	otherErr := fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(otherPVC), pvc)
	asserts.NoError(otherErr)
	asserts.Equal(client.ObjectKeyFromObject(otherPVC), client.ObjectKeyFromObject(pvc))
}

// TestPostUpgradeCleanupNoLegacyPVC tests the PostUpgradeCleanup function
// GIVEN a call to PostUpgradeCleanup
// WHEN The legacy DB upgrade PVC does NOT exist
// THEN no error is returned
func TestPostUpgradeCleanupNoLegacyPVC(t *testing.T) {
	asserts := assert.New(t)
	legacyPVC := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: legacyDBDumpClaim, Namespace: ComponentNamespace},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()

	pvc := &v1.PersistentVolumeClaim{}
	notFoundErr := fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(legacyPVC), pvc)
	asserts.Error(notFoundErr)
	asserts.True(errors.IsNotFound(notFoundErr))

	asserts.NoError(PostUpgradeCleanup(vzlog.DefaultLogger(), fakeClient))
}

// TestPostUpgradeCleanupErrorOnDelete tests the PostUpgradeCleanup function
// GIVEN a call to PostUpgradeCleanup
// WHEN an error other than IsNotFound is returned from the controllerruntime client
// THEN an error is returned
func TestPostUpgradeCleanupErrorOnDelete(t *testing.T) {
	asserts := assert.New(t)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	legacyPVC := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: legacyDBDumpClaim, Namespace: ComponentNamespace},
	}

	expectedErr := fmt.Errorf("An error")
	mock.EXPECT().Delete(gomock.Any(), legacyPVC).Return(expectedErr)

	err := PostUpgradeCleanup(vzlog.DefaultLogger(), mock)
	asserts.Error(err)
	asserts.Equal(expectedErr, err)
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
// WHEN The Keycloak component is nil
// THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, nil, false, profilesDir).EffectiveCR()))
}

// TestIsEnabledNilKeycloak tests the IsEnabled function
// GIVEN a call to IsEnabled
// WHEN The Keycloak component is nil
// THEN true is returned
func TestIsEnabledNilKeycloak(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesDir).EffectiveCR()))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
// WHEN The Keycloak component enabled is nil
// THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesDir).EffectiveCR()))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
// WHEN The Keycloak component is explicitly enabled
// THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesDir).EffectiveCR()))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
// WHEN The Keycloak component is explicitly disabled
// THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesDir).EffectiveCR()))
}

// TestIsEnabledManagedClusterProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
// WHEN The Keycloak enabled flag is nil and managed cluster profile
// THEN false is returned
func TestIsEnabledManagedClusterProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	cr.Spec.Profile = vzapi.ManagedCluster
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesDir).EffectiveCR()))
}

// TestIsEnabledProdProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
// WHEN The Keycloak enabled flag is nil and prod profile
// THEN false is returned
func TestIsEnabledProdProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	cr.Spec.Profile = vzapi.Prod
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesDir).EffectiveCR()))
}

// TestIsEnabledDevProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
// WHEN The Keycloak enabled flag is nil and dev profile
// THEN false is returned
func TestIsEnabledDevProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	cr.Spec.Profile = vzapi.Dev
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesDir).EffectiveCR()))
}

// TestConvertOldInstallArgs tests the convertOldInstallArgs function
// GIVEN a call to convertOldInstallArgs
// WHEN The old persistence values are passed in
// THEN the new values are returned
func TestConvertOldInstallArgs(t *testing.T) {
	persistenceVals := []bom.KeyValue{
		{
			Key:   "persistence.enabled",
			Value: "true",
		},
		{
			Key:   "persistence.storageClass",
			Value: "standard",
		},
		{
			Key:   "persistence.size",
			Value: "10Gi",
		},
		{
			Key:   "persistence.accessModes",
			Value: "ReadWriteOnce",
		},
	}
	newPersistenceVals := convertOldInstallArgs(persistenceVals)
	assert.Contains(t, newPersistenceVals, bom.KeyValue{Key: "persistence.enabled", Value: "true"})
	assert.Contains(t, newPersistenceVals, bom.KeyValue{Key: "datadirVolumeClaimTemplate.storageClassName", Value: "standard"})
	assert.Contains(t, newPersistenceVals, bom.KeyValue{Key: "datadirVolumeClaimTemplate.resources.requests.storage", Value: "10Gi"})
	assert.Contains(t, newPersistenceVals, bom.KeyValue{Key: "datadirVolumeClaimTemplate.accessModes", Value: "ReadWriteOnce"})
}

// TestCreateLegacyUpgradeJob tests the createLegacyUpgradeJob function
// GIVEN a call to createLegacyUpgradeJob
// WHEN I pass in a component context
// THEN an upgrade job is created
func TestCreateLegacyUpgradeJob(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	vz := &vzapi.Verrazzano{}
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ComponentNamespace,
		},
		Data: map[string][]byte{secretKey: []byte("test-user-key")},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(secret).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
	err := createLegacyUpgradeJob(ctx)
	assert.NoError(t, err)
	job := &batchv1.Job{}
	err = ctx.Client().Get(context.TODO(), types.NamespacedName{Name: dbLoadJobName, Namespace: ComponentNamespace}, job)
	assert.NoError(t, err)
	assert.NotNil(t, job)
	assert.True(t, len(job.Spec.Template.Spec.Containers) == 1)
	assert.True(t, len(job.Spec.Template.Spec.InitContainers) == 1)
}

// TestCheckDbMigrationJobCompletionForFailedJob tests the checkDbMigrationJobCompletion function
// GIVEN a call to checkDbMigrationJobCompletion for a failed job
// WHEN I pass in a component context
// THEN a new job instance is created
func TestCheckDbMigrationJobCompletionForFailedJob(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	vz := &vzapi.Verrazzano{}
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbLoadJobName,
			Namespace: ComponentNamespace,
		},
		Status: batchv1.JobStatus{
			Failed: 1,
		},
	}
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ComponentNamespace,
		},
		Data: map[string][]byte{secretKey: []byte("test-user-key")},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(job, secret).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
	result := checkDbMigrationJobCompletion(ctx)
	assert.False(t, result)
	newJob := &batchv1.Job{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Name: dbLoadJobName, Namespace: ComponentNamespace}, newJob)
	assert.NoError(t, err)
	assert.NotNil(t, newJob)
	assert.True(t, len(newJob.Spec.Template.Spec.Containers) == 1)
	assert.True(t, len(newJob.Spec.Template.Spec.InitContainers) == 1)
}

// TestCheckDbMigrationJobCompletionForSuccessfulJob tests the checkDbMigrationJobCompletion function
// GIVEN a call to checkDbMigrationJobCompletion for a successful job
// WHEN I pass in a component context
// THEN the check is successful (true)
func TestCheckDbMigrationJobCompletionForSuccessfulJob(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	vz := &vzapi.Verrazzano{}
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbLoadJobName,
			Namespace: ComponentNamespace,
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
		},
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"job-name": dbLoadJobName},
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name: dbLoadContainerName,
					State: v1.ContainerState{
						Terminated: &v1.ContainerStateTerminated{
							ExitCode: 0,
						},
					},
				},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(job, pod).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, nil, false, profilesDir).Init(ComponentName).Operation(vzconst.UpgradeOperation)
	result := checkDbMigrationJobCompletion(ctx)
	assert.True(t, result)
}

func getBoolPtr(b bool) *bool {
	return &b
}

// newInnoDBCluster returns an unstructured representation of an InnoDBCluster resource
func newInnoDBCluster(status string) *unstructured.Unstructured {
	innoDBCluster := unstructured.Unstructured{}
	innoDBCluster.SetGroupVersionKind(innoDBClusterGVK)
	innoDBCluster.SetNamespace(ComponentNamespace)
	innoDBCluster.SetName(helmReleaseName)
	unstructured.SetNestedField(innoDBCluster.Object, status, innoDBClusterStatusFields...)
	return &innoDBCluster
}
