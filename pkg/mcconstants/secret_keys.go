// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package mcconstants - Constants in this file are keys in MultiCluster related secrets
package mcconstants

// CaCrtKey is the CA cert key in the tls secret
const CaCrtKey = "ca.crt"

// ESCaBundleKey is the ES CA cert key in the registration secret
const ESCaBundleKey = "es-ca-bundle"

// AdminCaBundleKey is the admin CA cert key in the registration secret
const AdminCaBundleKey = "ca-bundle"

// FluentdESCaBundleKey is the CA cert key in the Verrazzano CRD fluentd Opensearch secret
const FluentdESCaBundleKey = "ca-bundle"

// JaegerOSTLSKey is the key in registration secret containing TLS key used by Jaeger to connect to OpenSearch storage
// when using mutual TLS
const JaegerOSTLSKey = "jaeger-os-tls.key"

// JaegerOSTLSCertKey is the key in registration secret containing TLS cert used by Jaeger to connect to OpenSearch storage
// when using mutual TLS
const JaegerOSTLSCertKey = "jaeger-os-tls.cert"

// JaegerOSTLSCAKey is the key in registration secret containing TLS CA used by Jaeger to connect to OpenSearch storage
const JaegerOSTLSCAKey = "jaeger-os-ca.crt"

// JaegerManagedClusterSecretName is the name of the Jaeger secret in the managed cluster
// #nosec
const JaegerManagedClusterSecretName = "verrazzano-jaeger-managed-cluster"

// KubeconfigKey is the kubeconfig key
const KubeconfigKey = "admin-kubeconfig"

// ManagedClusterNameKey is the key for the managed cluster name
const ManagedClusterNameKey = "managed-cluster-name"

// RegistrationPasswordKey is the password key in registration secret
const RegistrationPasswordKey = "password"

// RegistrationUsernameKey is the username key in registration secret
const RegistrationUsernameKey = "username"

// VerrazzanoPasswordKey is the password key in Verrazzano secret
const VerrazzanoPasswordKey = "password"

// VerrazzanoUsernameKey is the username key in Verrazzano secret
const VerrazzanoUsernameKey = "username"

// JaegerOSPasswordKey is the password key in Jaeger secret to connect to the OpenSearch storage
const JaegerOSPasswordKey = "ES_PASSWORD"

// JaegerOSUsernameKey is the username key in Jaeger secret to connect to the OpenSearch storage
const JaegerOSUsernameKey = "ES_USERNAME"

// TokenKey is the key for the service account token
const TokenKey = "token"

// ESURLKey is the key for Elasticsearch URL
const ESURLKey = "es-url"

// JaegerOSURLKey is the key in registration secret containing Jaeger OpenSearch URL
const JaegerOSURLKey = "jaeger-os-url"

// YamlKey is the key for YAML that can be applied using kubectl
const YamlKey = "yaml"

// KeycloakURLKey is the key for Keycloak URL
const KeycloakURLKey = "keycloak-url"
