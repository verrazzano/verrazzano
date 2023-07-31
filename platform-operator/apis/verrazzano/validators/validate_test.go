// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validators

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// For unit testing
const (
	actualBomFilePath          = "../../../verrazzano-bom.json"
	testBomFilePath            = "../testdata/test_bom.json"
	invalidTestBomFilePath     = "../testdata/invalid_test_bom.json"
	invalidPathTestBomFilePath = "../testdata/invalid_test_bom_path.json"

	v0170 = "v0.17.0"
	v110  = "v1.1.0"
)

// TestGetCurrentBomVersion Tests basic getBomVersion() happy path
// GIVEN a request for the current VZ Bom version
// WHEN the version in the Bom is available
// THEN no error is returned and a valid SemVersion representing the Bom version is returned
func TestGetCurrentBomVersion(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	expectedVersion, err := semver.NewSemVersion(v110)
	assert.NoError(t, err)

	version, err := GetCurrentBomVersion()
	assert.NoError(t, err)
	assert.Equal(t, expectedVersion, version)
}

// TestActualBomFile Tests GetCurrentBomVersion with the actual verrazzano-bom.json that is in this
// code repo to ensure the file can at least be parsed
func TestActualBomFile(t *testing.T) {
	// repeat the test with the _actual_ bom file in the code repository
	// to make sure it can at least be parsed without an error
	config.SetDefaultBomFilePath(actualBomFilePath)
	_, err := GetCurrentBomVersion()
	absPath, err2 := filepath.Abs(actualBomFilePath)
	if err2 != nil {
		absPath = actualBomFilePath
	}
	assert.NoError(t, err, "Could not get BOM version from file %s", absPath)
}

// TestGetCurrentBomVersionFileReadError Tests  getBomVersion() when there is an error reading the BOM file
// GIVEN a request for the current VZ Bom version
// WHEN an error occurs reading the BOM file from the filesystem
// THEN an error is returned and nil is returned for the Bom SemVersion
func TestGetCurrentBomVersionFileReadError(t *testing.T) {
	config.SetDefaultBomFilePath(invalidPathTestBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	version, err := GetCurrentBomVersion()
	assert.Error(t, err)
	assert.Nil(t, version)
}

// TestGetCurrentBomVersionBadYAML Tests  getBomVersion() when the BOM file is invalid
// GIVEN a request for the current VZ Bom version
// WHEN an error occurs reading in the BOM file as json
// THEN an error is returned and nil is returned for the Bom SemVersion
func TestGetCurrentBomVersionBadYAML(t *testing.T) {
	config.SetDefaultBomFilePath(invalidTestBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	version, err := GetCurrentBomVersion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
	assert.Nil(t, version)
}

// TestValidateVersionInvalidVersionCheckingDisabled Tests  ValidateVersion() when version checking is disabled
// GIVEN a request for the current VZ Bom version
// WHEN the version provided is not valid version and checking is disabled
// THEN no error is returned
func TestValidateVersionInvalidVersionCheckingDisabled(t *testing.T) {
	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})
	assert.NoError(t, ValidateVersion("blah"))
}

// TestValidateVersionInvalidVersion Tests  ValidateVersion() for invalid version
// GIVEN a request for the current VZ Bom version
// WHEN the version provided is not valid version
// THEN an error is returned
func TestValidateVersionInvalidVersion(t *testing.T) {
	assert.Error(t, ValidateVersion("blah"))
}

