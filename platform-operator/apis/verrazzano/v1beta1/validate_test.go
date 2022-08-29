// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1beta1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

// For unit testing
const (
	testBomFilePath         = "testdata/test_bom.json"
	testRollbackBomFilePath = "testdata/rollback_bom.json"

	v0160 = "v0.16.0"
	v0170 = "v0.17.0"
	v0180 = "v0.18.0"
	v100  = "v1.0.0"
	v110  = "v1.1.0"
	v120  = "v1.2.0"
)

// TestValidUpgradeRequestNoCurrentVersion Tests the condition for valid upgrade where the version is not specified in the current spec
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the current version is not specified
// THEN ensure no error is returned from ValidateUpgradeRequest
func TestValidUpgradeRequestNoCurrentVersion(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
		},
		Status: VerrazzanoStatus{
			Version: v100,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v110,
			Profile: "dev",
		},
	}
	assert.NoError(t, validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version))
}

// TestUpdateBeforeUpgrade Tests ValidateUpgradeRequest
// GIVEN an edit to update a Verrazzano spec
// WHEN no new version is requested and the spec has been modified
// THEN ensure an error is returned
func TestUpdateBeforeUpgrade(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
		},
		Status: VerrazzanoStatus{
			Version: v100,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
			Components: ComponentSpec{
				DNS: &DNSComponent{
					Wildcard: &Wildcard{
						Domain: "sslip.io",
					},
				},
			},
		},
	}
	assert.Error(t, validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version))
}

// TestUpdateWithUpgrade Tests ValidateUpgradeRequest
// GIVEN an edit to update a Verrazzano spec
// WHEN a valid new version is requested and the spec has been modified
// THEN ensure no error is returned
func TestUpdateWithUpgrade(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
		},
		Status: VerrazzanoStatus{
			Version: v100,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
			Version: v110,
			Components: ComponentSpec{
				DNS: &DNSComponent{
					Wildcard: &Wildcard{
						Domain: "sslip.io",
					},
				},
			},
		},
	}
	assert.NoError(t, validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version))
}

// TestUpgradeNewVerDoesNotMatchBOMVer Tests ValidateUpgradeRequest
// GIVEN an edit to update a Verrazzano spec
// WHEN a valid new version is requested that is less than the BOM version
// THEN ensure an error is returned
func TestUpgradeNewVerDoesNotMatchBOMVer(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
		},
		Status: VerrazzanoStatus{
			Version: v100,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
			Version: "v0.9.0",
		},
	}
	assert.Error(t, validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version))
}

// TestUpgradeNewVerLessThanCurrentCer Tests ValidateUpgradeRequest
// GIVEN an edit to update a Verrazzano spec
// WHEN a valid new version is requested that is less than the current spec version
// THEN ensure an error is returned
func TestUpgradeNewVerLessThanCurrentCer(t *testing.T) {
	// This case can probably never happen in reality?
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v120,
			Profile: Dev,
		},
		Status: VerrazzanoStatus{
			Version: v120,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: Dev,
			Version: v110,
		},
	}
	assert.Error(t, validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version))
}

// TestValidUpgradeRequestCurrentVersionExists Tests the condition for valid upgrade where versions are specified in both specs
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the current version is less than the current version
// THEN ensure no error is returned from ValidateUpgradeRequest
func TestValidUpgradeRequestCurrentVersionExists(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v0160,
			Profile: Dev,
		},
		Status: VerrazzanoStatus{
			Version: v0160,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v110,
			Profile: Dev,
		},
	}
	assert.NoError(t, validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version))
}

// TestValidUpgradeRequestCurrentVersionExists Tests the condition where both specs are at the same version
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version and the current version are at the latest version
// THEN ensure no error is returned from ValidateUpgradeRequest
func TestValidUpgradeNotNecessary(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v0170,
			Profile: Dev,
		},
		Status: VerrazzanoStatus{
			Version: v0170,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v110,
			Profile: Dev,
		},
	}
	assert.NoError(t, validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version))
}

