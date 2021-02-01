#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install
UNINSTALL_DIR=$SCRIPT_DIR/..

. $INSTALL_DIR/common.sh
. $INSTALL_DIR/config.sh
. $UNINSTALL_DIR/uninstall-utils.sh

set -o pipefail

function delete_mysql() {
  # delete helm installation of MySQL
  log "Deleting MySQL"
  helm ls -A \
    | awk '/mysql/ {print $1}' \
    | xargsr helm delete -n keycloak \
    || err_return $? "Could not delete MySQL from helm" || return $? # return on pipefail
}

function delete_keycloak() {
  # delete helm installation of Keycloak
  log "Deleting Keycloak"
  helm ls -A \
    | awk '/keycloak/ {print $1}' \
    | xargsr helm delete -n keycloak \
    || err_return $? "Could not delete keycloak from helm" || return $? # return on pipefail

  # delete keycloak namespace
  log "Deleting keycloak namespace finalizers"
  patch_k8s_resources namespace ":metadata.name" "Could not remove finalizers from namespace keycloak" '/keycloak/ {print $1}' '{"metadata":{"finalizers":null}}' \
    || return $? # return on pipefail

  log "Deleting Keycloak namespace"
  kubectl delete namespace keycloak --ignore-not-found=true || err_return $? "Could not delete namespace keycloak" || return $?
}

function delete_resources() {
  log "Deleting ClusterRoles and ClusterRoleBindings"
  # deleting clusterrolebindings
  delete_k8s_resources clusterrolebinding ":metadata.name" "Could not delete ClusterRoleBindings from Keycloak" '/cattle-admin|proxy-role-binding-kubernetes-master/' \
    || return $? # return on pipefail

  # deleting clusterroles
  delete_k8s_resources clusterrole ":metadata.name" "Could not delete ClusterRoles from Keycloak" '/cattle-admin|local-cluster|proxy-clusterrole-kubeapiserver/' \
    || return $? # return on pipefail
}

action "Deleting MySQL Components" delete_mysql || exit 1
action "Deleting Keycloak Components" delete_keycloak || exit 1
action "Deleting Leftover Resources" delete_resources || exit 1