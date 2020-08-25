#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install

. $INSTALL_DIR/common.sh

function delete_verrazzano() {
  # delete helm installation of Verrazzano
  log "Deleting Verrazzano"
  helm delete verrazzano -n verrazzano-system || 2>/dev/null

  # delete verrazzano-managed-cluster-local secret
  log "Deleting Verrazzano secrets"
  if [ "$(kubectl get secret verrazzano-managed-cluster-local)" ] ; then
    kubectl delete secret verrazzano-managed-cluster-local
  fi

  # delete crds
  log "Deleting Verrazzano crds"
  kubectl get crds --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'verrazzano.io' \
    | xargs kubectl patch crd -p '{"metadata":{"finalizers":null}}' --type=merge

  kubectl get crds --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'verrazzano.io' \
    | xargs kubectl delete crd

  # deleting certificatesigningrequests
  log "Deleting CertificateSigningRequests"
  kubectl get csr --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'csr-' \
    | xargs kubectl delete csr

  log "Deleting ClusterRoles and ClusterRoleBindings"
  # deleting clusterrolebindings
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | grep -E 'verrazzano' \
    | awk '{print $1}' \
    | xargs kubectl delete clusterrolebinding

  # deleting clusterroles
  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | grep -E 'verrazzano' \
    | awk '{print $1}' \
    | xargs kubectl delete clusterrole

  # deleting namespaces
  log "Deleting Verrazzano namespaces"
  kubectl get namespace --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | grep -E 'k8s-app:verrazzano.io|verrazzano-system' \
    | awk '{print $1}' \
    | xargs kubectl delete namespace
}

check_network
action "Deleting Verrazzano Components" delete_verrazzano || exit 1