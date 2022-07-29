// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"context"
	"os"
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
		Name:      secretName,
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
	ctx := spi.NewFakeContext(fakeClient, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, minExpectedHelmOverridesCount)
	assert.Equal(t, mySQLUsername, bom.FindKV(kvs, mySQLUsernameKey))
	assert.NotEmpty(t, bom.FindKV(kvs, "initdbScripts.create-db\\.sql"))
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
	secret.Data[mySQLKey] = []byte("test-key")
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&secret).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, false, profilesDir).Init(ComponentName).Operation(vzconst.InstallOperation)
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 4+minExpectedHelmOverridesCount)
	assert.Equal(t, "test-root-key", bom.FindKV(kvs, helmRootPwd))
	assert.Equal(t, "test-key", bom.FindKV(kvs, helmPwd))
	assert.Equal(t, mySQLUsername, bom.FindKV(kvs, mySQLUsernameKey))
	assert.NotEmpty(t, bom.FindKV(kvs, "initdbScripts.create-db\\.sql"))

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
	assert.NotEmpty(t, bom.FindKV(kvs, "initdbScripts.create-db\\.sql"))
	assert.NotEmpty(t, bom.FindKV(kvs, "auth.username"))
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
	assert.Equal(t, "false", bom.FindKV(kvs, "primary.persistence.enabled"))
	assert.NotEmpty(t, bom.FindKV(kvs, "initdbScripts.create-db\\.sql"))
	assert.NotEmpty(t, bom.FindKV(kvs, "auth.username"))
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
	assert.Len(t, kvs, 2+minExpectedHelmOverridesCount)
	assert.Equal(t, "true", bom.FindKV(kvs, "primary.persistence.enabled"))
	assert.Equal(t, "100Gi", bom.FindKV(kvs, "primary.persistence.size"))
	assert.NotEmpty(t, bom.FindKV(kvs, "initdbScripts.create-db\\.sql"))
	assert.NotEmpty(t, bom.FindKV(kvs, "auth.username"))
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
	assert.NotEmpty(t, bom.FindKV(kvs, "initdbScripts.create-db\\.sql"))
	assert.NotEmpty(t, bom.FindKV(kvs, "auth.username"))
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
	assert.Len(t, kvs, 2+minExpectedHelmOverridesCount)
	assert.Equal(t, "true", bom.FindKV(kvs, "primary.persistence.enabled"))
	assert.Equal(t, "100Gi", bom.FindKV(kvs, "primary.persistence.size"))
	assert.NotEmpty(t, bom.FindKV(kvs, "initdbScripts.create-db\\.sql"))
	assert.NotEmpty(t, bom.FindKV(kvs, "auth.username"))
	assert.NotEmpty(t, bom.FindKV(kvs, "image"))
}

// TestAppendMySQLOverridesUpgrade tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides during upgrade
// WHEN I pass in an empty VZ CR
// THEN the correct overrides are returned
func TestAppendMySQLOverridesUpgrade(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	vz := &vzapi.Verrazzano{}
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ComponentNamespace, Name: secretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = secretName
			secret.Data = map[string][]byte{}
			secret.Data[mySQLRootKey] = []byte("test-root-key")
			secret.Data[mySQLKey] = []byte("test-key")
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
	assert.Len(t, kvs, 3+minExpectedHelmOverridesCount)
	assert.Equal(t, "test-root-key", bom.FindKV(kvs, helmRootPwd))
	assert.Equal(t, "test-key", bom.FindKV(kvs, helmPwd))
	assert.NotEmpty(t, bom.FindKV(kvs, "auth.username"))
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
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName + "-0",
				Labels: map[string]string{
					"controller-revision-hash": ComponentName + "-f97fd59d8",
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
	assert.True(t, isMySQLReady(spi.NewFakeContext(fakeClient, nil, false)))
}

// TestIsMySQLNotReady tests the isMySQLReady function
// GIVEN a call to isMySQLReady
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
func TestIsMySQLNotReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentName,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   0,
		},
	}).Build()
	assert.False(t, isMySQLReady(spi.NewFakeContext(fakeClient, nil, false)))
}

// TestSQLFileCreatedAndRemoved tests the creation and deletion of the mysql db init file
// WHEN the appendMySQLOverrides and then postInstall functions are called
// THEN ensure that the mysql db init file is created successfully and then deleted successfully
func TestSQLFileCreatedAndRemoved(t *testing.T) {
	fakeContext := spi.NewFakeContext(nil, nil, false)
	tmpFile, err := createMySQLInitFile(fakeContext)
	assert.NoError(t, err)
	tmpFileContents, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, tmpFileContents)
	removeMySQLInitFile(fakeContext)
	assert.NoFileExists(t, tmpFile)
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
