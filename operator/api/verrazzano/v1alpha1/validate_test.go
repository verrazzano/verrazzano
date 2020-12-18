// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
	assert.Error(t, ValidateUpgradeRequest(currentSpec, newSpec))
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

// TestValidateInProgress tests that an install or upgrade is not in progress
// GIVEN various Verrrazzano resource states
// THEN ensure TestValidateInProgress returns correctly
func TestValidateInProgress(t *testing.T) {
	assert.NoError(t, ValidateInProgress(Ready))
	err := ValidateInProgress(Installing)
	if assert.Error(t, err) {
		assert.Equal(t, "Updates to resource not allowed while install or an upgrade is in in progress", err.Error())
	}
	err = ValidateInProgress(Upgrading)
	if assert.Error(t, err) {
		assert.Equal(t, "Updates to resource not allowed while install or an upgrade is in in progress", err.Error())
	}
}
