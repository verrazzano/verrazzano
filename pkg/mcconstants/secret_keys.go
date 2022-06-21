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

// FluentdESCaBundleKey is the CA cert key in the Verrazzano CRD fluentd Elasticsearch secret
const FluentdESCaBundleKey = "ca-bundle"

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

// TokenKey is the key for the service account token
const TokenKey = "token"

// ESURLKey is the key for Elasticsearch URL
const ESURLKey = "es-url"

// YamlKey is the key for YAML that can be applied using kubectl
const YamlKey = "yaml"

// KeycloakURLKey is the key for Keycloak URL
const KeycloakURLKey = "keycloak-url"