// TestValidateUpgradeBadOldVersion Tests scenario where there is an invalid version string in the old spec (should never happen, but...code coverage)
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the current version is not valid but the new version is
// THEN ensure an error is returned from ValidateUpgradeRequest
func TestValidateUpgradeBadOldVersion(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "blah",
			Profile: Dev,
		},
		Status: VerrazzanoStatus{
			Version: v0160,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v0170,
			Profile: Dev,
		},
	}
	assert.Error(t, validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version))
}

// TestValidateUpgradeBadOldVersion Tests scenario where there is an invalid version string in the new spec
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the current version is there and valid valid but the new version is not
// THEN ensure an error is returned from ValidateUpgradeRequest
func TestValidateUpgradeBadNewVersion(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v0160,
			Profile: Dev,
		},
		Status: VerrazzanoStatus{
			Version: v0160,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "blah",
			Profile: Dev,
		},
	}
	assert.Error(t, validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version))
}

// TestNoVersionsSpecified Tests ValidateUpgradeRequest
// GIVEN an edit to update a Verrazzano spec
// WHEN the new version and the current version are not specified, but the installed version is up to date
// THEN no error is returned from ValidateUpgradeRequest
func TestNoVersionsSpecified(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: Dev,
		},
		Status: VerrazzanoStatus{
			Version: v110,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: Dev,
		},
	}
	assert.NoError(t, validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version))
}

// TestValidValidVersionWithProfileChange Tests the validate fails if the upgrade version is OK but the profile is changed
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the profile field is changed
// THEN an error is returned from ValidateUpgradeRequest
func TestValidVersionWithProfileChange(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: Dev,
		},
		Status: VerrazzanoStatus{
			Version: v0160,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v0170,
			Profile: Prod,
		},
	}
	assert.Error(t, validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version))
}

// TestValidVersionWithEnvNameChange Tests the validate fails if the upgrade version is OK but the EnvironmentName is changed
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the EnvironmentName field is changed
// THEN an error is returned from ValidateUpgradeRequest
func TestValidVersionWithEnvNameChange(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: Dev,
		},
		Status: VerrazzanoStatus{
			Version: v0160,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version:         v0170,
			Profile:         Dev,
			EnvironmentName: "newEnv",
		},
	}
	assert.Error(t, validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version))
}

// TestValidVersionWithCertManagerChange Tests the validate fails if the upgrade version is OK but the CertManagerComponent is changed
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the CertManagerComponent field is changed
// THEN an error is returned from ValidateUpgradeRequest
func TestValidVersionWithCertManagerChange(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: Dev,
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
		},
		Status: VerrazzanoStatus{
			Version: v0160,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v0170,
			Profile: Dev,
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
		},
	}
	assert.Error(t, validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version))
}

// TestValidVersionWithNewDNS Tests the validate fails if the upgrade version is OK but the DNS component is added
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the DNS component is added
// THEN an error is returned from ValidateUpgradeRequest
func TestValidVersionWithNewDNS(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: Dev,
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
		},
		Status: VerrazzanoStatus{
			Version: v0160,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v0170,
			Profile: Dev,
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
		},
	}
	assert.Error(t, validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version))
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
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: Dev,
			Components: ComponentSpec{
				Ingress: &IngressNginxComponent{
					Type: "sometype",
					//NGINXInstallArgs: []InstallArgs{
					//	{
					//		Name:      "arg1",
					//		Value:     "val1",
					//		SetString: false,
					//	},
					//},
					Ports: []corev1.ServicePort{
						{
							Name:     "port1",
							Protocol: "TCP",
							Port:     8000,
						},
					},
				},
			},
		},
		Status: VerrazzanoStatus{
			Version: v0160,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v0170,
			Profile: Dev,
			Components: ComponentSpec{
				Ingress: &IngressNginxComponent{
					Type: "sometype",
					//NGINXInstallArgs: []InstallArgs{
					//	{
					//		Name:      "arg1",
					//		Value:     "val1",
					//		SetString: false,
					//	},
					//},
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
		},
	}
	err := validators.ValidateUpgradeRequest(newSpec.Spec.Version, currentSpec.Status.Version, currentSpec.Spec.Version)
	return err
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
	vzOld := Verrazzano{}

	vzOld.Status.State = VzStateReady
	assert.NoError(t, ValidateInProgress(&vzOld))

	vzOld.Status.State = VzStatePaused
	assert.NoError(t, ValidateInProgress(&vzOld))

	vzOld.Status.State = VzStateReconciling
	err := ValidateInProgress(&vzOld)
	assert.NoError(t, err)

	vzOld.Status.State = VzStateUninstalling
	err = ValidateInProgress(&vzOld)
	if assert.Error(t, err) {
		assert.Equal(t, validators.ValidateInProgressError, err.Error())
	}

	vzOld.Status.State = VzStateUpgrading
	err = ValidateInProgress(&vzOld)
	if assert.Error(t, err) {
		assert.Equal(t, validators.ValidateInProgressError, err.Error())
	}
}