// TestValidateVersionBadBomFile Tests  ValidateVersion() the BOM file is bad
// GIVEN a request for the current VZ Bom version
// WHEN the version provided is not valid version
// THEN a json parsing error is returned
func TestValidateVersionBadBomfile(t *testing.T) {
	config.SetDefaultBomFilePath(invalidTestBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	err := ValidateVersion(v0170)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
}

// Test_validateSecretContents Tests validateSecretContents
// GIVEN a call to validateSecretContents
// WHEN the YAML bytes are not valid
// THEN an error is returned
func Test_validateSecretContents(t *testing.T) {
	err := ValidateSecretContents("mysecret", []byte("foo"), &AuthData{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error unmarshaling JSON")
}

// Test_validateSecretContentsEmpty Tests validateSecretContents
// GIVEN a call to validateSecretContents
// WHEN the YAML bytes are empty
// THEN an error is returned
func Test_validateSecretContentsEmpty(t *testing.T) {
	err := ValidateSecretContents("mysecret", []byte{}, &AuthData{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Secret \"mysecret\" data is empty")
}

// TestValidateVersionHigherOrEqualEmptyRequestedVersion Tests ValidateVersionHigherOrEqual() requestedVersion is empty
// GIVEN a request for the validating a requested version to be equal to or higher than provided current version
// WHEN the requestedVersion provided is emptty
// THEN failure is returned
func TestValidateVersionHigherOrEqualEmptyRequestedVersion(t *testing.T) {
	assert.False(t, ValidateVersionHigherOrEqual("v1.0.1", ""))
}

// TestValidateVersionHigherOrEqualEmptyVersion Tests  ValidateVersionHigherOrEqual() currentVersion is empty
// GIVEN a request for the validating a requested version to be equal to or higher than provided current version
// WHEN the currentVersion provided is emptty
// THEN failure is returned
func TestValidateVersionHigherOrEqualEmptyCurrentVersion(t *testing.T) {
	assert.False(t, ValidateVersionHigherOrEqual("", "v1.0.1"))
}

// TestValidateVersionHigherOrEqualEmptyVersion Tests  ValidateVersionHigherOrEqual() requestedVersion is invalid
// GIVEN a request for the validating a requested version to be equal to or higher than provided current version
// WHEN the requestedVersion provided is invalid
// THEN failure is returned
func TestValidateVersionHigherOrEqualInvalidRequestedVersion(t *testing.T) {
	assert.False(t, ValidateVersionHigherOrEqual("v1.0.1", "xyz.zz"))
}

// TestValidateVersionHigherOrEqualEmptyVersion Tests  ValidateVersionHigherOrEqual() currentVersion is invalid
// GIVEN a request for the validating a requested version to be equal to or higher than provided current version
// WHEN the currentVersion provided is invalid
// THEN failure is returned
func TestValidateVersionHigherOrEqualInvalidVersion(t *testing.T) {
	assert.False(t, ValidateVersionHigherOrEqual("xyz.zz", "v1.0.1"))
}

// TestValidateVersionHigherOrEqualEmptyVersion Tests ValidateVersionHigherOrEqual() versions are equal
// GIVEN a request for the validating a requested version to be equal to or higher than provided current version
// WHEN the requested version is equal to current version
// THEN success is returned
func TestValidateVersionHigherOrEqualCurrentVersion(t *testing.T) {
	assert.True(t, ValidateVersionHigherOrEqual("v1.0.1", "v1.0.1"))
}

// TestValidateVersionHigherOrEqualEmptyVersion Tests  ValidateVersionHigherOrEqual() requestedVersion is higher
// GIVEN a request for the validating a requested version to be equal to or higher than provided current version
// WHEN the requested version is greater than current ersion
// THEN failure is returned
func TestValidateVersionHigherOrEqualHigherVersion(t *testing.T) {
	assert.False(t, ValidateVersionHigherOrEqual("v1.0.1", "v1.0.2"))
}

// TestValidateVersionHigherOrEqualEmptyVersion Tests  ValidateVersionHigherOrEqual() requestedVersion is lower
// GIVEN a request for the validating a requested version to be equal to or higher than provided current version
// WHEN the requested version is lower than current version
// THEN success is returned
func TestValidateVersionHigherOrEqualLowerVersion(t *testing.T) {
	assert.True(t, ValidateVersionHigherOrEqual("v1.0.2", "v1.0.1"))
}

// TestValidateProfileInvalidProfile Tests cleanTempFiles()
// GIVEN a call to cleanTempFiles
// WHEN there are leftover validation temp files in the TMP dir
// THEN the temp files are cleaned up properly
func Test_cleanTempFiles(t *testing.T) {
	assert := assert.New(t)

	tmpFiles := []*os.File{}
	for i := 1; i < 5; i++ {
		temp, err := os.CreateTemp(os.TempDir(), validateTempFilePattern)
		assert.NoErrorf(err, "Unable to create temp file %s for testing: %s", temp.Name(), err)
		assert.FileExists(temp.Name())
		tmpFiles = append(tmpFiles, temp)
	}

	err := CleanTempFiles(zap.S())
	if assert.NoError(err) {
		for _, tmpFile := range tmpFiles {
			assert.NoFileExists(tmpFile.Name(), "Error, temp file %s not deleted", tmpFile.Name())
		}
	}
}

// TestVerifyPlatformOperatorSingletonNoFailedPods Tests the VerifyPlatformOperatorSingleton check
// GIVEN a VerifyPlatformOperatorSingleton call
// WHEN there are FAILED pods
// THEN an error is returned
func TestVerifyPlatformOperatorSingletonFailedPods(t *testing.T) {
	tests := []struct {
		name       string
		shouldPass bool
		failedPods int
		podList    []string
	}{
		{name: "Single VPO instance", shouldPass: true, podList: []string{"appName"}},
		{name: "Single VPO instance unhealthy", shouldPass: true, failedPods: 1, podList: []string{"appName"}},
		{name: "Multiple VPO instances all healthy", shouldPass: false, podList: []string{"foo", "goo"}},
		{name: "Multiple VPO instances unhealthy", shouldPass: false, failedPods: 1, podList: []string{"foo", "goo", "app"}},
		{name: "Multiple VPO instances 1 healthy", shouldPass: true, failedPods: 2, podList: []string{"foo", "goo", "app"}},
		{name: "Multiple VPO instances 1 unhealthy", shouldPass: true, failedPods: 1, podList: []string{"foo", "goo"}},
		{name: "Multiple VPO instances all unhealthy", shouldPass: true, failedPods: 4, podList: []string{"foo", "goo", "app", "app1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labels := map[string]string{
				"app": "verrazzano-platform-operator",
			}

			runtimeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithLists(&corev1.PodList{
				TypeMeta: metav1.TypeMeta{},
				Items:    createPodList(tt.podList, labels, tt.failedPods)},
			).Build()

			if tt.shouldPass {
				assert.NoError(t, VerifyPlatformOperatorSingleton(runtimeClient))
			} else if !tt.shouldPass {
				assert.Error(t, VerifyPlatformOperatorSingleton(runtimeClient))
			}
		})
	}
}

func createPodList(listOfPods []string, labels map[string]string, failedPods int) []corev1.Pod {
	var list []corev1.Pod
	for _, podName := range listOfPods {
		pod := corev1.Pod{}
		pod.Name = podName
		pod.Namespace = constants.VerrazzanoInstallNamespace
		pod.Labels = labels
		if failedPods != 0 {
			pod.Status.Phase = "Failed"
			failedPods--
		}
		list = append(list, pod)
	}
	return list
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	clientgoscheme.AddToScheme(scheme)
	return scheme
}
