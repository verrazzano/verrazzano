// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validators

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"go.uber.org/zap"
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

func TestCheckUpgradeRequired(t *testing.T) {
	//can insert a case for when semver returns an error
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	bomVersion, err := GetCurrentBomVersion()
	assert.NoError(t, err)

	currVersion := "An Invalid Version"
	err = CheckUpgradeRequired(currVersion, bomVersion)
	assert.Error(t, err)

	currVersion = "1.1.5"
	err = CheckUpgradeRequired(currVersion, bomVersion)
	assert.NoError(t, err)

	currVersion = "1.0.5"
	err = CheckUpgradeRequired(currVersion, bomVersion)
	assert.Error(t, err)

	currVersion = ""
	err = CheckUpgradeRequired(currVersion, bomVersion)
	assert.NoError(t, err)

}

func TestValidateNewVersionforInvalidVersions(t *testing.T) {
	//can add the case to test each of them separately, when they are invalid
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	bomVersion, err := GetCurrentBomVersion()
	assert.NoError(t, err)

	currStatusVerString := "dummystr1"
	currSpecVerString := "dummmystr2"
	newVerString := "dummystr3"

	err = ValidateNewVersion(currStatusVerString, currSpecVerString, newVerString, bomVersion)
	assert.Error(t, err)

	currStatusVerString = "dummystr1"
	currSpecVerString = "dummmystr2"
	newVerString = "1.1.0"

	err = ValidateNewVersion(currStatusVerString, currSpecVerString, newVerString, bomVersion)
	assert.Error(t, err)

	currStatusVerString = "1.0.0"
	currSpecVerString = "dummmystr2"
	newVerString = "1.1.0"

	err = ValidateNewVersion(currStatusVerString, currSpecVerString, newVerString, bomVersion)
	assert.Error(t, err)

}

func TestValidateNewVersion(t *testing.T) {
	//can add the case to test each of them separately, when they are invalid
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	bomVersion, err := GetCurrentBomVersion()
	assert.NoError(t, err)

	currStatusVerString := "1.1.1"
	currSpecVerString := "1.1.0"
	newVerString := "1.1.2"

	err = ValidateNewVersion(currStatusVerString, currSpecVerString, newVerString, bomVersion)
	assert.Error(t, err)

	currStatusVerString = "1.1.2"
	currSpecVerString = "1.1.0"
	newVerString = "1.1.0"

	err = ValidateNewVersion(currStatusVerString, currSpecVerString, newVerString, bomVersion)
	assert.Error(t, err)

	currStatusVerString = "0.9.5"
	currSpecVerString = "1.2.0"
	newVerString = "1.1.0"

	err = ValidateNewVersion(currStatusVerString, currSpecVerString, newVerString, bomVersion)
	assert.Error(t, err)

	currStatusVerString = "1.0.0"
	currSpecVerString = "1.0.0"
	newVerString = "1.1.0"

	err = ValidateNewVersion(currStatusVerString, currSpecVerString, newVerString, bomVersion)
	assert.NoError(t, err)

}

/*
func TestValidatePrivateKey(t *testing.T) {
    secretName :="mysecret"
    var pemData = []byte(`
    -----BEGIN PUBLIC KEY-----
    MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAlRuRnThUjU8/prwYxbty
    WPT9pURI3lbsKMiB6Fn/VHOKE13p4D8xgOCADpdRagdT6n4etr9atzDKUSvpMtR3
    CP5noNc97WiNCggBjVWhs7szEe8ugyqF23XwpHQ6uV1LKH50m92MbOWfCtjU9p/x
    qhNpQQ1AZhqNy5Gevap5k8XzRmjSldNAFZMY7Yv3Gi+nyCwGwpVtBUwhuLzgNFK/
    yDtw2WcWmUU7NuC8Q6MWvPebxVtCfVp/iQU6q60yyt6aGOBkhAX0LpKAEhKidixY
    nP9PNVBvxgu3XZ4P36gZV6+ummKdBVnc3NqwBLu5+CcdRdusmHPHd5pHf4/38Z3/
    6qU2a/fPvWzceVTEgZ47QjFMTCTmCwNt29cvi7zZeQzjtwQgn4ipN9NibRH/Ax/q
    TbIzHfrJ1xa2RteWSdFjwtxi9C20HUkjXSeI4YlzQMH0fPX6KCE7aVePTOnB69I/
    a9/q96DiXZajwlpq3wFctrs1oXqBp5DVrCIj8hU2wNgB7LtQ1mCtsYz//heai0K9
    PhE4X6hiE0YmeAZjR0uHl8M/5aW9xCoJ72+12kKpWAa0SFRWLy6FejNYCYpkupVJ
    yecLk/4L1W0l6jQQZnWErXZYe0PNFcmwGXy1Rep83kfBRNKRy5tvocalLlwXLdUk
    AIU+2GKjyT3iMuzZxxFxPFMCAwEAAQ==
    -----END PUBLIC KEY-----
    and some more`)

    err :=ValidatePrivateKey(secretName,pemData)
    assert.NoError(t, err)
}*/
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

