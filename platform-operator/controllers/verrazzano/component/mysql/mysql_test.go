// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package mysql

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const profilesDir = "../../../../manifests/profiles"

var pvc100Gi, _ = resource.ParseQuantity("100Gi")

// genericHelmTestRunner is used to run generic OS commands with expected results
type genericHelmTestRunner struct {
	stdOut []byte
	stdErr []byte
	err    error
}

// notDeployedRunner returns the expected helm status output when mysql is not deployed
var notDeployedRunner = genericHelmTestRunner{
	stdOut: []byte{},
	stdErr: []byte("Error: release: not found"),
	err:    fmt.Errorf("release was not found in output"),
}

// deployedRunner returns the expected helm status output when mysql is deployed
var deployedRunner = genericHelmTestRunner{
	stdOut: []byte("{\"info\":{\"status\":\"deployed\"}}"),
	stdErr: []byte{},
	err:    nil,
}

// Run genericHelmTestRunner executor
func (r genericHelmTestRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return r.stdOut, r.stdErr, r.err
}

// TestAppendMySQLOverrides tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides
// WHEN I pass in an empty VZ CR
// THEN the correct overrides are returned
func TestAppendMySQLOverrides(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	ctx := spi.NewFakeContext(nil, vz, false, profilesDir).For(ComponentName).Operation("install")
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 2)
	assert.Equal(t, mySQLUsername, bom.FindKV(kvs, mySQLUsernameKey))
	assert.NotEmpty(t, bom.FindKV(kvs, "initializationFiles.create-db\\.sql"))
}

// TestAppendMySQLOverridesWithInstallArgs tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides
// WHEN I pass in an empty VZ CR with MySQL install args
// THEN the override key value pairs contain the install args
func TestAppendMySQLOverridesWithInstallArgs(t *testing.T) {
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
	ctx := spi.NewFakeContext(nil, vz, false, profilesDir).For(ComponentName).Operation("install")
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 3)
	assert.Equal(t, "value", bom.FindKV(kvs, "key"))
}

// TestAppendMySQLOverridesDev tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides
// WHEN I pass in an VZ CR with the dev profile
// THEN the overrides contain the correct mysql persistence config
func TestAppendMySQLOverridesDev(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("dev"),
			DefaultVolumeSource: &v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		},
	}
	ctx := spi.NewFakeContext(nil, vz, false, profilesDir).For(ComponentName).Operation("install")
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 3)
	assert.Equal(t, "false", bom.FindKV(kvs, "persistence.enabled"))
}

// TestAppendMySQLOverridesDevWithPersistence tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides
// WHEN I pass in an VZ CR with the dev profile and persistence overrides
// THEN the overrides contain the correct mysql persistence config
func TestAppendMySQLOverridesDevWithPersistence(t *testing.T) {
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
	ctx := spi.NewFakeContext(nil, vz, false, profilesDir).For(ComponentName).Operation("install")
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 4)
	assert.Equal(t, "true", bom.FindKV(kvs, "persistence.enabled"))
	assert.Equal(t, "100Gi", bom.FindKV(kvs, "persistence.size"))
}

// TestAppendMySQLOverridesProd tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides
// WHEN I pass in an VZ CR with the prod profile
// THEN the overrides contain the correct mysql persistence config
func TestAppendMySQLOverridesProd(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("prod"),
		},
	}
	ctx := spi.NewFakeContext(nil, vz, false, profilesDir).For(ComponentName).Operation("install")
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 2)
}

// TestAppendMySQLOverridesProdWithOverrides tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides
// WHEN I pass in an VZ CR with the pred profile and a default volume source override
// THEN the overrides contain the correct mysql persistence config
func TestAppendMySQLOverridesProdWithOverrides(t *testing.T) {
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
	ctx := spi.NewFakeContext(nil, vz, false, profilesDir).For(ComponentName).Operation("install")
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 4)
	assert.Equal(t, "true", bom.FindKV(kvs, "persistence.enabled"))
	assert.Equal(t, "100Gi", bom.FindKV(kvs, "persistence.size"))
}

// TestAppendMySQLOverridesUpgrade tests the appendMySQLOverrides function
// GIVEN a call to appendMySQLOverrides during upgrade
// WHEN I pass in an empty VZ CR
// THEN the correct overrides are returned
func TestAppendMySQLOverridesUpgrade(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: vzconst.KeycloakNamespace, Name: secretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = secretName
			secret.Data = map[string][]byte{}
			secret.Data[mySQLRootKey] = []byte("test-root-key")
			secret.Data[mySQLKey] = []byte("test-key")
			return nil
		})
	ctx := spi.NewFakeContext(mock, vz, false, profilesDir).For(ComponentName).Operation("upgrade")
	kvs, err := appendMySQLOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 3)
	assert.Equal(t, "test-root-key", bom.FindKV(kvs, helmRootPwd))
	assert.Equal(t, "test-key", bom.FindKV(kvs, helmPwd))
}

// TestIsMySQLReady tests the isReady function
// GIVEN a call to isReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsMySQLReady(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconst.KeycloakNamespace,
			Name:      ComponentName,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	})
	assert.True(t, isReady(spi.NewFakeContext(fakeClient, nil, false), ComponentName, vzconst.KeycloakNamespace))
}

// TestIsMySQLNotReady tests the isReady function
// GIVEN a call to isReady
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
func TestIsMySQLNotReady(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconst.KeycloakNamespace,
			Name:      ComponentName,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       0,
			AvailableReplicas:   0,
			UnavailableReplicas: 1,
		},
	})
	assert.False(t, isReady(spi.NewFakeContext(fakeClient, nil, false), "", vzconst.KeycloakNamespace))
}

// TestSQLFileCreatedAndDRemoved tests the creation and deletion of the mysql db init file
// WHEN the appendMySQLOverrides and then postInstall functions are called
// THEN ensure that the mysql db init file is created successfully and then deleted successfully
func TestSQLFileCreatedAndDRemoved(t *testing.T) {
	fakeContext := spi.NewFakeContext(nil, nil, false)
	tmpFile, err := createMySQLInitFile(fakeContext)
	assert.NoError(t, err)
	tmpFileContents, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, tmpFileContents)
	removeMySQLInitFile(fakeContext)
	assert.NoFileExists(t, tmpFile)
}
