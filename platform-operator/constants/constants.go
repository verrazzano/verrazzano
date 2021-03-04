// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

// SystemTLS is the name of the system-tls secret in the Verrazzano system namespace
const SystemTLS = "system-tls"

// VerrazzanoSystemNamespace is the system namespace for verrazzano
const VerrazzanoSystemNamespace = "verrazzano-system"

// Verrazzano is the name of the verrazzano secret in the Verrazzano system namespace
const Verrazzano = "verrazzano"

// VerrazzanoMultiClusterNamespace is the multi-cluster namespace for verrazzano
const VerrazzanoMultiClusterNamespace = "verrazzano-mc"

// MCAdminSecret contains information to access the admin cluster, such as the admin kubeconfig.
// This secret is used by the MC agent running on the managed cluster.
const MCAdminSecret = "verrazzano-cluster-admin"

// MCElasticsearchSecret contains information to access the admin Elasticsearch from the managed cluster.
const MCElasticsearchSecret = "verrazzano-cluster-elasticsearch"

// MCRegistrationSecret contains information which related to the managed cluster itself, such as the
// managed cluster name.
const MCRegistrationSecret = "verrazzano-cluster-registration"
