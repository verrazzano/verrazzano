#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../install

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
  kubectl get namespace --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'keycloak' \
    | xargs kubectl delete namespace
}

function delete_resources() {
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
  if [ "$(kubectl get secret ocr)" ] ; then
    kubectl delete secret ocr
  fi
}

action "Deleting MySQL Components" delete_mysql
action "Deleting Keycloak Components" delete_keycloak
action "Deleting Leftover Resources" delete_resources
action "Finalizing Uninstall" finalize