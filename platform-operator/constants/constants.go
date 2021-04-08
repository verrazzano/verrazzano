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

// AdminClusterConfigMapName is the name of the configmap that contains admin cluster server address
const AdminClusterConfigMapName = "verrazzano-admin-cluster"

// ServerDataKey is the key into ConfigMap data for cluster server address
const ServerDataKey = "server"

// VzConsoleIngress - the name of the ingress for verrazzano console and api
const VzConsoleIngress = "verrazzano-ingress"