// TestIsKubernetesVersionSupported tests IsKubernetesVersionSupported()
// GIVEN a request for the validating that the Kubernetes version of cluster is supported by the operator
// WHEN the Kubernetes version and Supported versions can be determined without error
// AND the Kubernetes version is either equal to one of the supported versions or is a patch version of a supported version
// THEN only true is returned
func TestIsKubernetesVersionSupported(t *testing.T) {
	tests := []struct {
		name                               string
		getSupportedKubernetesVersionsFunc func() ([]string, error)
		getKubernetesVersionFunc           func() (string, error)
		result                             bool
	}{
		{
			name:                               "testFailGettingSupportedVersions",
			getSupportedKubernetesVersionsFunc: func() ([]string, error) { return nil, fmt.Errorf("errored out") },
			getKubernetesVersionFunc:           func() (string, error) { return "v0.1.5", nil },
			result:                             false,
		},
		{
			name:                               "testFailGettingKubernetesVersion",
			getSupportedKubernetesVersionsFunc: func() ([]string, error) { return []string{"v0.1.0", "v0.2.0"}, nil },
			getKubernetesVersionFunc:           func() (string, error) { return "", fmt.Errorf("errored out") },
			result:                             false,
		},
		{
			name:                               "testPassNoSupportedVersionsInBom",
			getSupportedKubernetesVersionsFunc: func() ([]string, error) { return nil, nil },
			getKubernetesVersionFunc:           func() (string, error) { return "", fmt.Errorf("errored out") },
			result:                             true,
		},
		{
			name:                               "testFailInvalidSupportedVersionsInBom",
			getSupportedKubernetesVersionsFunc: func() ([]string, error) { return []string{"v1.2.0", "vx.y"}, nil },
			getKubernetesVersionFunc:           func() (string, error) { return "v1.3.9", nil },
			result:                             false,
		},
		{
			name:                               "testFailInvalidKubernetesVersions",
			getSupportedKubernetesVersionsFunc: func() ([]string, error) { return []string{"v1.2.0", "v1.3.0"}, nil },
			getKubernetesVersionFunc:           func() (string, error) { return "vx.y", nil },
			result:                             false,
		},
		{
			name:                               "testPassExactSupportedKubernetesVersion",
			getSupportedKubernetesVersionsFunc: func() ([]string, error) { return []string{"v1.2.5", "v1.3.0"}, nil },
			getKubernetesVersionFunc:           func() (string, error) { return "v1.2.5", nil },
			result:                             true,
		},
		{
			name:                               "testPassPatchSupportedKubernetesVersion",
			getSupportedKubernetesVersionsFunc: func() ([]string, error) { return []string{"v1.2.5", "v1.3.0"}, nil },
			getKubernetesVersionFunc:           func() (string, error) { return "v1.3.8", nil },
			result:                             true,
		},
		{
			name:                               "testPassNotSupportedKubernetesVersion",
			getSupportedKubernetesVersionsFunc: func() ([]string, error) { return []string{"v1.2.5", "v1.3.0"}, nil },
			getKubernetesVersionFunc:           func() (string, error) { return "v1.4.8", nil },
			result:                             false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getSupportedKubernetesVersionsOriginal := getSupportedKubernetesVersions
			getKubernetesVersionOriginal := getKubernetesVersion
			getKubernetesClusterVersion = tt.getKubernetesVersionFunc
			getSupportedVersions = tt.getSupportedKubernetesVersionsFunc
			defer func() {
				getSupportedVersions = getSupportedKubernetesVersionsOriginal
				getKubernetesClusterVersion = getKubernetesVersionOriginal

			}()
			assert.Equal(t, tt.result, IsKubernetesVersionSupported())
		})
	}
}
