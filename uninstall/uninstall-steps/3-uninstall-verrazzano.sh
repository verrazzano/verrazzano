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
    | xargs helm uninstall -n verrazzano-system \
    || err_exit $? "Could not delete verrazzano from helm" # return on pipefail

  # delete verrazzano-managed-cluster-local secret
  log "Deleting Verrazzano secrets"
  kubectl delete secret verrazzano-managed-cluster-local --ignore-not-found=true || err_exit $? "Could not delete secrets from Verrazzano"

  # delete crds
  log "Deleting Verrazzano crd finalizers"
  kubectl get crds --no-headers -o custom-columns=":metadata.name" \
    | awk '/verrazzano.io/' \
    | xargs kubectl patch crd -p '{"metadata":{"finalizers":null}}' --type=merge \
    || err_exit $? "Could not remove finalizers from CustomResourceDefinitions in Verrazzano" # return on pipefail

  log "Deleting Verrazzano crds"
  kubectl get crds --no-headers -o custom-columns=":metadata.name" \
    | awk '/verrazzano.io/' \
    | xargs kubectl delete crd \
    || err_exit $? "Could not delete CustomResourceDefinitions from Verrazzano" # return on pipefail

  # deleting certificatesigningrequests
  log "Deleting CertificateSigningRequests"
  kubectl get csr --no-headers -o custom-columns=":metadata.name" \
    | awk '/csr-/' \
    | xargs kubectl delete csr \
    || err_exit $? "Could not delete CertificateSigningRequests from Verrazzano" # return on pipefail

  log "Deleting ClusterRoles and ClusterRoleBindings"
  # deleting clusterrolebindings
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | awk '/verrazzano/ {print $1}' \
    | xargs kubectl delete clusterrolebinding \
    || err_exit $? "Could not delete ClusterRoleBindings from Verrazzano" # return on pipefail

  # deleting clusterroles
  log "Deleting ClusterRoles"
  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | awk '/verrazzano/ {print $1}' \
    | xargs kubectl delete clusterrole \
    || err_exit $? "Could not delete ClusterRoles from Verrazzano" # return on pipefail

  # deleting namespaces
  log "Deleting Verrazzano namespace finalizers"
  # delete namespace finalizers
  kubectl get namespace --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | awk '/k8s-app:verrazzano.io|verrazzano-system/ {print $1}' \
    | xargs kubectl patch namespace -p '{"metadata":{"finalizers":null}}' --type=merge \
    || err_exit $? "Could not remove finalizers from Verrazzano namespaces" # return on pipefail

  log "Deleting Verrazzano namespaces"
  kubectl get namespace --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | awk '/k8s-app:verrazzano.io|verrazzano-system/ {print $1}' \
    | xargs kubectl delete namespace \
    || err_exit $? "Could not delete Verrazzano namespaces" # return on pipefail
}

action "Deleting Verrazzano Components" delete_verrazzano || exit 1