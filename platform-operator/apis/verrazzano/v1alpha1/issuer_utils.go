// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/constants"
)

func NewDefaultClusterIssuer() *ClusterIssuerComponent {
	return &ClusterIssuerComponent{
		ClusterResourceNamespace: constants.CertManagerNamespace,
		IssuerConfig: IssuerConfig{
			CA: &CAIssuer{
				SecretName: constants.DefaultVerrazzanoCASecretName,
			},
		},
	}
}

// IsCAIssuer returns true of the issuer configuration is for a CA issuer, or an error if it is misconfigured
func (c *ClusterIssuerComponent) IsCAIssuer() (bool, error) {
	if c.CA == nil && c.LetsEncrypt == nil {
		return false, fmt.Errorf("Illegal state, either CAIssuer or LetsEncrypt issuer must be configured")
	}
	if c.CA != nil && c.LetsEncrypt != nil {
		return false, fmt.Errorf("Illegal state, can not configure CAIssuer and LetsEncrypt issuer simultaneously")
	}
	return c.CA != nil, nil
}

// IsLetsEncryptIssuer returns true of the issuer configuration is for a LetsEncrypt issuer, or an error if it is misconfigured
func (c *ClusterIssuerComponent) IsLetsEncryptIssuer() (bool, error) {
	isCAIssuer, err := c.IsCAIssuer()
	if err != nil {
		return false, err
	}
	return !isCAIssuer, nil
}

// IsDefaultIssuer returns true of the issuer configuration is for the Verrazzano default self-signed issuer, or an error if it is misconfigured
func (c *ClusterIssuerComponent) IsDefaultIssuer() (bool, error) {
	isCAIssuer, err := c.IsCAIssuer()
	if err != nil {
		return false, err
	}
	return isCAIssuer && c.ClusterResourceNamespace == constants.CertManagerNamespace &&
		c.CA.SecretName == constants.DefaultVerrazzanoCASecretName, nil
}
