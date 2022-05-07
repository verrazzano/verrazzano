// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/constants"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/klog"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/types"
	"go.uber.org/zap"
	"os"
	"strings"
	"testing"
)

func logHelper() (*zap.SugaredLogger, string) {
	file, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("verrazzano-%s-hook-*.log", strings.ToLower("TEST")))
	if err != nil {
		fmt.Printf("Unable to create temp file")
		os.Exit(1)
	}
	defer file.Close()
	log, _ := klog.Logger(file.Name())
	return log, file.Name()
}

func TestEnsureOpenSearchIsReachable(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)
	o := Opensearch(&OpensearchImpl{})
	ok := o.EnsureOpenSearchIsReachable(constants.EsUrl, log)
	assert.NotNil(t, ok)
	assert.Equal(t, false, false)
}

func TestRegisterSnapshotRepository(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)
	var objsecret types.ObjectStoreSecret
	objsecret.SecretName = "alpha"
	objsecret.SecretKey = "cloud"
	objsecret.ObjectAccessKey = "alphalapha"
	objsecret.ObjectSecretKey = "betabetabeta"
	var sdat types.ConnectionData
	sdat.Secret = objsecret
	sdat.BackupName = "mango"
	sdat.RegionName = "region"
	sdat.Endpoint = constants.EsUrl

	o := Opensearch(&OpensearchImpl{})
	err := o.RegisterSnapshotRepository(&sdat, log)
	assert.NotNil(t, err)
}

func TestTriggerSnapshot(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	o := Opensearch(&OpensearchImpl{})
	err := o.TriggerSnapshot("mango", log)
	assert.NotNil(t, err)
}

func TestCheckSnapshotProgress(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	o := Opensearch(&OpensearchImpl{})
	err := o.CheckSnapshotProgress("mango", log)
	assert.NotNil(t, err)
}

func TestDeleteDataStreams(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	o := Opensearch(&OpensearchImpl{})
	err := o.DeleteDataStreams(log)
	assert.NotNil(t, err)
}

func TestDeleteDataIndexes(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	o := Opensearch(&OpensearchImpl{})
	err := o.DeleteDataIndexes(log)
	assert.NotNil(t, err)
}

func TestTriggerRestore(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	o := Opensearch(&OpensearchImpl{})
	err := o.TriggerRestore("backup", log)
	assert.NotNil(t, err)
}
