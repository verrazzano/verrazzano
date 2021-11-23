// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package mysql

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/helm"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
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

// Run genericHelmTestRunner executor
func (r genericHelmTestRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return r.stdOut, r.stdErr, r.err
}

// TestAppendMySQLOverrides tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
//  WHEN I pass a VZ spec with defaults
//  THEN the values created properly
func TestAppendMySQLOverrides(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	helm.SetCmdRunner(notDeployedRunner)
	defer helm.SetDefaultRunner()
	kvs, err := AppendMySQLOverrides(spi.NewFakeContext(nil, vz, false), "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 2)
	assert.Equal(t, bom.FindKV(kvs, mysqlUsernameKey), mysqlUsername)
	assert.Empty(t, bom.FindKV(kvs, "initializationFiles.create-db\\.sql"))
}

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
	assert.Equal(t, bom.FindKV(kvs, "key"), "value")
}

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
	kvs, err := AppendMySQLOverrides(spi.NewFakeContext(nil, vz, false), "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 3)
	assert.Equal(t, bom.FindKV(kvs, "persistence.enabled"), "false")
}

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
	kvs, err := AppendMySQLOverrides(spi.NewFakeContext(nil, vz, false), "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 4)
	assert.Equal(t, bom.FindKV(kvs, "persistence.enabled"), "true")
	assert.Equal(t, bom.FindKV(kvs, "persistence.size"), "100Gi")
}

func TestAppendMySQLOverridesProd(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.ProfileType("prod"),
		},
	}
	helm.SetCmdRunner(notDeployedRunner)
	defer helm.SetDefaultRunner()
	kvs, err := AppendMySQLOverrides(spi.NewFakeContext(nil, vz, false), "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 3)
	assert.Equal(t, bom.FindKV(kvs, "persistence.enabled"), "true")
	assert.Equal(t, bom.FindKV(kvs, "persistence.size"), "500Gi")
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
