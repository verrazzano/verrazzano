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
	"github.com/verrazzano/verrazzano/platform-operator/internal/helm"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

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
	err:    fmt.Errorf("Release was not found in output"),
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

// TestAppendMySQLOverrides tests the AppendMySQLOverrides function
// GIVEN a call to AppendMySQLOverrides
// WHEN I pass in an empty VZ CR
// THEN the correct overrides are returned
func TestAppendMySQLOverrides(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	helm.SetCmdRunner(notDeployedRunner)
	defer helm.SetDefaultRunner()
	kvs, err := AppendMySQLOverrides(spi.NewFakeContext(nil, vz, false), "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 2)
	assert.Equal(t, mysqlUsername, bom.FindKV(kvs, mysqlUsernameKey))
	assert.NotEmpty(t, bom.FindKV(kvs, "initializationFiles.create-db\\.sql"))
}

// TestAppendMySQLOverridesWithInstallArgs tests the AppendMySQLOverrides function
// GIVEN a call to AppendMySQLOverrides
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
	helm.SetCmdRunner(notDeployedRunner)
	defer helm.SetDefaultRunner()
	kvs, err := AppendMySQLOverrides(spi.NewFakeContext(nil, vz, false), "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 3)
	assert.Equal(t, "value", bom.FindKV(kvs, "key"))
}

// TestAppendMySQLOverridesDev tests the AppendMySQLOverrides function
// GIVEN a call to AppendMySQLOverrides
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
	helm.SetCmdRunner(notDeployedRunner)
	defer helm.SetDefaultRunner()
	kvs, err := AppendMySQLOverrides(spi.NewFakeContext(nil, vz, false, "../../../../manifests/profiles"), "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 3)
	assert.Equal(t, "false", bom.FindKV(kvs, "persistence.enabled"))
}

// TestAppendMySQLOverridesDevWithPersistence tests the AppendMySQLOverrides function
// GIVEN a call to AppendMySQLOverrides
// WHEN I pass in an VZ CR with the dev profile
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
	helm.SetCmdRunner(notDeployedRunner)
	defer helm.SetDefaultRunner()
	kvs, err := AppendMySQLOverrides(spi.NewFakeContext(nil, vz, false, "../../../../manifests/profiles"), "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 4)
	assert.Equal(t, "true", bom.FindKV(kvs, "persistence.enabled"))
	assert.Equal(t, "100Gi", bom.FindKV(kvs, "persistence.size"))
}

// TestAppendMySQLOverridesProd tests the AppendMySQLOverrides function
// GIVEN a call to AppendMySQLOverrides
// WHEN I pass in an VZ CR with the dev profile
// THEN the overrides contain the correct mysql persistence config
func TestAppendMySQLOverridesProd(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("prod"),
		},
	}
	helm.SetCmdRunner(notDeployedRunner)
	defer helm.SetDefaultRunner()
	kvs, err := AppendMySQLOverrides(spi.NewFakeContext(nil, vz, false, "../../../../manifests/profiles"), "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 4)
	assert.Equal(t, "true", bom.FindKV(kvs, "persistence.enabled"))
	assert.Equal(t, "50Gi", bom.FindKV(kvs, "persistence.size"))
}

// TestAppendMySQLOverridesUpgrade tests the AppendMySQLOverrides function
// GIVEN a call to AppendMySQLOverrides during upgrade
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
			secret.Data[mysqlRootKey] = []byte("test-root-key")
			secret.Data[mysqlKey] = []byte("test-key")
			return nil
		})
	helm.SetCmdRunner(deployedRunner)
	defer helm.SetDefaultRunner()
	kvs, err := AppendMySQLOverrides(spi.NewFakeContext(mock, vz, false), "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 3)
	assert.Equal(t, "test-root-key", bom.FindKV(kvs, helmRootPwd))
	assert.Equal(t, "test-key", bom.FindKV(kvs, helmPwd))
}

// TestIsMySQLReady tests the IsReady function
// GIVEN a call to IsReady
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
	assert.True(t, IsReady(spi.NewFakeContext(fakeClient, nil, false), ComponentName, vzconst.KeycloakNamespace))
}

// TestIsMySQLNotReady tests the IsReady function
// GIVEN a call to IsReady
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
	assert.False(t, IsReady(spi.NewFakeContext(fakeClient, nil, false), "", vzconst.KeycloakNamespace))
}

func TestSQLFileCreatedAndDeleted(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	helm.SetCmdRunner(notDeployedRunner)
	defer helm.SetDefaultRunner()
	fakeContext := spi.NewFakeContext(nil, vz, false, "../../../../manifests/profiles")
	_, err := AppendMySQLOverrides(fakeContext, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	sqlFileContents, err := os.ReadFile(os.TempDir() + "/" + mysqlDBFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, sqlFileContents)
	err = PostInstall(fakeContext)
	assert.NoError(t, err)
	assert.NoFileExists(t, os.TempDir()+"/"+mysqlDBFile)
}

/*func TestCreateDBFile(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	fmt.Println(os.TempDir() + "/" + mysqlDBFile)
	err := createDBFile(spi.NewFakeContext(nil, vz, false, "../../../../manifests/profiles"))
	assert.Nil(t, err, "error creating db file")
}

func TestAppendOverrides(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	var devProfile vzapi.ProfileType = "dev"
	vz.Spec.Profile = devProfile
	ctx := spi.NewFakeContext(nil, vz, false, "../../../../manifests/profiles")
	var kvs []bom.KeyValue
	kvs, err := AppendMySQLOverrides(ctx, "", "", "", kvs)
	fmt.Println(kvs)
	assert.Nil(t, err, "Should be nil", err.Error())
}*/
