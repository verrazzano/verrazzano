// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

const (
	YamlTraits                = "traits"
	IngressTrait              = "IngressTrait"
	GatewayAPIVersion         = "networking.istio.io/v1beta1"
	VirtualServiceAPIVersion  = "networking.istio.io/v1beta1"
	DestinationRuleAPIVersion = "networking.istio.io/v1beta13"
	AuthorizationAPIVersion   = "security.istio.io/v1beta1"
	HTTPSProtocol             = "HTTPS"
	VerrazzanoClusterIssuer   = "verrazzano-cluster-issuer"
	CertificateAPIVersion     = "cert-manager.io/v1"
	CompAPIVersion            = "core.oam.dev/v1alpha2"
	MetricsTrait              = "MetricsTrait"
)

var (
	WeblogicPortNames = []string{"tcp-cbt", "tcp-ldap", "tcp-iiop", "tcp-snmp", "tcp-default", "tls-ldaps",
		"tls-default", "tls-cbts", "tls-iiops", "tcp-internal-t3", "internal-t3"}
)