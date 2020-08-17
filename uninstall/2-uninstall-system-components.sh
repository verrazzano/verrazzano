#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../install

. $INSTALL_DIR/common.sh

function delete_external_dns() {
  log "Deleting external-dns"
  helm delete external-dns -n cert-manager || 2>/dev/null

  # delete clusterrole and clusterrolebinding
  log "Deleting ClusterRoles and ClusterRoleBindings for external-dns"
  if [ "$(kubectl get clusterrole external-dns)" ] ; then
    kubectl delete clusterrole external-dns
  fi

  if [ "$(kubectl get clusterrolebinding external-dns)" ] ; then
    kubectl delete clusterrolebinding external-dns
  fi
}

function delete_nginx() {
  # uninstall ingress-nginx
  log "Deleting ingress-nginx"
  helm delete ingress-controller -n ingress-nginx || 2>/dev/null

  # delete the nginx clusterrole and clusterrolebinding
  log "Deleting ClusterRoles and ClusterRoleBindings for ingress-nginx"
  if [ "$(kubectl get clusterrole ingress-controller-nginx-ingress)" ] ; then
    kubectl delete clusterrole ingress-controller-nginx-ingress
  fi

  if [ "$(kubectl get clusterrolebinding ingress-controller-nginx-ingress)" ] ; then
    kubectl delete clusterrolebinding ingress-controller-nginx-ingress
  fi

  # delete ingress-nginx namespace
  log "Deleting ingress-nginx namespace"
  if [ "$(kubectl get namespace ingress-nginx)" ] ; then
    kubectl delete namespace ingress-nginx
  fi
}

function delete_cert_manager() {
  # uninstall cert manager deployment
  log "Deleting cert-manager"
  helm delete cert-manager -n cert-manager || 2>/dev/null

  # delete the custom resource definition for cert manager
  log "deleting the custom resource definition for cert manager"
  kubectl delete -f https://raw.githubusercontent.com/jetstack/cert-manager/release-0.13/deploy/manifests/00-crds.yaml

  # delete cert manager config map
  log "Deleting config map for cert manager"
  if [ "$(kubectl get configmap cert-manager-controller -n kube-system)" ] ; then
    kubectl delete configmap cert-manager-controller -n kube-system
  fi

  # delete namespace
  log "Deleting cert manager namespace"
  if [ "$(kubectl get namespace cert-manager)" ] ; then
    kubectl delete namespace cert-manager
  fi
}

function delete_rancher() {
  # Deleting rancher components
  log "Deleting rancher"
  helm delete rancher -n cattle-system || 2>/dev/null

  log "Deleting CRDs from rancher"
  while [ "$(kubectl get crds --no-headers -o custom-columns=":metadata.name" | grep -E 'coreos.com|.cattle.io')" ]
  do
    # remove finalizers from crds
    kubectl get crds --no-headers -o custom-columns=":metadata.name" \
      | grep -E 'coreos.com|.cattle.io' \
      | xargs kubectl patch crd -p '{"metadata":{"finalizers":null}}' --type=merge

    # delete crds (include timeout for undiscovered finalizer problem)
    kubectl get crds --no-headers -o custom-columns=":metadata.name" \
      | grep -E 'coreos.com|.cattle.io' \
      | xargs kubectl delete crd &
    sleep 30
    kill $! || 2>/dev/null
  done

  # delete clusterrolebindings deployed by rancher
  log "Deleting ClusterRoleBindings"
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'clusterrolebinding-|cattle-globalrolebinding-|globaladmin-user|grb-u|rancher' \
    | xargs kubectl delete clusterrolebinding

  # delete clusterroles
  log "Deleting ClusterRoles"
  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'p-|project-|user-|cluster-owner|create-ns' \
    | xargs kubectl delete clusterrole

  # delete rolebinding
  log "Deleting RoleBindings"
  local default_names=("default" "kube-node-lease" "kube-public" "kube-system")
  for namespace in "${default_names[@]}"
  do
    kubectl get rolebinding --no-headers -o custom-columns=":metadata.name" -n "${namespace}"\
      | grep 'clusterrolebinding-' \
      | xargs kubectl delete rolebinding -n "${namespace}"
  done

  # delete configmap in kube-system
  if [ "$(kubectl get configmap cattle-controllers -n kube-system)" ] ; then
    kubectl delete configmap cattle-controllers -n kube-system
  fi

  log "Deleting cattle namespaces"
  # delete namespace finalizers
  kubectl get namespaces --no-headers -o custom-columns=":metadata.name" \
    | grep 'cattle' \
    | xargs kubectl patch namespace -p '{"metadata":{"finalizers":null}}' --type=merge

  # delete cattle namespaces
  kubectl get namespaces --no-headers -o custom-columns=":metadata.name" | grep -E 'cattle|local' | xargs kubectl delete namespaces
}

action "Deleting External DNS Components" delete_external_dns
action "Deleting Nginx Components" delete_nginx
action "Deleting Cert Manager Components" delete_cert_manager
action "Deleting Rancher Components" delete_rancher