// TestValidateEnable tests that a component can be enabled when Verrazzano is ready or installing
// GIVEN various Verrrazzano resource states
// THEN ensure TestValidateInProgress returns correctly
func TestValidateEnable(t *testing.T) {
	tests := []struct {
		testName string
		vzOld    Verrazzano
		values   []string
		expected string
	}{
		{
			testName: "1",
			vzOld: Verrazzano{
				Spec: VerrazzanoSpec{
					Version:         "",
					Profile:         "",
					EnvironmentName: "",
					Components: ComponentSpec{
						CoherenceOperator: &CoherenceOperatorComponent{
							Enabled: newBool(false),
						},
					},
				},
			},
		},
		{
			testName: "2",
			vzOld: Verrazzano{
				Spec: VerrazzanoSpec{
					Version:         "",
					Profile:         "",
					EnvironmentName: "",
					Components: ComponentSpec{
						WebLogicOperator: &WebLogicOperatorComponent{
							Enabled: newBool(false),
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			test.vzOld.Status.State = VzStateReady
			err := ValidateInProgress(&test.vzOld)
			assert.NoError(t, err, "Unexpected error enabling Coherence")

			test.vzOld.Status.State = VzStateReconciling
			err = ValidateInProgress(&test.vzOld)
			assert.NoError(t, err, "Unexpected error enabling Coherence")

			test.vzOld.Status.State = VzStatePaused
			err = ValidateInProgress(&test.vzOld)
			assert.NoError(t, err, "Unexpected error enabling Coherence")

			test.vzOld.Status.State = VzStateUpgrading
			err = ValidateInProgress(&test.vzOld)
			if assert.Error(t, err) {
				assert.Equal(t, validators.ValidateInProgressError, err.Error())
			}

			test.vzOld.Status.State = VzStateUninstalling
			err = ValidateInProgress(&test.vzOld)
			if assert.Error(t, err) {
				assert.Equal(t, validators.ValidateInProgressError, err.Error())
			}
		})
	}
}

// TestValidateOciDnsSecretBadSecret tests that validate fails if a secret in the Verrazzano CR does not exist
// GIVEN a Verrazzano spec containing a secret that does not exist
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
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

	err = validateOCISecrets(client, &vz.Spec)
	assert.Error(t, err)
	assert.Equal(t, "Secret \"oci-bad-secret\" must be created in the \"verrazzano-install\" namespace before installing Verrrazzano", err.Error())
}

// TestValidateOciDnsSecretUserAuth tests validateOCISecrets
// GIVEN a Verrazzano spec containing an OCI DNS user-auth secret that exists
// WHEN validateOCISecrets is called
// THEN success is returned from validateOCISecrets
func TestValidateOciDnsSecretUserAuth(t *testing.T) {
	runValidateOCIDNSAuthTest(t, validators.UserPrincipal)
}

// TestValidateOciDnsSecretInstancePrincipalAuth tests validateOCISecrets
// GIVEN a Verrazzano spec containing an OCI DNS instance-principal auth secret that exists
// WHEN validateOCISecrets is called
// THEN success is returned from validateOCISecrets
func TestValidateOciDnsSecretInstancePrincipalAuth(t *testing.T) {
	runValidateOCIDNSAuthTest(t, validators.InstancePrincipal)
}

func runValidateOCIDNSAuthTest(t *testing.T, authType validators.AuthenticationType) {
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

	var ociConfig validators.OciAuth
	switch authType {
	case validators.UserPrincipal:
		key, err := generateTestPrivateKey()
		assert.NoError(t, err)
		ociConfig = validators.OciAuth{
			Auth: validators.AuthData{
				Region:      "us-ashburn-1",
				Tenancy:     "my-tenancy",
				User:        "my-user",
				Fingerprint: "a-fingerprint",
				AuthType:    authType,
				Key:         string(key),
			},
		}
	default:
		ociConfig = validators.OciAuth{
			Auth: validators.AuthData{
				AuthType: authType,
			},
		}
	}

	secretData, err := yaml.Marshal(&ociConfig)
	assert.NoError(t, err, "Error marshalling test data")

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci",
			Namespace: constants.VerrazzanoInstallNamespace,
		},
		Data: map[string][]byte{
			validators.OciDNSSecretFileName: secretData,
		},
	}
	err = client.Create(context.TODO(), secret)
	assert.NoError(t, err)

	err = validateOCISecrets(client, &vz.Spec)
	assert.NoError(t, err)
}

