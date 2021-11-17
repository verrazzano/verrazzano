// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package mysql

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"go.uber.org/zap"
	"os"
	"testing"
)

func TestCreateDBFile(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	fmt.Println(os.TempDir() + "/" + mysqlDBFile)
	err := createDBFile(spi.NewContext(zap.S(), nil, vz, false))
	assert.Nil(t, err, "error creating db file")
}

func TestPVs(t *testing.T) {
	/*	var volumeClaims map[string]*corev1.PersistentVolumeClaim
		volumeClaims, err := pkg.GetPersistentVolumes("keycloak")
		if err != nil {
			fmt.Print(err)
		}
		fmt.Printf("This many PVs: %v", len(volumeClaims))
		keys := reflect.ValueOf(volumeClaims).MapKeys()
		fmt.Println(keys)
		fmt.Println(volumeClaims["mysql"].Name)
		fmt.Println(volumeClaims["mysql"].Spec.Size())*/
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		fmt.Println(err)
	}

	vz, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	keycloak := vz.Spec.Components.Keycloak
	fmt.Println(keycloak)
}
