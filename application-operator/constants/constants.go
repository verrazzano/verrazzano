// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

// VerrazzanoSystemNamespace is the system namespace for verrazzano
const VerrazzanoSystemNamespace = "verrazzano-system"

// VerrazzanoMultiClusterNamespace is the multi-cluster namespace for verrazzano
const VerrazzanoMultiClusterNamespace = "verrazzano-mc"

// MCRegistrationSecret - the name of the secret that contains the cluster registration information
const MCRegistrationSecret = "verrazzano-cluster"

// AdminKubeconfigData - the field name in MCRegistrationSecret that contains the admin cluster's kubeconfig
const AdminKubeconfigData = "admin-kubeconfig"

// ClusterNameData - the field name in MCRegistrationSecret that contains this managed cluster's name
const ClusterNameData = "managed-cluster-name"

// ElasticsearchSecretName - the name of the secret in the Verrazzano System namespace,
// that contains credentials and other details for for the admin cluster's Elasticsearch endpoint
const ElasticsearchSecretName = "verrazzano-cluster-elasticsearch"

// ElasticsearchURLData - the field name in ElasticsearchSecret that contains the admin cluster's
// Elasticsearch endpoint's URL
const ElasticsearchURLData = "url"

// ElasticsearchUsernameData - the field name in ElasticsearchSecret that contains the admin
// cluster's Elasticsearch username
const ElasticsearchUsernameData = "username"

// ElasticsearchPasswordData - the field name in ElasticsearchSecret that contains the admin
// cluster's Elasticsearch password
const ElasticsearchPasswordData = "password"

// LabelVerrazzanoManaged - constant for a Kubernetes label that is applied by Verrazzano
const LabelVerrazzanoManaged = "verrazzano-managed"

// LabelIstioInjection - constant for a Kubernetes label that is applied by Verrazzano
const LabelIstioInjection = "istio-injection"
