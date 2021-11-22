// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package mysql

import (
	"github.com/stretchr/testify/assert"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestInstallAppendMySQLOverrides(t *testing.T) {

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