// TestValidateOciDnsSecretNoDataKeys tests validateOCISecrets
// GIVEN a Verrazzano spec containing an OCI DNS instance-principal auth secret that exists but has no data keys
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
func TestValidateOciDnsSecretNoDataKeys(t *testing.T) {
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
			Namespace: constants.VerrazzanoInstallNamespace,
		},
	}
	err = client.Create(context.TODO(), secret)
	assert.NoError(t, err)

	err = validateOCISecrets(client, &vz.Spec)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Secret \"oci\" for OCI DNS should have one data key, found 0")
	}
}

// TestValidateOciDnsSecretTooManyDataKeys tests validateOCISecrets
// GIVEN a Verrazzano spec containing an OCI DNS instance-principal auth secret that exists but has more than one data key
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
func TestValidateOciDnsSecretTooManyDataKeys(t *testing.T) {
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
			Namespace: constants.VerrazzanoInstallNamespace,
		},
		Data: map[string][]byte{
			validators.OciDNSSecretFileName:        []byte("value1"),
			validators.OciDNSSecretFileName + "-2": []byte("value2"),
		},
	}
	err = client.Create(context.TODO(), secret)
	assert.NoError(t, err)

	err = validateOCISecrets(client, &vz.Spec)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Secret \"oci\" for OCI DNS should have one data key, found 2")
	}

}

// TestValidateOciDnsSecretInvalidAPIKey tests validateOCISecrets
// GIVEN a Verrazzano spec containing a secret that exists but with an invalid private key
// WHEN validateOCISecrets is called
// THEN an error returned from validateOCISecrets
func TestValidateOciDnsSecretInvalidAPIKey(t *testing.T) {
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

	assert.NoError(t, err)
	ociConfig := validators.OciAuth{
		Auth: validators.AuthData{
			Region:      "us-ashburn-1",
			Tenancy:     "my-tenancy",
			User:        "my-user",
			Fingerprint: "a-fingerprint",
			AuthType:    validators.UserPrincipal,
			Key:         "foo",
		},
	}
	secretData, err := yaml.Marshal(&ociConfig)
	assert.NoError(t, err, "Error marshalling test data")

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci",
			Namespace: constants.VerrazzanoInstallNamespace,
		},
		Data: map[string][]byte{
			validators.OciDNSSecretFileName: secretData,
		},
	}
	err = client.Create(context.TODO(), secret)
	assert.NoError(t, err)

	err = validateOCISecrets(client, &vz.Spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Private key in secret \"oci\" is either empty or not a valid key in PEM format")
}

