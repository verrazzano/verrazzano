#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install
UNINSTALL_DIR=$SCRIPT_DIR/..

. $INSTALL_DIR/common.sh
. $INSTALL_DIR/config.sh
. $UNINSTALL_DIR/uninstall-utils.sh

set -o pipefail

# Delete all of the MultiCluster resources in all namespaces
function delete_multicluster_resources {
    # Get the cluster agent secret to determine whether this is a managed cluster
    local is_managed_cluster="false"
    kubectl get secret -n verrazzano-system verrazzano-cluster-agent > /dev/null 2>&1
    if [ $? -eq 0 ]; then
      is_managed_cluster="true"
    fi
    log "is_managed_cluster is ${is_managed_cluster}"
    if [ "$is_managed_cluster" == "true" ] ; then
      log "Deleting managed cluster secrets"
      kubectl delete secret -n verrazzano-system verrazzano-cluster-agent verrazzano-cluster-registration verrazzano-cluster-elasticsearch --ignore-not-found=true
      log "Wait for one minute, since it may take the agent up to a minute to detect secret deletion and stop syncing from admin cluster"
      sleep 60
    fi
    log "Deleting VMCs"
    delete_k8s_resources verrazzanomanagedcluster ":metadata.name" "Could not delete VerrazzanoManagedClusters from Verrazzano" "" "verrazzano-mc"
    log "Deleting VerrazzanoProjects"
    delete_k8s_resources verrazzanoproject ":metadata.name" "Could not delete VerrazzanoProjects from Verrazzano" "" "verrazzano-mc"
    log "Deleting MultiClusterApplicationConfigurations"
    delete_k8s_resource_from_all_namespaces multiclusterapplicationconfigurations.clusters.verrazzano.io no
    log "Deleting MultiClusterComponents"
    delete_k8s_resource_from_all_namespaces multiclustercomponents.clusters.verrazzano.io no
    log "Deleting MultiClusterConfigMaps"
    delete_k8s_resource_from_all_namespaces multiclusterconfigmaps.clusters.verrazzano.io no
    log "Deleting MultiClusterLoggingScopes"
    delete_k8s_resource_from_all_namespaces multiclusterloggingscopes.clusters.verrazzano.io no
    log "Deleting MultiClusterSecrets"
    delete_k8s_resource_from_all_namespaces multiclustersecrets.clusters.verrazzano.io no
}

action "Deleting Multicluster resources" delete_multicluster_resources || exit 1
