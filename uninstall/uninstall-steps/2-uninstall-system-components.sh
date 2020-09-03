#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install

. $INSTALL_DIR/common.sh

set -o pipefail

function delete_external_dns() {
  log "Deleting external-dns"
  helm ls -A \
    | awk '/external-dns/ {print $1}' \
    | xargs helm uninstall -n cert-manager \
    || return $? # return on pipefail

  # delete clusterrole and clusterrolebinding
  log "Deleting ClusterRoles and ClusterRoleBindings for external-dns"
  kubectl delete clusterrole external-dns --ignore-not-found=true || return $?
  kubectl delete clusterrolebinding external-dns --ignore-not-found=true || return $?
}

function delete_nginx() {
  # uninstall ingress-nginx
  log "Deleting ingress-nginx"
  helm ls -A \
    | awk '/ingress-controller/ {print $1}' \
    | xargs helm uninstall -n ingress-nginx \
    || return $? # return on pipefail

  # delete the nginx clusterrole and clusterrolebinding
  log "Deleting ClusterRoles and ClusterRoleBindings for ingress-nginx"
  kubectl delete clusterrole ingress-controller-nginx-ingress --ignore-not-found=true || return $?
  kubectl delete clusterrolebinding ingress-controller-nginx-ingress --ignore-not-found=true || return $?

  # delete ingress-nginx namespace
  log "Deleting ingress-nginx namespace"
  kubectl get namespaces --no-headers -o custom-columns=":metadata.name" \
    | awk '/ingress-nginx/ {print $1}' \
    | xargs kubectl patch namespace -p '{"metadata":{"finalizers":null}}' --type=merge  \
    || return $? # return on pipefail

  kubectl delete namespace ingress-nginx --ignore-not-found=true || return $?
}

function delete_cert_manager() {
  # uninstall cert manager deployment
  log "Deleting cert-manager"
  helm ls -A \
    | awk '/cert-manager/ {print $1}' \
    | xargs helm uninstall -n cert-manager \
    || return $? # return on pipefail

  # delete the custom resource definition for cert manager
  log "deleting the custom resource definition for cert manager"
  kubectl delete -f "https://raw.githubusercontent.com/jetstack/cert-manager/release-${CERT_MANAGER_RELEASE}/deploy/manifests/00-crds.yaml" --ignore-not-found=true || return $?

  # delete cert manager config map
  log "Deleting config map for cert manager"
  kubectl delete configmap cert-manager-controller -n kube-system --ignore-not-found=true || return $?

  log "Deleting cert manager namespace"
  # delete namespace finalizers
  kubectl get namespaces --no-headers -o custom-columns=":metadata.name" \
    | awk '/cert-manager/ {print $1}' \
    | xargs kubectl patch namespace -p '{"metadata":{"finalizers":null}}' --type=merge \
    || return $? # return on pipefail

  # delete namespace
  kubectl delete namespace cert-manager --ignore-not-found=true || return $?
}

function delete_rancher() {
  # Deleting rancher components
  log "Deleting rancher"
  helm ls -A \
    | awk '/rancher/ {print $1}' \
    | xargs helm uninstall -n cattle-system \
    || return $? # return on pipefail

  log "Deleting CRDs from rancher"

  local crd_content=$(kubectl get crds --no-headers -o custom-columns=":metadata.name,:spec.group" | awk '/coreos.com|cattle.io/')

  while [ "$crd_content" ]
  do
      # remove finalizers from crds
    kubectl get crds --no-headers -o custom-columns=":metadata.name,:spec.group" \
      | awk '/coreos.com|cattle.io/ {print $1}' \
      | xargs kubectl patch crd -p '{"metadata":{"finalizers":null}}' --type=merge \
      || return $? # return on pipefail

    # delete crds
    kubectl get crds --no-headers -o custom-columns=":metadata.name,:spec.group" \
      | awk '/coreos.com|cattle.io/ {print $1}' \
      | xargs kubectl delete crd  \
      || return $? &# return on pipefail
    sleep 30
    kill $! || true
    crd_content=$(kubectl get crds --no-headers -o custom-columns=":metadata.name,:spec.group" | awk '/coreos.com|cattle.io/')
  done

  # delete clusterrolebindings deployed by rancher
  log "Deleting ClusterRoleBindings"
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | awk '/cattle.io|app:rancher/ {print $1}' \
    | xargs kubectl delete clusterrolebinding \
    || return $? # return on pipefail

  # delete clusterroles
  log "Deleting ClusterRoles"
  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name,:metadata.labels" \
    | awk '/cattle.io/ {print $1}' \
    | xargs kubectl delete clusterrole \
    || return $? # return on pipefail

  # delete rolebinding
  log "Deleting RoleBindings"
  local default_names=("default" "kube-node-lease" "kube-public" "kube-system")
  for namespace in "${default_names[@]}"
  do
    kubectl get rolebinding --no-headers -o custom-columns=":metadata.name" -n "${namespace}"\
      | awk '/clusterrolebinding-/' \
      | xargs kubectl delete rolebinding -n "${namespace}" \
      || return $? # return on pipefail
  done

  # delete configmap in kube-system
  kubectl delete configmap cattle-controllers -n kube-system  --ignore-not-found=true || return $?

  log "Deleting cattle namespaces"
  # delete namespace finalizers
  kubectl get namespaces --no-headers -o custom-columns=":metadata.name" \
    | awk '/cattle-|local|p-|user-/ {print $1}' \
    | xargs kubectl patch namespace -p '{"metadata":{"finalizers":null}}' --type=merge \
    || return $? # return on pipefail

  # delete cattle namespaces
  kubectl get namespaces --no-headers -o custom-columns=":metadata.name" \
    | awk '/cattle-|local|p-|user-/ {print $1}' \
    | xargs kubectl delete namespaces \
    || return $? # return on pipefail

  # delete annotations left in kube-system secrets
  log "Delete Rancher Secret Annotations"
  for namespace in "${default_names[@]}"
  do
    kubectl get secret -n "${namespace}" --no-headers -o custom-columns=":metadata.name,:metadata.annotations" \
      | awk '/field.cattle.io\/projectId:/ {print $1}' \
      | xargs -I resource kubectl annotate secret resource -n "${namespace}" field.cattle.io/projectId- \
      || return $? # return on pipefail
  done
}

action "Deleting Rancher Components" delete_rancher || exit 1
action "Deleting External DNS Components" delete_external_dns || exit 1
action "Deleting Nginx Components" delete_nginx || exit 1
action "Deleting Cert Manager Components" delete_cert_manager || exit 1
