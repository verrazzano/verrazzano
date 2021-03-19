// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

// CaCrtKey is the CA cert key in the system-tls secret
const CaCrtKey = "ca.crt"

// CaBundleKey is the CA cert key in the Elasticsearch secret
const CaBundleKey = "ca-bundle"

// KubeconfigKey is the kubeconfig key
const KubeconfigKey = "admin-kubeconfig"

// ManagedClusterNameKey is the key for the managed cluster name
const ManagedClusterNameKey = "managed-cluster-name"

// PasswordKey is the password key
const PasswordKey = "password"

// UsernameKey is the username key
const UsernameKey = "username"

// TokenKey is the key for the service account token
const TokenKey = "token"

// ESURLKey is the key for Elasticsearch URL
const ESURLKey = "es-url"

// YamlKey is the key for YAML that can be applied using kubectl
const YamlKey = "yaml"
