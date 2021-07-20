// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

// VerrazzanoSystemNamespace is the system namespace for verrazzano
const VerrazzanoSystemNamespace = "verrazzano-system"

// Verrazzano is the name of the verrazzano secret in the Verrazzano system namespace
const Verrazzano = "verrazzano"

// VerrazzanoMultiClusterNamespace is the multi-cluster namespace for verrazzano
const VerrazzanoMultiClusterNamespace = "verrazzano-mc"

// MCRegistrationSecret contains information which related to the managed cluster itself, such as the
// managed cluster name.
const MCRegistrationSecret = "verrazzano-cluster-registration"

// AdminClusterConfigMapName is the name of the configmap that contains admin cluster server address
const AdminClusterConfigMapName = "verrazzano-admin-cluster"

// ServerDataKey is the key into ConfigMap data for cluster server address
const ServerDataKey = "server"
