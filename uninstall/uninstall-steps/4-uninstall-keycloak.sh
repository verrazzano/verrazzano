#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install

. $INSTALL_DIR/common.sh

set -o pipefail

function delete_mysql() {
  # delete helm installation of MySQL
  log "Deleting MySQL"
  helm ls -A \
    | grep "mysql" || true  \
    | awk '{print $1}' \
    | xargs helm delete -n keycloak \
    || return $? # return on pipefail
}

function delete_keycloak() {
  # delete helm installation of Keycloak
  log "Deleting Keycloak"
  helm ls -A \
    | awk '{print $1}' \
    | grep "keycloak" || true  \
    | xargs helm delete -n keycloak \
    || return $? # return on pipefail

  # delete keycloak namespace
  log "Deleting Keycloak namespace"
  kubectl delete namespace keycloak --ignore-not-found=true || return $?
}

function delete_resources() {
  log "Deleting ClusterRoles and ClusterRoleBindings"
  # deleting clusterrolebindings
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'cattle-admin|proxy-role-binding-kubernetes-master' || true  \
    | xargs kubectl delete clusterrolebinding \
    || return $? # return on pipefail

  # deleting clusterroles
  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'cattle-admin|local-cluster|proxy-clusterrole-kubeapiserver' || true  \
    | xargs kubectl delete clusterrole \
    || return $? # return on pipefail
}

function finalize() {
  # Grab all leftover Helm repos and delete resources
  log "Deleting Helm repos"
  helm repo ls || true \
    | awk 'NR>1 {print $1}' \
    | xargs -I name helm repo remove name \
    || return $? # return on pipefail

  # Removing possible reference to verrazzano in clusterroles and clusterrolebindings
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'verrazzano' || true  \
    | xargs kubectl delete clusterrolebinding \
    || return $? # return on pipefail

  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'verrazzano' || true  \
    | xargs kubectl delete clusterrole \
    || return $? # return on pipefail
}

action "Deleting MySQL Components" delete_mysql || exit 1
action "Deleting Keycloak Components" delete_keycloak || exit 1
action "Deleting Leftover Resources" delete_resources || exit 1
action "Finalizing Uninstall" finalize || exit 1