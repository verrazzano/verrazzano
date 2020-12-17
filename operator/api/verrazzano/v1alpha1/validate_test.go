// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/operator/internal/util/env"
	"github.com/verrazzano/verrazzano/operator/internal/util/semver"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"os"
	"testing"
)

const validChartYAML = `
apiVersion: v1
description: A Helm chart for Verrazzano
name: verrazzano
version: 0.7.0
appVersion: 0.7.0
`

// TestValidUpgradeRequestNoCurrentVersion Tests the SemVersion parser for valid version strings
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the current version is not specified
// THEN ensure no error is returned from ValidateUpgradeRequest
func TestValidUpgradeRequestNoCurrentVersion(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.7.0",
		Profile: "dev",
	}
	assert.NoError(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestValidUpgradeRequestCurrentVersionExists Tests the SemVersion parser for valid version strings
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the current version is less than the current version
// THEN ensure no error is returned from ValidateUpgradeRequest
func TestValidUpgradeRequestCurrentVersionExists(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Version: "v0.6.0",
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.7.0",
		Profile: "dev",
	}
	assert.NoError(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestValidUpgradeRequestCurrentVersionExists Tests the SemVersion parser for valid version strings
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version and the current version are at the latest version
// THEN ensure no error is returned from ValidateUpgradeRequest
func TestValidUpgradeNotNecessary(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Version: "v0.7.0",
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.7.0",
		Profile: "dev",
	}
	assert.NoError(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

func TestValidateUpgradeBadOldVersion(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Version: "blah",
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.7.0",
		Profile: "dev",
	}
	assert.Error(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

func TestValidateUpgradeBadNewVersion(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Version: "v0.6.0",
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version: "blah",
		Profile: "dev",
	}
	assert.Error(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestNoVersionsSpecified Tests the validate passes if no versions are specified in either spec
// GIVEN an edit to update a Verrazzano spec
// WHEN the new version and the current version are not specified
// THEN ensure no error is returned from ValidateUpgradeRequest
func TestNoVersionsSpecified(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Profile: "dev",
	}
	assert.NoError(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestValidValidVersionWithProfileChange Tests the validate fails if the upgrade version is OK but the profile is changed
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the profile field is changed
// THEN an error is returned from ValidateUpgradeRequest
func TestValidVersionWithProfileChange(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.7.0",
		Profile: "prod",
	}
	assert.Error(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestProfileChangeOnlyNoVersions Tests the validate fails if no versions specified but the profile is changed
// GIVEN an edit to update a Verrazzano spec to change a profile value
// WHEN the no version strings are specified
// THEN no error is returned from ValidateUpgradeRequest
func TestProfileChangeOnlyNoVersions(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Profile: "prod",
	}
	assert.NoError(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestProfileChangeOnlyNoNewVersionString Tests the validate fails if the old spec has a version but the new one doesn't
// GIVEN an edit to update a Verrazzano spec to change a profile value
// WHEN the old spec specifies a version but the new one does not
// THEN an error is returned from ValidateUpgradeRequest
func TestProfileChangeOnlyNoNewVersionString(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
		Version: "v0.6.0",
	}
	newSpec := &VerrazzanoSpec{
		Profile: "prod",
	}
	err := ValidateUpgradeRequest(currentSpec, newSpec)
	assert.Error(t, err)
	assert.Equal(t, "Requested version is not specified", err.Error())
}

// TestValidVersionWithEnvNameChange Tests the validate fails if the upgrade version is OK but the EnvironmentName is changed
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the EnvironmentName field is changed
// THEN an error is returned from ValidateUpgradeRequest
func TestValidVersionWithEnvNameChange(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version:         "v0.7.0",
		Profile:         "dev",
		EnvironmentName: "newEnv",
	}
	assert.Error(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestValidVersionWithEnvNameChange Tests the validate fails if the upgrade version is OK but the EnvironmentName is changed
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the EnvironmentName field is changed
// THEN an error is returned from ValidateUpgradeRequest
func TestValidVersionWithCertManagerChange(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
		Components: ComponentSpec{
			CertManager: CertManagerComponent{
				Certificate: Certificate{
					Acme: Acme{
						Provider:     "MyProvider",
						EmailAddress: "email1@mycompany.com",
						Environment:  "someEnv",
					},
				},
			},
		},
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.7.0",
		Profile: "dev",
		Components: ComponentSpec{
			CertManager: CertManagerComponent{
				Certificate: Certificate{
					Acme: Acme{
						Provider:     "MyProvider",
						EmailAddress: "email2@mycompany.com",
						Environment:  "someEnv",
					},
				},
			},
		},
	}
	assert.Error(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestValidVersionWithNewDNS Tests the validate fails if the upgrade version is OK but the DNS component is added
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the DNS component is added
// THEN an error is returned from ValidateUpgradeRequest
func TestValidVersionWithNewDNS(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
		Components: ComponentSpec{
			CertManager: CertManagerComponent{
				Certificate: Certificate{
					Acme: Acme{
						Provider:     "MyProvider",
						EmailAddress: "email1@mycompany.com",
						Environment:  "someEnv",
					},
				},
			},
		},
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.7.0",
		Profile: "dev",
		Components: ComponentSpec{
			CertManager: CertManagerComponent{
				Certificate: Certificate{
					Acme: Acme{
						Provider:     "MyProvider",
						EmailAddress: "email1@mycompany.com",
						Environment:  "someEnv",
					},
				},
			},
			DNS: DNSComponent{
				OCI: OCI{
					OCIConfigSecret:        "secret",
					DNSZoneCompartmentOCID: "zonecompocid",
					DNSZoneOCID:            "zoneOcid",
					DNSZoneName:            "zoneName",
				},
			},
		},
	}
	assert.Error(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestValidVersionWithIngressChange Tests the validate fails if the upgrade version is OK but the Ingress component is changed
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the Ingress component is changed
// THEN an error is returned from ValidateUpgradeRequest
func TestValidVersionWithIngressChange(t *testing.T) {
	assert.Error(t, runValidateWithIngressChangeTest())
}

// TestValidVersionWithIngressChangeVersionCheckDisabled Tests the validate passes for component change with version check disabled
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the Ingress component is changed, but version checking is disabled
// THEN no error is returned from ValidateUpgradeRequest
func TestValidVersionWithIngressChangeVersionCheckDisabled(t *testing.T) {
	os.Setenv(env.CheckVersionDisabled, "true")
	defer os.Unsetenv(env.CheckVersionDisabled)
	assert.NoError(t, runValidateWithIngressChangeTest())
}

// runValidateWithIngressChangeTest Shared test logic for ingress change validation
func runValidateWithIngressChangeTest() error {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
		Components: ComponentSpec{
			Ingress: IngressNginxComponent{
				Type: "sometype",
				NGINXInstallArgs: []InstallArgs{
					{
						Name:      "arg1",
						Value:     "val1",
						SetString: false,
					},
				},
				Ports: []corev1.ServicePort{
					{
						Name:     "port1",
						Protocol: "TCP",
						Port:     8000,
					},
				},
			},
		},
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.7.0",
		Profile: "dev",
		Components: ComponentSpec{
			Ingress: IngressNginxComponent{
				Type: "sometype",
				NGINXInstallArgs: []InstallArgs{
					{
						Name:      "arg1",
						Value:     "val1",
						SetString: false,
					},
				},
				Ports: []corev1.ServicePort{
					{
						Name:     "port1",
						Protocol: "TCP",
						Port:     8000,
					},
					{
						Name:     "port2",
						Protocol: "TCP",
						Port:     9000,
					},
				},
			},
		},
	}
	err := ValidateUpgradeRequest(currentSpec, newSpec)
	return err
}

func TestGetCurrentChartVersion(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	expectedVersion, err := semver.NewSemVersion("v0.7.0")
	assert.NoError(t, err)

	version, err := getCurrentChartVersion()
	assert.NoError(t, err)
	assert.Equal(t, expectedVersion, version)
}

func TestGetCurrentChartVersionFileReadError(t *testing.T) {
	readFileFunction = func(string) ([]byte, error) {
		return []byte{}, errors.New("Unexpected file read error")
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	version, err := getCurrentChartVersion()
	assert.Error(t, err)
	assert.Nil(t, version)
}

func TestGetCurrentChartVersionBadYAML(t *testing.T) {
	const invalidChartYAML = "{"
	readFileFunction = func(string) ([]byte, error) {
		return []byte(invalidChartYAML), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	version, err := getCurrentChartVersion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error converting YAML to JSON")
	assert.Nil(t, version)
}

func TestValidateVersionInvalidVersionCheckingDisabled(t *testing.T) {
	os.Setenv(env.CheckVersionDisabled, "true")
	defer os.Unsetenv(env.CheckVersionDisabled)
	assert.NoError(t, ValidateVersion("blah"))
}

func TestValidateVersionInvalidVersion(t *testing.T) {
	assert.Error(t, ValidateVersion("blah"))
}

func TestValidateVersionBadChartYAML(t *testing.T) {
	const invalidChartYAML = "{"
	readFileFunction = func(string) ([]byte, error) {
		return []byte(invalidChartYAML), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	err := ValidateVersion("v0.7.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error converting YAML to JSON")
}

