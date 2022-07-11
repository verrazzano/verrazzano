#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install
UNINSTALL_DIR=$SCRIPT_DIR/..

. $INSTALL_DIR/common.sh
. $INSTALL_DIR/config.sh
. $UNINSTALL_DIR/uninstall-utils.sh

set -o pipefail

function delete_nginx() {
  # uninstall ingress-nginx
  log "Deleting ingress-nginx"
  helm ls -n ingress-nginx \
    | awk '/ingress-controller/ {print $1}' \
    | xargsr helm uninstall -n ingress-nginx \
    || err_return $? "Could not delete ingress-controller from helm" || return $? # return on pipefail

  # delete the nginx clusterrole and clusterrolebinding
  log "Deleting ClusterRoles and ClusterRoleBindings for ingress-nginx"
  kubectl delete clusterrole ingress-controller-ingress-nginx --ignore-not-found=true || err_return $? "Could not delete ClusterRole ingress-controller-ingress-nginx" || return $?
  kubectl delete clusterrolebinding ingress-controller-ingress-nginx --ignore-not-found=true || err_return $? "Could not delete ClusterRoleBinding ingress-controller-ingress-nginx" || return $?

  # delete ingress-nginx namespace
  log "Deleting ingress-nginx namespace finalizers"
  patch_k8s_resources namespaces ":metadata.name" "Could not remove finalizer from namespace ingress-nginx" '/ingress-nginx/ {print $1}' '{"metadata":{"finalizers":null}}' \
    || return $? # return on pipefail

  log "Deleting ingress-nginx namespace"
  kubectl delete namespace ingress-nginx --ignore-not-found=true || err_return $? "Could not delete namespace ingress-nginx" || return $?
}

action "Deleting NGINX Components" delete_nginx || exit 1
