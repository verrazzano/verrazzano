// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"testing"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/util/semver"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// For unit testing
const testBomFilePath = "../../../controllers//verrazzano/testdata/test_bom.json"
const invalidTestBomFilePath = "../../../controllers//verrazzano/testdata/invalid_test_bom.json"
const invalidPathTestBomFilePath = "../../../controllers//verrazzano/testdata/invalid_test_bom_path.json"

// TestValidUpgradeRequestNoCurrentVersion Tests the condition for valid upgrade where the version is not specified in the current spec
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the current version is not specified
// THEN ensure no error is returned from ValidateUpgradeRequest
func TestValidUpgradeRequestNoCurrentVersion(t *testing.T) {
	component.SetUnitTestBomFilePath(testBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.17.0",
		Profile: "dev",
	}
	assert.NoError(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestValidUpgradeRequestCurrentVersionExists Tests the condition for valid upgrade where versions are specified in both specs
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the current version is less than the current version
// THEN ensure no error is returned from ValidateUpgradeRequest
func TestValidUpgradeRequestCurrentVersionExists(t *testing.T) {
	component.SetUnitTestBomFilePath(testBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
	}()
	currentSpec := &VerrazzanoSpec{
		Version: "v0.16.0",
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.17.0",
		Profile: "dev",
	}
	assert.NoError(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestValidUpgradeRequestCurrentVersionExists Tests the condition where both specs are at the same version
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version and the current version are at the latest version
// THEN ensure no error is returned from ValidateUpgradeRequest
func TestValidUpgradeNotNecessary(t *testing.T) {
	component.SetUnitTestBomFilePath(testBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
	}()
	currentSpec := &VerrazzanoSpec{
		Version: "v0.17.0",
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.17.0",
		Profile: "dev",
	}
	assert.NoError(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestValidateUpgradeBadOldVersion Tests scenario where there is an invalid version string in the old spec (should never happen, but...code coverage)
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the current version is not valid but the new version is
// THEN ensure an error is returned from ValidateUpgradeRequest
func TestValidateUpgradeBadOldVersion(t *testing.T) {
	component.SetUnitTestBomFilePath(testBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
	}()
	currentSpec := &VerrazzanoSpec{
		Version: "blah",
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.17.0",
		Profile: "dev",
	}
	assert.Error(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestValidateUpgradeBadOldVersion Tests scenario where there is an invalid version string in the new spec
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the current version is there and valid valid but the new version is not
// THEN ensure an error is returned from ValidateUpgradeRequest
func TestValidateUpgradeBadNewVersion(t *testing.T) {
	component.SetUnitTestBomFilePath(testBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
	}()
	currentSpec := &VerrazzanoSpec{
		Version: "v0.16.0",
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
	component.SetUnitTestBomFilePath(testBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
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
	component.SetUnitTestBomFilePath(testBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.17.0",
		Profile: "prod",
	}
	assert.Error(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestProfileChangeOnlyNoVersions Tests the validate fails if no versions specified but the profile is changed
// GIVEN an edit to update a Verrazzano spec
// WHEN only the profile has changed
// THEN no error is returned from ValidateUpgradeRequest
func TestProfileChangeOnlyNoVersions(t *testing.T) {
	component.SetUnitTestBomFilePath(testBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
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
	component.SetUnitTestBomFilePath(testBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
		Version: "v0.16.0",
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
	component.SetUnitTestBomFilePath(testBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version:         "v0.17.0",
		Profile:         "dev",
		EnvironmentName: "newEnv",
	}
	assert.Error(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestValidVersionWithCertManagerChange Tests the validate fails if the upgrade version is OK but the CertManagerComponent is changed
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the CertManagerComponent field is changed
// THEN an error is returned from ValidateUpgradeRequest
func TestValidVersionWithCertManagerChange(t *testing.T) {
	component.SetUnitTestBomFilePath(testBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
		Components: ComponentSpec{
			CertManager: &CertManagerComponent{
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
		Version: "v0.17.0",
		Profile: "dev",
		Components: ComponentSpec{
			CertManager: &CertManagerComponent{
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
	component.SetUnitTestBomFilePath(testBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
		Components: ComponentSpec{
			CertManager: &CertManagerComponent{
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
		Version: "v0.17.0",
		Profile: "dev",
		Components: ComponentSpec{
			CertManager: &CertManagerComponent{
				Certificate: Certificate{
					Acme: Acme{
						Provider:     "MyProvider",
						EmailAddress: "email1@mycompany.com",
						Environment:  "someEnv",
					},
				},
			},
			DNS: &DNSComponent{
				OCI: &OCI{
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
	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})
	assert.NoError(t, runValidateWithIngressChangeTest())
}

// runValidateWithIngressChangeTest Shared test logic for ingress change validation
func runValidateWithIngressChangeTest() error {
	component.SetUnitTestBomFilePath(testBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
		Components: ComponentSpec{
			Ingress: &IngressNginxComponent{
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
		Version: "v0.17.0",
		Profile: "dev",
		Components: ComponentSpec{
			Ingress: &IngressNginxComponent{
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

// TestGetCurrentBomVersion Tests basic getBomVersion() happy path
// GIVEN a request for the current VZ Bom version
// WHEN the version in the Bom is available
// THEN no error is returned and a valid SemVersion representing the Bom version is returned
func TestGetCurrentBomVersion(t *testing.T) {
	component.SetUnitTestBomFilePath(testBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
	}()
	expectedVersion, err := semver.NewSemVersion("v0.17.0")
	assert.NoError(t, err)

	version, err := GetCurrentBomVersion()
	assert.NoError(t, err)
	assert.Equal(t, expectedVersion, version)
}

// TestGetCurrentBomVersionFileReadError Tests  getBomVersion() when there is an error reading the BOM file
// GIVEN a request for the current VZ Bom version
// WHEN an error occurs reading the BOM file from the filesystem
// THEN an error is returned and nil is returned for the Bom SemVersion
func TestGetCurrentBomVersionFileReadError(t *testing.T) {
	component.SetUnitTestBomFilePath(invalidPathTestBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
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
	component.SetUnitTestBomFilePath(invalidTestBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
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
	component.SetUnitTestBomFilePath(invalidTestBomFilePath)
	defer func() {
		component.SetUnitTestBomFilePath("")
	}()
	err := ValidateVersion("v0.17.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
}

// TestValidateActiveInstall tests that there is no Verrazzano installs active
// GIVEN a client for accessing Verrazzano resources
// WHEN no Verrazzano resources are found
// THEN ensure no error is returned from ValidateActiveInstall
func TestValidateActiveInstall(t *testing.T) {
	client := fake.NewFakeClientWithScheme(newScheme())
	assert.NoError(t, ValidateActiveInstall(client))
}

// TestValidateActiveInstallFail tests that there are active Verrazzano installs
// GIVEN a client for accessing Verrazzano resources
// WHEN a Verrazzano resources is found
// THEN ensure an error is returned from ValidateActiveInstall
func TestValidateActiveInstallFail(t *testing.T) {
	vz := &Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "test-resource",
		},
		Spec: VerrazzanoSpec{},
	}
	client := fake.NewFakeClientWithScheme(newScheme())
	assert.NoError(t, client.Create(context.TODO(), vz))
	err := ValidateActiveInstall(client)
	if assert.Error(t, err) {
		assert.Equal(t, "Only one install of Verrazzano is allowed", err.Error())
	}
}

// TestValidateInProgress tests that an install, uninstall or upgrade is not in progress
// GIVEN various Verrrazzano resource states
// THEN ensure TestValidateInProgress returns correctly
func TestValidateInProgress(t *testing.T) {
	assert.NoError(t, ValidateInProgress(Ready))
	err := ValidateInProgress(Installing)
	if assert.Error(t, err) {
		assert.Equal(t, "Updates to resource not allowed while install, uninstall or upgrade is in progress", err.Error())
	}
	err = ValidateInProgress(Uninstalling)
	if assert.Error(t, err) {
		assert.Equal(t, "Updates to resource not allowed while install, uninstall or upgrade is in progress", err.Error())
	}
	err = ValidateInProgress(Upgrading)
	if assert.Error(t, err) {
		assert.Equal(t, "Updates to resource not allowed while install, uninstall or upgrade is in progress", err.Error())
	}
}

// TestValidateOciDnsSecretBadSecret tests that validate fails if a secret in the verrazzano CR does not exist
// GIVEN a Verrazzano spec containing a secret that does not exist
// WHEN ValidateOciDNSSecret is called
// THEN an error is returned from ValidateOciDNSSecret
func TestValidateOciDnsSecretBadSecret(t *testing.T) {
	vz := Verrazzano{
		Spec: VerrazzanoSpec{
			Components: ComponentSpec{
				DNS: &DNSComponent{
					OCI: &OCI{
						OCIConfigSecret: "oci-bad-secret",
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	err := AddToScheme(scheme)
	assert.NoError(t, err)
	err = clientgoscheme.AddToScheme(scheme)
	assert.NoError(t, err)
	client := fake.NewFakeClientWithScheme(scheme)

	err = ValidateOciDNSSecret(client, &vz.Spec)
	assert.Error(t, err)
	assert.Equal(t, "The secret \"oci-bad-secret\" must be created in the verrazzano-system namespace before installing Verrrazzano for OCI DNS", err.Error())
}

// TestValidateOciDnsSecretGoodSecret tests that validate succeeds if a secret in the verrazzano CR exists
// GIVEN a Verrazzano spec containing a secret that exists
// WHEN ValidateOciDNSSecret is called
// THEN success is returned from ValidateOciDNSSecret
func TestValidateOciDnsSecretGoodSecret(t *testing.T) {
	vz := Verrazzano{
		Spec: VerrazzanoSpec{
			Components: ComponentSpec{
				DNS: &DNSComponent{
					OCI: &OCI{
						OCIConfigSecret: "oci",
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	err := AddToScheme(scheme)
	assert.NoError(t, err)
	err = clientgoscheme.AddToScheme(scheme)
	assert.NoError(t, err)
	client := fake.NewFakeClientWithScheme(scheme)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci",
			Namespace: "verrazzano-system",
		},
	}
	err = client.Create(context.TODO(), secret)
	assert.NoError(t, err)

	err = ValidateOciDNSSecret(client, &vz.Spec)
	assert.NoError(t, err)
}

// TestValidateOciDnsSecretNoOci tests that validate succeeds if the DNS component is not OCI
// GIVEN a Verrazzano spec containing a wildcard DNS component
// WHEN ValidateOciDNSSecret is called
// THEN success is returned from ValidateOciDNSSecret
func TestValidateOciDnsSecretNoOci(t *testing.T) {
	vz := Verrazzano{
		Spec: VerrazzanoSpec{
			Components: ComponentSpec{
				DNS: &DNSComponent{
					Wildcard: &Wildcard{
						Domain: "nip.io",
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	err := AddToScheme(scheme)
	assert.NoError(t, err)
	err = clientgoscheme.AddToScheme(scheme)
	assert.NoError(t, err)
	client := fake.NewFakeClientWithScheme(scheme)

	err = ValidateOciDNSSecret(client, &vz.Spec)
	assert.NoError(t, err)
}
