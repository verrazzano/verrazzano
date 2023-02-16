// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

// TestGetPodListAll tests the functionality to return the list of all pods
func TestGetPodListAll(t *testing.T) {
	nsName := "test"
	podLength := 5
	var podList = []client.Object{}
	for i := 0; i < podLength; i++ {
		podList = append(podList, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsName + fmt.Sprint(i),
				Namespace: nsName,
				Labels:    map[string]string{"name": "myapp"},
			},
		})
	}
	//  GIVEN a k8s cluster with no pods,
	//	WHEN I call functions to get the list of pods in the k8s cluster,
	//	THEN expect it to be an empty list.
	pods, err := GetPodListAll(fake.NewClientBuilder().Build(), nsName)
	assert.NoError(t, err)
	assert.Empty(t, pods)

	//  GIVEN a k8s cluster with 5 pods,
	//	WHEN I call functions to get the list of pods in the k8s cluster without label,
	//	THEN expect it to be list all pods.
	pods, err = GetPodListAll(fake.NewClientBuilder().WithObjects(podList...).Build(), nsName)
	assert.NoError(t, err)
	assert.Equal(t, podLength, len(pods))
}