// TestValidateOciDnsSecretInvalidAuthType tests validateOCISecrets
// GIVEN a Verrazzano spec containing a secret that exists but with an invalid OCI Auth type
// WHEN validateOCISecrets is called
// THEN an error returned from validateOCISecrets
func TestValidateOciDnsSecretInvalidAuthType(t *testing.T) {
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

	key, err := generateTestPrivateKey()
	assert.NoError(t, err)
	ociConfig := validators.OciAuth{
		Auth: validators.AuthData{
			Region:      "us-ashburn-1",
			Tenancy:     "my-tenancy",
			User:        "my-user",
			Fingerprint: "a-fingerprint",
			AuthType:    "InvalidAuthType",
			Key:         string(key),
		},
	}
	secretData, err := yaml.Marshal(&ociConfig)
	assert.NoError(t, err, "error marshalling test data")

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci",
			Namespace: constants.VerrazzanoInstallNamespace,
		},
		Data: map[string][]byte{
			validators.OciDNSSecretFileName: secretData,
		},
	}
	err = client.Create(context.TODO(), secret)
	assert.NoError(t, err)

	err = validateOCISecrets(client, &vz.Spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("Authtype \"InvalidAuthType\" in OCI secret must be either '%s' or '%s'", validators.UserPrincipal, validators.InstancePrincipal))
}

// TestValidateOciDnsSecretNoOci tests that validate succeeds if the DNS component is not OCI
// GIVEN a Verrazzano spec containing a wildcard DNS component
// WHEN validateOCISecrets is called
// THEN success is returned from validateOCISecrets
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

	err = validateOCISecrets(client, &vz.Spec)
	assert.NoError(t, err)
}

// TestValidateFluentdOCISecretGoodSecretWithPassphrase tests validateOCISecrets
// GIVEN a Verrazzano spec containing a fluentd configuration with a valid Fluentd OCI secret that exists with a passphrase
// WHEN validateOCISecrets is called
// THEN success is returned from validateOCISecrets
func TestValidateFluentdOCISecretGoodSecretWithPassphrase1(t *testing.T) {
	ociConfigBytes := `
[DEFAULT]
user=ocid1.user.oc1..sfafasfasfsdafas
tenancy=ocid1.tenancy.oc1..sfdasfsafas
region=us-ashburn-1
fingerprint=a0:bb:dd:c2:dd:e0:f1:fa:cd:d1:8a:11:bb:c0:f1:55
key_file=/root/.oci/key
pass_phrase=apassphrase
`
	runTestFluentdOCIConfig(t, ociConfigBytes)
}

// TestValidateFluentdOCISecretGoodSecretNoPassphrase tests validateOCISecrets
// GIVEN a Verrazzano spec containing a fluentd configuration with a valid Fluentd OCI secret that exists without a passphrase
// WHEN validateOCISecrets is called
// THEN success is returned from validateOCISecrets
func TestValidateFluentdOCISecretGoodSecretNoPassphrase(t *testing.T) {
	ociConfigBytes := `
[DEFAULT]
user=ocid1.user.oc1..sfafasfasfsdafas
tenancy=ocid1.tenancy.oc1..sfdasfsafas
region=us-ashburn-1
fingerprint=a0:bb:dd:c2:dd:e0:f1:fa:cd:d1:8a:11:bb:c0:f1:55
key_file=/root/.oci/key
`
	runTestFluentdOCIConfig(t, ociConfigBytes)
}

// TestValidateFluentdOCISecretBadProfileKey tests validateOCISecrets
// GIVEN a Verrazzano spec containing a fluentd configuration with an OCI secret with a bad profile key
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
func TestValidateFluentdOCISecretBadProfileKey(t *testing.T) {
	ociConfigBytes := `
[blah]
user=ocid1.user.oc1..sfafasfasfsdafas
tenancy=ocid1.tenancy.oc1..sfdasfsafas
region=us-ashburn-1
fingerprint=a0:bb:dd:c2:dd:e0:f1:fa:cd:d1:8a:11:bb:c0:f1:55
key_file=/root/.oci/key
`
	runTestFluentdOCIConfig(t, ociConfigBytes, "configuration file did not contain profile: DEFAULT")
}

// TestValidateFluentdOCISecretNoProfileKey tests validateOCISecrets
// GIVEN a Verrazzano spec containing a fluentd configuration with an OCI secret with no OCI profile key
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
func TestValidateFluentdOCISecretNoProfileKey(t *testing.T) {
	ociConfigBytes := `
user=ocid1.user.oc1..sfafasfasfsdafas
tenancy=ocid1.tenancy.oc1..sfdasfsafas
region=us-ashburn-1
fingerprint=a0:bb:dd:c2:dd:e0:f1:fa:cd:d1:8a:11:bb:c0:f1:55
key_file=/root/.oci/key
`
	runTestFluentdOCIConfig(t, ociConfigBytes, "configuration file did not contain profile: DEFAULT")
}

