// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

// ClusterIssuerConfigComponentName is the name of the CertManager config component
const ClusterIssuerConfigComponentName = "cluster-issuer"

// CertManagerComponentName is the name of the Verrazzano CertManager component
const CertManagerComponentName = "cert-manager"

// CertManagerComponentJSONName is the JSON name of the verrazzano component in CRD
const CertManagerComponentJSONName = "certManager"

// CertManagerOCIDNSComponentName is the name of the OCI DNS webhook component
const CertManagerOCIDNSComponentName = "verrazzano-ocidns-webhook"

// DefaultCACertificateSecretName is the default Verrazzano self-signed CA secret
const DefaultCACertificateSecretName = "verrazzano-ca-certificate-secret" //nolint:gosec //#gosec G101
