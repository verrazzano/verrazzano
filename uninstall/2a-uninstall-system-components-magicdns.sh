#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../install

. $INSTALL_DIR/common.sh

function delete_nginx() {
  # uninstall ingress-nginx
  log "Uninstalling ingress-nginx"
  helm delete ingress_controller -n ingress-nginx || 2>/dev/null

  # delete the nginx clusterrole and clusterrolebinding
  if [ "$(kubectl get clusterrole ingress-controller-nginx-ingress)" ] ; then
    kubectl delete clusterrole ingress-controller-nginx-ingress
  fi

  if [ "$(kubectl get clusterrolebinding ingress-controller-nginx-ingress)" ] ; then
    kubectl delete clusterrolebinding ingress-controller-nginx-ingress
  fi

  # delete ingress-nginx namespace
  if [ "$(kubectl get namespace ingress-nginx)" ] ; then
    kubectl delete namespace ingress-nginx
  fi
}

function delete_cert_manager() {
    # uninstall cert manager deployment
  log "Uninstalling cert-manager"
  helm delete cert-manager -n cert-manager || 2>/dev/null

  # delete the custom resource definition for cert manager
  kubectl delete -f https://raw.githubusercontent.com/jetstack/cert-manager/release-0.13/deploy/manifests/00-crds.yaml

  # delete namespace
  kubectl delete namespace cert-manager
}

action "Deleting Nginx Components" delete_nginx
action "Delete Cert Manager Components" delete_cert_manager