// TestValidateFluentdOCISecretNoProfileKey tests validateOCISecrets
// GIVEN a Verrazzano spec containing a fluentd configuration with an OCI secret with an empty user OCID
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
func TestValidateFluentdOCISecretNoUserOCID(t *testing.T) {
	ociConfigBytes := `
[DEFAULT]
user=
tenancy=ocid1.tenancy.oc1..sfdasfsafas
region=us-ashburn-1
fingerprint=a0:bb:dd:c2:dd:e0:f1:fa:cd:d1:8a:11:bb:c0:f1:55
key_file=/root/.oci/key
`
	runTestFluentdOCIConfig(t, ociConfigBytes, "User OCID not specified in Fluentd OCI config secret \"fluentd-oci\"")
}

// TestValidateFluentdOCISecretNoTenancyOCID tests validateOCISecrets
// GIVEN a Verrazzano spec containing a fluentd configuration with an OCI secret with an empty tenancy OCID
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
func TestValidateFluentdOCISecretNoTenancyOCID(t *testing.T) {
	ociConfigBytes := `
[DEFAULT]
user=ocid1.user.oc1..sfafasfasfsdafas
tenancy=
region=us-ashburn-1
fingerprint=a0:bb:dd:c2:dd:e0:f1:fa:cd:d1:8a:11:bb:c0:f1:55
key_file=/root/.oci/key
`
	runTestFluentdOCIConfig(t, ociConfigBytes, "Tenancy OCID not specified in Fluentd OCI config secret \"fluentd-oci\"")
}

// TestValidateFluentdOCISecretNoRegion tests validateOCISecrets
// GIVEN a Verrazzano spec containing a fluentd configuration with an OCI secret with an empty region name
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
func TestValidateFluentdOCISecretNoRegion(t *testing.T) {
	ociConfigBytes := `
[DEFAULT]
user=ocid1.user.oc1..sfafasfasfsdafas
tenancy=ocid1.tenancy.oc1..sfdasfsafas
region=
fingerprint=a0:bb:dd:c2:dd:e0:f1:fa:cd:d1:8a:11:bb:c0:f1:55
key_file=/root/.oci/key
`
	runTestFluentdOCIConfig(t, ociConfigBytes, "region can not be empty or have spaces")
}

// TestValidateFluentdOCISecretNoFingerprint tests validateOCISecrets
// GIVEN a Verrazzano spec containing a fluentd configuration with an OCI secret with an empty key fingerprint
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
func TestValidateFluentdOCISecretNoFingerprint(t *testing.T) {
	ociConfigBytes := `
[DEFAULT]
user=ocid1.user.oc1..sfafasfasfsdafas
tenancy=ocid1.tenancy.oc1..sfdasfsafas
region=us-ashburn-1
fingerprint=
key_file=/root/.oci/key
`
	runTestFluentdOCIConfig(t, ociConfigBytes, "Fingerprint not specified in Fluentd OCI config secret \"fluentd-oci\"")
}

func runTestFluentdOCIConfig(t *testing.T, ociConfigBytes string, errorMsg ...string) {
	const ociSecretName = "fluentd-oci"
	vz := Verrazzano{
		Spec: VerrazzanoSpec{
			Components: ComponentSpec{
				Fluentd: &FluentdComponent{
					OCI: &OciLoggingConfiguration{
						APISecret: ociSecretName,
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

	key, err := generateTestPrivateKey()
	assert.NoError(t, err)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ociSecretName,
			Namespace: constants.VerrazzanoInstallNamespace,
		},
		Data: map[string][]byte{
			validators.FluentdOCISecretConfigEntry:     []byte(ociConfigBytes),
			validators.FluentdOCISecretPrivateKeyEntry: key,
		},
	}
	err = client.Create(context.TODO(), secret)
	assert.NoError(t, err)

	err = validateOCISecrets(client, &vz.Spec)
	if len(errorMsg) > 0 {
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), errorMsg[0])
		}
	} else {
		assert.NoError(t, err)
	}
}

