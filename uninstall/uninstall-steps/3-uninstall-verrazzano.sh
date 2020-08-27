#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install

. $INSTALL_DIR/common.sh

set -o pipefail

function delete_verrazzano() {
  # delete helm installation of Verrazzano
  log "Deleting Verrazzano"
  local verr_res=("$(helm ls -A \
    | grep "verrazzano" || true)")

  printf "%s\n" "${verr_res[@]}" \
    | awk '{print $1}' \
    | xargs helm uninstall -n verrazzano-system \
    || return $? # return on pipefail

  # delete verrazzano-managed-cluster-local secret
  log "Deleting Verrazzano secrets"
  kubectl delete secret verrazzano-managed-cluster-local --ignore-not-found=true || return $?

  # delete crds
  log "Deleting Verrazzano crds"
  local verr_crd_fin_res=("$(kubectl get crds --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'verrazzano.io' || true)")

  printf "%s\n" "${verr_crd_fin_res[@]}" \
    | xargs kubectl patch crd -p '{"metadata":{"finalizers":null}}' --type=merge \
    || return $? # return on pipefail

  local verr_crd_rec=("$(kubectl get crds --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'verrazzano.io' || true)")

  printf "%s\n" "${verr_crd_rec[@]}" \
    | xargs kubectl delete crd \
    || return $? # return on pipefail

  # deleting certificatesigningrequests
  log "Deleting CertificateSigningRequests"
  local verr_csr_res=("$(kubectl get csr --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'csr-' || true)")

  printf "%s\n" "${verr_csr_res[@]}" \
    | xargs kubectl delete csr \
    || return $? # return on pipefail

  log "Deleting ClusterRoles and ClusterRoleBindings"
  # deleting clusterrolebindings
  local verr_crb_res=("$(kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | grep -E 'verrazzano' || true)")

  printf "%s\n" "${verr_crb_res[@]}" \
    | awk '{print $1}' \
    | xargs kubectl delete clusterrolebinding \
    || return $? # return on pipefail

  # deleting clusterroles
  local verr_cr_res=("$(kubectl get clusterrole --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | grep -E 'verrazzano' || true)")

  printf "%s\n" "${verr_cr_res[@]}" \
    | awk '{print $1}' \
    | xargs kubectl delete clusterrole \
    || return $? # return on pipefail

  # deleting namespaces
  log "Deleting Verrazzano namespaces"
  local verr_ns_res=("$(kubectl get namespace --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | grep -E 'k8s-app:verrazzano.io|verrazzano-system' || true)")

  printf "%s\n" "${verr_ns_res[@]}" \
    | awk '{print $1}' \
    | xargs kubectl delete namespace \
    || return $? # return on pipefail
}

action "Deleting Verrazzano Components" delete_verrazzano || exit 1