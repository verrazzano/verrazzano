// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package utils_test

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/constants"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/utils"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

// TestPopulateConnData tests the PopulateConnData method for the following use case.
// GIVEN a velero backup name
// WHEN velero backup is in progress
// THEN fetches the secret associate with velero backup
func TestPopulateConnData(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)
	var clientk client.Client
	k8s := utils.K8s(&utils.K8sImpl{})
	dclient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	conData, err := k8s.PopulateConnData(dclient, clientk, constants.VeleroNameSpace, "Foo", log)
	spew.Dump(conData)
	assert.NotNil(t, err)
}

// TestGetBackupStorageLocation tests the GetBackupStorageLocation method for the following use case.
// GIVEN a velero backup storage location name
// WHEN invoked
// THEN fetches backup storage location object
func TestGetBackupStorageLocation(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)
	k8s := utils.K8s(&utils.K8sImpl{})
	dclient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	_, err := k8s.GetBackupStorageLocation(dclient, "system", "fsl", log)
	assert.NotNil(t, err)
}

// TestGetBackup tests the GetBackup method for the following use case.
// GIVEN a velero backup name
// WHEN invoked
// THEN fetches backup object
func TestGetBackup(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)
	k8s := utils.K8s(&utils.K8sImpl{})
	dclient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	_, err := k8s.GetBackup(dclient, "system", "foo", log)
	assert.NotNil(t, err)
}