// TestValidateFluentdOCISecretInvalidKeyFormat tests validateOCISecrets
// GIVEN a Verrazzano spec containing a fluentd configuration with a Fluentd OCI secret with a key not in PEM format
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
func TestValidateFluentdOCISecretInvalidKeyFormat(t *testing.T) {
	runFluentdInvalidKeyTest(t, []byte("foo"), "not a valid key")
}

// TestValidateFluentdOCISecretNoKeyData tests validateOCISecrets
// GIVEN a Verrazzano spec containing a fluentd configuration with a Fluentd OCI secret with a empty key
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
func TestValidateFluentdOCISecretNoKeyData(t *testing.T) {
	runFluentdInvalidKeyTest(t, []byte{}, "Private key in secret \"fluentd-oci\" is either empty or not a valid key in PEM format")
}

func runFluentdInvalidKeyTest(t *testing.T, key []byte, msgSnippet string) {
	const ociSecretName = "fluentd-oci"
	vz := Verrazzano{
		Spec: VerrazzanoSpec{
			Components: ComponentSpec{
				Fluentd: &FluentdComponent{
					OCI: &OciLoggingConfiguration{
						APISecret: ociSecretName,
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

	ociConfigBytes := `
[DEFAULT]
user=my-user
tenancy=my-tenancy
region=us-ashburn-1
fingerprint=a-fingerprint
key_file=/root/.oci/key
`

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ociSecretName,
			Namespace: constants.VerrazzanoInstallNamespace,
		},
		Data: map[string][]byte{
			validators.FluentdOCISecretConfigEntry:     []byte(ociConfigBytes),
			validators.FluentdOCISecretPrivateKeyEntry: key,
		},
	}
	err = client.Create(context.TODO(), secret)
	assert.NoError(t, err)

	err = validateOCISecrets(client, &vz.Spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), msgSnippet)
}

// TestValidateFluentdOCISecretMissingKeySection tests validateOCISecrets
// GIVEN a Verrazzano spec containing a fluentd configuration with a Fluentd OCI secret that has a missing API key
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
func TestValidateFluentdOCISecretMissingKeySection(t *testing.T) {
	const ociSecretName = "fluentd-oci"
	vz := Verrazzano{
		Spec: VerrazzanoSpec{
			Components: ComponentSpec{
				Fluentd: &FluentdComponent{
					OCI: &OciLoggingConfiguration{
						APISecret: ociSecretName,
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

	ociConfigBytes := `
[DEFAULT]
user=my-user
tenancy=my-tenancy
region=us-ashburn-1
fingerprint=a-fingerprint
key_file=/root/.oci/key
`
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ociSecretName,
			Namespace: constants.VerrazzanoInstallNamespace,
		},
		Data: map[string][]byte{
			validators.FluentdOCISecretConfigEntry: []byte(ociConfigBytes),
		},
	}
	err = client.Create(context.TODO(), secret)
	assert.NoError(t, err)

	err = validateOCISecrets(client, &vz.Spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("Expected entry \"%s\" not found in secret \"%s\"", validators.FluentdOCISecretPrivateKeyEntry, ociSecretName))
}

// TestValidateFluentdOCISecretMissingConfigSection tests validateOCISecrets
// GIVEN a Verrazzano spec containing a fluentd configuration with a Fluentd OCI secret that has a missing OCI Config key
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
func TestValidateFluentdOCISecretMissingConfigSection(t *testing.T) {
	const ociSecretName = "fluentd-oci"
	vz := Verrazzano{
		Spec: VerrazzanoSpec{
			Components: ComponentSpec{
				Fluentd: &FluentdComponent{
					OCI: &OciLoggingConfiguration{
						APISecret: ociSecretName,
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

	key, err := generateTestPrivateKey()
	assert.NoError(t, err)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ociSecretName,
			Namespace: constants.VerrazzanoInstallNamespace,
		},
		Data: map[string][]byte{
			validators.FluentdOCISecretPrivateKeyEntry: key,
		},
	}
	err = client.Create(context.TODO(), secret)
	assert.NoError(t, err)

	err = validateOCISecrets(client, &vz.Spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Did not find OCI configuration in secret \"fluentd-oci\"")
}

// TestValidateFluentdOCISecretMissingConfigSection tests validateOCISecrets
// GIVEN a Verrazzano spec containing a fluentd configuration with a Fluentd OCI secret that does not exist
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
func TestValidateFluentdOCISecretMissingSecret(t *testing.T) {
	const ociSecretName = "fluentd-oci"
	vz := Verrazzano{
		Spec: VerrazzanoSpec{
			Components: ComponentSpec{
				Fluentd: &FluentdComponent{
					OCI: &OciLoggingConfiguration{
						APISecret: ociSecretName,
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

	err = validateOCISecrets(client, &vz.Spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("Secret \"%s\" must be created in the \"%s\" namespace", ociSecretName, constants.VerrazzanoInstallNamespace))
}

// TestValidateFluentdOCISecretInvalidKeyPath tests validateOCISecrets
// GIVEN a Verrazzano spec containing a fluentd configuration with a Fluentd OCI secret that an incorrect key path
// WHEN validateOCISecrets is called
// THEN an error is returned from validateOCISecrets
func TestValidateFluentdOCISecretInvalidKeyPath(t *testing.T) {
	ociConfigBytes := `
[DEFAULT]
user=my-user
tenancy=my-tenancy
region=us-ashburn-1
fingerprint=a-fingerprint
key_file=invalid/path/to/key
`
	runTestFluentdOCIConfig(t, ociConfigBytes, "Unexpected or missing value for the Fluentd OCI key file location in secret \"fluentd-oci\", should be \"/root/.oci/key\"")
}

func newBool(v bool) *bool {
	b := v
	return &b
}

// TestValidateProfileEmptyProfile Tests ValidateProfile() for empty profile
// GIVEN a request for empty profile
// WHEN the profile provided is empty
// THEN no error is returned
func TestValidateProfileEmptyProfile(t *testing.T) {
	assert.NoError(t, ValidateProfile(""))
}

// TestValidateProfileEmptyProfile Tests ValidateProfile() for d pevrofile
// GIVEN a request for dev profile
// WHEN the profile provided is dev
// THEN no error is returned
func TestValidateProfileDevProfile(t *testing.T) {
	assert.NoError(t, ValidateProfile(Dev))
}

// TestValidateProfileInvalidProfile Tests ValidateProfile() for invalid profile
// GIVEN a request for invalid profile
// WHEN the profile provided is invalid
// THEN an error is returned
func TestValidateProfileInvalidProfile(t *testing.T) {
	assert.Error(t, ValidateProfile("wrong-profile"))
}

func TestValidateInstallOverrides(t *testing.T) {
	assert := assert.New(t)

	testNoOverride := []Overrides{{}}

	testBadOverride := []Overrides{
		{
			ConfigMapRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{},
				Key:                  "",
				Optional:             nil,
			},
			SecretRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{},
				Key:                  "",
				Optional:             nil,
			},
		},
	}

	testGoodOverride := []Overrides{
		{
			ConfigMapRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{},
				Key:                  "",
				Optional:             nil,
			},
		},
	}

	err1 := ValidateInstallOverrides(testBadOverride)
	err2 := ValidateInstallOverrides(testNoOverride)
	err3 := ValidateInstallOverrides(testGoodOverride)

	assert.Error(err1)
	assert.Error(err2)
	assert.NoError(err3)
}

var testKey = []byte{}

// Generate RSA for testing.
func generateTestPrivateKey() ([]byte, error) {
	var err error
	if len(testKey) == 0 { // cache the test key, we only need one valid one and it can be expensive
		testKey, err = generateTestPrivateKeyWithType("RSA PRIVATE KEY")
	}
	return testKey, err
}

// Generate RSA for testing with the specified type
func generateTestPrivateKeyWithType(keyType string) ([]byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return []byte{}, err
	}

	// Encode private key to PKCS#1 ASN.1 PEM.
	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  keyType,
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)
	return keyPEM, nil
}
