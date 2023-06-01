// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1beta1

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"testing"
)

const (
	emailAddress                 = "foo@bar.com"
	envType                      = "staging"
	testClusterResourceNamespace = "myclusterResourceNamespace"
	customCAName                 = "myCA"
)

// TestNewDefaultClusterIssuer Tests the TestNewDefaultClusterIssuer constructor
// GIVEN a call to NewDefaultClusterIssuer()
// THEN a valid ClusterIssuerComponent is returned with the default self-signed CA issuer configuration
func TestNewDefaultClusterIssuer(t *testing.T) {
	asserts := assert.New(t)
	issuer := NewDefaultClusterIssuer()

	asserts.Equal(constants.CertManagerNamespace, issuer.ClusterResourceNamespace)
	asserts.Equal(constants.DefaultVerrazzanoCASecretName, issuer.CA.SecretName)
	asserts.Nil(issuer.LetsEncrypt)

	isDefaultIssuer, err := issuer.IsDefaultIssuer()
	asserts.True(isDefaultIssuer)
	asserts.NoError(err)

	isCAIssuer, err := issuer.IsCAIssuer()
	asserts.True(isCAIssuer)
	asserts.NoError(err)

	isACMEIssuer, err := issuer.IsLetsEncryptIssuer()
	asserts.False(isACMEIssuer)
	asserts.NoError(err)
}

// TestClusterIssuerComponentIsCAIssuerIsLEIssuer Tests the IsCAIssuer and IsLetsEncryptIssuer methods
// GIVEN a call to IsCAIssuer() or IsLetsEncryptIssuer() for various configurations
// THEN the functions behave as expected
func TestClusterIssuerComponentIsCAIssuerIsLEIssuer(t *testing.T) {
	asserts := assert.New(t)

	caIssuer := ClusterIssuerComponent{
		Enabled:                  newBool(true),
		ClusterResourceNamespace: constants.CertManagerNamespace,
		IssuerConfig: IssuerConfig{
			CA: &CAIssuer{SecretName: customCAName},
		},
	}

	isCAIssuer, err := caIssuer.IsCAIssuer()
	asserts.True(isCAIssuer)
	asserts.NoError(err)

	isACMEIssuer, err := caIssuer.IsLetsEncryptIssuer()
	asserts.False(isACMEIssuer)
	asserts.NoError(err)

	leIssuer := ClusterIssuerComponent{
		Enabled:                  newBool(true),
		ClusterResourceNamespace: constants.CertManagerNamespace,
		IssuerConfig: IssuerConfig{
			LetsEncrypt: &LetsEncryptACMEIssuer{
				EmailAddress: emailAddress,
				Environment:  envType,
			},
		},
	}

	isCAIssuer, err = leIssuer.IsCAIssuer()
	asserts.False(isCAIssuer)
	asserts.NoError(err)

	isACMEIssuer, err = leIssuer.IsLetsEncryptIssuer()
	asserts.True(isACMEIssuer)
	asserts.NoError(err)
}

// TestClusterIssuerComponentNotDefaultIssuer Tests the IsDefaultIssuer method
// GIVEN a call to IsDefaultIssuer()
// WHEN the issuer configuration is not a default self-signed configuration
// THEN the function returns false, or an error if the issuer is misconfigured
func TestClusterIssuerComponentNotDefaultIssuer(t *testing.T) {
	asserts := assert.New(t)

	// Test default clusterResourceNamespace, non-default secret name
	caIssuer1 := ClusterIssuerComponent{
		Enabled:                  newBool(true),
		ClusterResourceNamespace: constants.CertManagerNamespace,
		IssuerConfig: IssuerConfig{
			CA: &CAIssuer{SecretName: "myCA"},
		},
	}

	isDefIssuer1, err := caIssuer1.IsDefaultIssuer()
	asserts.False(isDefIssuer1)
	asserts.NoError(err)

	// Test default secret name, non-default cluster resource namespace
	caIssuer2 := ClusterIssuerComponent{
		Enabled:                  newBool(true),
		ClusterResourceNamespace: testClusterResourceNamespace,
		IssuerConfig: IssuerConfig{
			CA: &CAIssuer{SecretName: constants.DefaultVerrazzanoCASecretName},
		},
	}
	isDefIssuer2, err := caIssuer2.IsDefaultIssuer()
	asserts.False(isDefIssuer2)
	asserts.NoError(err)

	// Test LE issuer
	leIssuer := ClusterIssuerComponent{
		Enabled:                  newBool(true),
		ClusterResourceNamespace: constants.CertManagerNamespace,
		IssuerConfig: IssuerConfig{
			LetsEncrypt: &LetsEncryptACMEIssuer{
				EmailAddress: emailAddress,
				Environment:  envType,
			},
		},
	}
	isDefIssuer3, err := leIssuer.IsDefaultIssuer()
	asserts.False(isDefIssuer3)
	asserts.NoError(err)
}

// TestClusterIssuerComponentBadIssuerConfig Tests the various IsXXX method
// GIVEN a call to IsXXXXIssuer()
// WHEN the issuer configuration is invalid
// THEN the functions returns false and an error
func TestClusterIssuerComponentBadIssuerConfig(t *testing.T) {
	asserts := assert.New(t)

	badConfig := ClusterIssuerComponent{
		Enabled:                  newBool(true),
		ClusterResourceNamespace: constants.CertManagerNamespace,
		IssuerConfig: IssuerConfig{
			CA: &CAIssuer{SecretName: customCAName},
			LetsEncrypt: &LetsEncryptACMEIssuer{
				EmailAddress: emailAddress,
				Environment:  envType,
			},
		},
	}

	isCAIssuer, err := badConfig.IsCAIssuer()
	asserts.Error(err)
	asserts.False(isCAIssuer)

	isLEIssuer, err := badConfig.IsLetsEncryptIssuer()
	asserts.Error(err)
	asserts.False(isLEIssuer)

	isDefIssuer, err := badConfig.IsDefaultIssuer()
	asserts.Error(err)
	asserts.False(isDefIssuer)
}
