// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

// authProxyValues struct representing the Helm chart values for this component
type authProxyValues struct {
	Name                 string        `json:"name,omitempty"`
	ImageName            string        `json:"imageName,omitempty"`
	ImageVersion         string        `json:"imageVersion,omitempty"`
	MetricsImageName     string        `json:"metricsImageName,omitempty"`
	MetricsImageVersion  string        `json:"metricsImageVersion,omitempty"`
	PullPolicy           string        `json:"pullPolicy,omitempty"`
	Replicas             uint32        `json:"replicas,omitempty"`
	Port                 int           `json:"port,omitempty"`
	ImpersonatorRoleName string        `json:"impersonatorRoleName,omitempty"`
	Proxy                *proxyValues  `json:"proxy,omitempty"`
	Config               *configValues `json:"config,omitempty"`
	DNS                  *dnsValues    `json:"dns,omitempty"`
	Affinity             string        `json:"affinity,omitempty"`
}

type proxyValues struct {
	OidcRealm                    string `json:"OidcRealm,omitempty"`
	PKCEClientID                 string `json:"PKCEClientID,omitempty"`
	PGClientID                   string `json:"PGClientID,omitempty"`
	RequiredRealmRole            string `json:"RequiredRealmRole,omitempty"`
	OidcCallbackPath             string `json:"OidcCallbackPath,omitempty"`
	OidcLogoutCallbackPath       string `json:"OidcLogoutCallbackPath,omitempty"`
	OidcSingleLogoutCallbackPath string `json:"OidcSingleLogoutCallbackPath,omitempty"`
	OidcProviderHost             string `json:"OidcProviderHost,omitempty"`
	OidcProviderHostInCluster    string `json:"OidcProviderHostInCluster,omitempty"`
	AuthnStateTTL                string `json:"AuthnStateTTL,omitempty"`
	MaxRequestSize               string `json:"MaxRequestSize,omitempty"`
	ProxyBufferSize              string `json:"ProxyBufferSize,omitempty"`
}

type configValues struct {
	EnvName                   string `json:"envName,omitempty"`
	DNSSuffix                 string `json:"dnsSuffix,omitempty"`
	PrometheusOperatorEnabled bool   `json:"prometheusOperatorEnabled,omitempty"`
	IngressClassName          string `json:"ingressClassName,omitempty"`
}

type dnsValues struct {
	Wildcard *wildcardDNSValues `json:"wildcard,omitempty"`
}

type wildcardDNSValues struct {
	Domain string `json:"domain,omitempty"`
}
