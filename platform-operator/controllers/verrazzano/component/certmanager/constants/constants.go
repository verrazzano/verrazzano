// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

const (
	// ClusterIssuerComponentName is the name of the CertManager config component
	ClusterIssuerComponentName = "cluster-issuer"

	// CertManagerComponentName is the name of the Verrazzano CertManager component
	CertManagerComponentName = "cert-manager"

	// CertManagerComponentJSONName is the JSON name of the verrazzano component in CRD
	CertManagerComponentJSONName = "certManager"

	// ClusterIssuerComponentJSONName - this is not a real component but declare it for compatibility
	ClusterIssuerComponentJSONName = "clusterIssuer"

	// CertManagerWebhookOCIComponentName is the name of the OCI DNS webhook component
	CertManagerWebhookOCIComponentName = "cert-manager-webhook-oci"

	// LetsEncryptProduction - LetsEncrypt production env
	LetsEncryptProduction = "production"

	// LetsEncryptStaging - LetsEncrypt staging env
	LetsEncryptStaging = "staging"
)
