#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install

. $INSTALL_DIR/common.sh

function delete_mysql() {
  # delete helm installation of MySQL
  log "Deleting MySQL"
  helm delete mysql -n keycloak || 2>/dev/null
}

function delete_keycloak() {
  # delete helm installation of Keycloak
  log "Deleting Keycloak"
  helm delete keycloak -n keycloak || 2>/dev/null

  # delete keycloak namespace
  log "Deleting Keycloak namespace"
  if [ "$(kubectl get namespace keycloak)" ] ; then
    kubectl delete namespace keycloak
  fi
}

function delete_resources() {
  log "Deleting ClusterRoles and ClusterRoleBindings"
  # deleting clusterrolebindings
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'cattle-admin|proxy-role-binding-kubernetes-master' \
    | xargs kubectl delete clusterrolebinding

  # deleting clusterroles
  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'cattle-admin|local-cluster|proxy-clusterrole-kubeapiserver' \
    | xargs kubectl delete clusterrole
}

function finalize() {
  # Grab all leftover Helm repos and delete resources
  log "Deleting Helm repos"
  helm repo ls | awk 'NR>1 {print $1}' | xargs -I name helm repo remove name

  # Removing possible reference to verrazzano in clusterroles and clusterrolebindings
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'verrazzano' \
    | xargs kubectl delete clusterrolebinding

  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'verrazzano' \
    | xargs kubectl delete clusterrole
}

action "Deleting MySQL Components" delete_mysql
action "Deleting Keycloak Components" delete_keycloak
action "Deleting Leftover Resources" delete_resources
action "Finalizing Uninstall" finalize