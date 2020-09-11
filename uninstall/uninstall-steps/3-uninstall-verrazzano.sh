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
  kubectl get crds --no-headers -o custom-columns=":metadata.name" \
    | awk '/verrazzano.io/' \
    | xargsr kubectl patch crd -p '{"metadata":{"finalizers":null}}' --type=merge \
    || err_return $? "Could not remove finalizers from CustomResourceDefinitions in Verrazzano" || return $? # return on pipefail

  log "Deleting Verrazzano crds"
  kubectl get crds --no-headers -o custom-columns=":metadata.name" \
    | awk '/verrazzano.io/' \
    | xargsr kubectl delete crd \
    || err_return $? "Could not delete CustomResourceDefinitions from Verrazzano" || return $? # return on pipefail

  # deleting certificatesigningrequests
  log "Deleting CertificateSigningRequests"
  kubectl get csr --no-headers -o custom-columns=":metadata.name" \
    | awk '/csr-/' \
    | xargsr kubectl delete csr \
    || err_return $? "Could not delete CertificateSigningRequests from Verrazzano" || return $? # return on pipefail

  log "Deleting ClusterRoles and ClusterRoleBindings"
  # deleting clusterrolebindings
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | awk '/verrazzano/ {print $1}' \
    | xargsr kubectl delete clusterrolebinding \
    || err_return $? "Could not delete ClusterRoleBindings from Verrazzano" || return $? # return on pipefail

  # deleting clusterroles
  log "Deleting ClusterRoles"
  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | awk '/verrazzano/ {print $1}' \
    | xargsr kubectl delete clusterrole \
    || err_return $? "Could not delete ClusterRoles from Verrazzano" || return $? # return on pipefail

  # deleting namespaces
  log "Deleting Verrazzano namespace finalizers"
  # delete namespace finalizers
  kubectl get namespace --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | awk '/k8s-app:verrazzano.io|verrazzano-system/ {print $1}' \
    | xargsr kubectl patch namespace -p '{"metadata":{"finalizers":null}}' --type=merge \
    || err_return $? "Could not remove finalizers from Verrazzano namespaces" || return $? # return on pipefail

  log "Deleting Verrazzano namespaces"
  kubectl get namespace --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | awk '/k8s-app:verrazzano.io|verrazzano-system/ {print $1}' \
    | xargsr kubectl delete namespace \
    || err_return $? "Could not delete Verrazzano namespaces" || return $? # return on pipefail
}

action "Deleting Verrazzano Components" delete_verrazzano || exit 1