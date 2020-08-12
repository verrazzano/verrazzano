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
  helm delete ingress_controller -n ingress-nginx

  # delete the nginx clusterrole and clusterrolebinding
  if [ "$(kubectl get clusterole ingress-controller-nginx-ingress)" ] ; then
    kubectl delete clusterrole ingress-controller-nginx-ingress
  fi

  if [ "$(kubectl get clusterolebinding ingress-controller-nginx-ingress)" ] ; then
    kubectl delete clusterrolebinding ingress-controller-nginx-ingress
  fi

  # delete ingress-nginx namespace
  if [ "$(kubectl get namespace ingress-nginx)" ] ; then
    kubectl delete namespace ingress-nginx
  fi
}

action "Deleting Nginx Components" delete_nginx