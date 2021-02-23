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

// ElasticsearchHostData - the field name in MCRegistrationSecret that contains the admin cluster's
// Elasticsearch endpoint's host name
const ElasticsearchHostData = "elasticsearch-host"

// ElasticsearchPortData - the field name in MCRegistrationSecret that contains the admin cluster's
// Elasticsearch endpoint's port number
const ElasticsearchPortData = "elasticsearch-port"

// ElasticsearchSecretData - the field name in MCRegistrationSecret that contains the name of a
// secret containing the credentials for the admin cluster's Elasticsearch endpoint
const ElasticsearchSecretData = "elasticsearch-secret"
