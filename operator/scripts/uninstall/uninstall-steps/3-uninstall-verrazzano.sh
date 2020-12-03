#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install
UNINSTALL_DIR=$SCRIPT_DIR/..

. $INSTALL_DIR/common.sh
. $UNINSTALL_DIR/uninstall-utils.sh

set -o pipefail

# This makes an attempt to uninstall the WebLogic Kubernetes operator, ignoring errors so that this can work
# in the case where there is a partial installation

function uninstall_weblogic_operator {
  log "Uninstalling WebLogic Kubernetes operator"
  helm delete weblogic-operator --namespace weblogic-operator

  log "Delete weblogic-operator namespace"
  kubectl delete ns weblogic-operator

  return 0
}


function delete_verrazzano() {
  # delete helm installation of Verrazzano
  log "Deleting Verrazzano"
  helm ls -A \
    | awk '/verrazzano/ {print $1}' \
    | xargsr helm uninstall -n verrazzano-system \
    || err_return $? "Could not delete verrazzano from helm" || return $? # return on pipefail

  # delete verrazzano-managed-cluster-local secret
  log "Deleting Verrazzano secrets"
  kubectl delete secret verrazzano-managed-cluster-local --ignore-not-found=true || err_return $? "Could not delete secrets from Verrazzano" || return $?

  # delete crds
  log "Deleting Verrazzano crd finalizers"
  patch_k8s_resources crds ":metadata.name" "Could not remove finalizers from CustomResourceDefinitions in Verrazzano" '/verrazzano.io/' '{"metadata":{"finalizers":null}}' \
    || return $? # return on pipefail

  log "Deleting Verrazzano crds"
  delete_k8s_resources crds ":metadata.name" "Could not delete CustomResourceDefinitions from Verrazzano" '/verrazzano.io/ && ! /verrazzanos.install.verrazzano.io/' \
    || return $? # return on pipefail

  # deleting certificatesigningrequests
  log "Deleting CertificateSigningRequests"
  delete_k8s_resources csr ":metadata.name" "Could not delete CertificateSigningRequests from Verrazzano" '/csr-/' \
    || return $? # return on pipefail

  log "Deleting ClusterRoleBindings"
  # deleting clusterrolebindings
  delete_k8s_resources clusterrolebinding ":metadata.name,:metadata.labels" "Could not delete ClusterRoleBindings from Verrazzano" '/verrazzano/ && ! /verrazzano-platform-operator/ && ! /verrazzano-install/ {print $1}' \
    || return $? # return on pipefail

  # deleting clusterroles
  log "Deleting ClusterRoles"
  delete_k8s_resources clusterrole ":metadata.name,:metadata.labels" "Could not delete ClusterRoles from Verrazzano" '/verrazzano/ {print $1}' \
    || return $? # return on pipefail

  # deleting namespaces
  log "Deleting Verrazzano namespace finalizers"
  # delete namespace finalizers
  patch_k8s_resources namespace ":metadata.name,:metadata.labels" "Could not remove finalizers from Verrazzano namespaces" '/k8s-app:verrazzano.io|verrazzano-system/ {print $1}' '{"metadata":{"finalizers":null}}' \
    || return $? # return on pipefail

  log "Deleting Verrazzano namespaces"
  delete_k8s_resources namespace ":metadata.name,:metadata.labels" "Could not delete Verrazzano namespaces" '/k8s-app:verrazzano.io|verrazzano-system/ {print $1}' \
    || return $? # return on pipefail
}

# This makes an attempt to uninstall OAM, ignoring errors so that this can work
# in the case where there is a partial installation
function uninstall_oam {

  log "Uninstall OAM"
  helm delete oam --namespace oam-system crossplane-master/oam-kubernetes-runtime

  log "Delete OAM roles"
  kubectl delete clusterrole oam-kubernetes-runtime-oam:system:aggregate-to-controller
  kubectl delete clusterrolebinding oam-kubernetes-runtime-oam:system:aggregate-to-controller
  kubectl delete clusterrolebinding cluster-admin-binding-oam
  kubectl delete ScopeDefinition healthscopes.core.oam.dev
  kubectl delete TraitDefinition manualscalertraits.core.oam.dev
  kubectl delete WorkloadDefinition containerizedworkloads.core.oam.dev

  log "Delete oam-system namespace"
  kubectl delete namespace oam-system

}

action "Uninstalling WebLogic Kubernetes operator " uninstall_weblogic_operator || exit 1
action "Deleting Verrazzano Components" delete_verrazzano || exit 1
action "Uninstalling OAM runtime" uninstall_oam || exit 1

