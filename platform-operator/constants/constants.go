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

// MCAgentSecret contains information needed by the agent to access the admin cluster, such as the admin kubeconfig.
// This secret is used by the MC agent running on the managed cluster.
const MCAgentSecret = "verrazzano-cluster-agent"

// MCElasticsearchSecret contains information to access the admin Elasticsearch from the managed cluster.
const MCElasticsearchSecret = "verrazzano-cluster-elasticsearch"

// MCRegistrationSecret contains information which related to the managed cluster itself, such as the
// managed cluster name.
const MCRegistrationSecret = "verrazzano-cluster-registration"

// MCLocalRegistrationSecret - the name of the local secret that contains the cluster registration information.
// Thos is created at Verrazzano install.
const MCLocalRegistrationSecret = "verrazzano-local-registration"

// MCClusterRole is the role name for the role used during VMC reconcile
const MCClusterRole = "verrazzano-managed-cluster"

// MCLocalCluster is the name of the local cluster
const MCLocalCluster = "local"
