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

function delete_mysql() {
  # delete helm installation of MySQL
  log "Deleting MySQL"
  helm ls -A \
    | awk '/mysql/ {print $1}' \
    | xargsr helm delete -n keycloak \
    || err_exit $? "Could not delete MySQL from helm" # return on pipefail
}

function delete_keycloak() {
  # delete helm installation of Keycloak
  log "Deleting Keycloak"
  helm ls -A \
    | awk '/keycloak/ {print $1}' \
    | xargsr helm delete -n keycloak \
    || err_exit $? "Could not delete keycloak from helm" # return on pipefail

  # delete keycloak namespace
  log "Deleting keycloak namespace finalizers"
  kubectl get namespace --no-headers -o custom-columns=":metadata.name" \
    | awk '/keycloak/ {print $1}' \
    | xargsr kubectl patch namespace -p '{"metadata":{"finalizers":null}}' --type=merge \
    || err_exit $? "Could not remove finalizers from namespace keycloak" # return on pipefail

  log "Deleting Keycloak namespace"
  kubectl delete namespace keycloak --ignore-not-found=true || err_exit $? "Could not delete namespace keycloak"
}

function delete_resources() {
  log "Deleting ClusterRoles and ClusterRoleBindings"
  # deleting clusterrolebindings
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | awk '/cattle-admin|proxy-role-binding-kubernetes-master/' \
    | xargsr kubectl delete clusterrolebinding \
    || err_exit $? "Could not delete ClusterRoleBindings from Keycloak" # return on pipefail

  # deleting clusterroles
  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name" \
    | awk '/cattle-admin|local-cluster|proxy-clusterrole-kubeapiserver/' \
    | xargsr kubectl delete clusterrole \
    || err_exit $? "Could not delete ClusterRoles from Keycloak" # return on pipefail
}

action "Deleting MySQL Components" delete_mysql || exit 1
action "Deleting Keycloak Components" delete_keycloak || exit 1
action "Deleting Leftover Resources" delete_resources || exit 1