// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"io/ioutil"
	"os"
	"testing"
)

// TestCreateReportArchive
// GIVEN a directory containing some files
//
//	WHEN I call function CreateReportArchive with a report file
//	THEN expect it to create the report file
func TestCreateReportArchive(t *testing.T) {
	tmpDir, _ := ioutil.TempDir("", "bug-report")
	defer os.RemoveAll(tmpDir)

	captureDir := tmpDir + string(os.PathSeparator) + "test-report"
	if err := os.Mkdir(captureDir, os.ModePerm); err != nil {
		assert.Error(t, err)
	}

	// Create some files inside bugReport
	_, err := os.Create(captureDir + string(os.PathSeparator) + "f1.txt")
	if err != nil {
		assert.Error(t, err)
	}

	_, err = os.Create(captureDir + string(os.PathSeparator) + "f2.txt")
	if err != nil {
		assert.Error(t, err)
	}

	_, err = os.Create(captureDir + string(os.PathSeparator) + "f3.txt")
	if err != nil {
		assert.Error(t, err)
	}

	bugReportFile, err := os.Create(tmpDir + string(os.PathSeparator) + "bug.tar.gz")
	if err != nil {
		assert.Error(t, err)
	}
	err = CreateReportArchive(captureDir, bugReportFile)
	if err != nil {
		assert.Error(t, err)
	}

	// Check file exists
	assert.FileExists(t, bugReportFile.Name())
}

// TestRemoveDuplicates
// GIVEN a string slice containing duplicates
//
//	WHEN I call function RemoveDuplicate
//	THEN expect it to remove the duplicate elements
func TestRemoveDuplicates(t *testing.T) {
	testSlice := []string{"abc", "def", "abc"}
	result := RemoveDuplicate(testSlice)
	assert.True(t, true, len(result) == 2)
}

// TestGroupVersionResource
//
//	WHEN I call functions to get the config schemes
//	THEN expect it to return the expected resource
func TestGroupVersionResource(t *testing.T) {
	assert.True(t, true, GetAppConfigScheme().Resource == constants.OAMAppConfigurations)
	assert.True(t, true, GetComponentConfigScheme().Resource == constants.OAMComponents)
	assert.True(t, true, GetMetricsTraitConfigScheme().Resource == constants.OAMMetricsTraits)
	assert.True(t, true, GetIngressTraitConfigScheme().Resource == constants.OAMIngressTraits)
	assert.True(t, true, GetMCComponentScheme().Resource == constants.OAMMCCompConfigurations)
	assert.True(t, true, GetMCAppConfigScheme().Resource == constants.OAMMCAppConfigurations)
	assert.True(t, true, GetVzProjectsConfigScheme().Resource == constants.OAMProjects)
	assert.True(t, true, GetManagedClusterConfigScheme().Resource == constants.OAMManagedClusters)
}
