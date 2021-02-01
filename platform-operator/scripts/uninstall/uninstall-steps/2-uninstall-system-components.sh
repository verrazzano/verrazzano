#!/bin/bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install
UNINSTALL_DIR=$SCRIPT_DIR/..

. $INSTALL_DIR/common.sh
. $INSTALL_DIR/config.sh
. $UNINSTALL_DIR/uninstall-utils.sh

set -o pipefail

function delete_external_dns() {
  log "Deleting external-dns"
  helm ls -A \
    | awk '/external-dns/ {print $1}' \
    | xargsr helm uninstall -n cert-manager \
    || err_return $? "Could not delete external-dns from helm" || return $? # return on pipefail

  # delete clusterrole and clusterrolebinding
  log "Deleting ClusterRoles and ClusterRoleBindings for external-dns"
  kubectl delete clusterrole external-dns --ignore-not-found=true || err_return $? "Could not delete ClusterRole external-dns" || return $?
  kubectl delete clusterrolebinding external-dns --ignore-not-found=true || err_return $? "Could not delete ClusterRoleBinding external-dns" || return $?
}

function delete_nginx() {
  # uninstall ingress-nginx
  log "Deleting ingress-nginx"
  helm ls -A \
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

function delete_cert_manager() {
  # uninstall cert manager deployment
  log "Deleting cert-manager"
  helm ls -A \
    | awk '/cert-manager/ {print $1}' \
    | xargsr helm uninstall -n cert-manager \
    || err_return $? "Could not delete cert-manager from helm" || return $? # return on pipefail

  # delete the custom resource definition for cert manager
  log "deleting the custom resource definition for cert manager"
  kubectl delete -f "${MANIFESTS_DIR}/cert-manager/00-crds.yaml" --ignore-not-found=true \
    || err_return $? "Could not delete CustomResourceDefinition from cert-manager" || return $?

  # delete cert manager config map
  log "Deleting config map for cert manager"
  kubectl delete configmap cert-manager-controller -n kube-system --ignore-not-found=true || err_return $? "Could not delete ConfigMap from cert-manager-controller" || return $?

  log "Deleting cert-manager namespace finalizers"
  # delete namespace finalizers
  patch_k8s_resources namespaces ":metadata.name" "Could not remove finalizers from namespace cert-manager" '/cert-manager/ {print $1}' '{"metadata":{"finalizers":null}}' \
    || return $? # return on pipefail

  # delete namespace
  log "Deleting cert-manager namespace"
  kubectl delete namespace cert-manager --ignore-not-found=true || err_return $? "Could not delete namespace cert-manager" || return $?
}

function delete_rancher() {
  # Deleting rancher components
  log "Deleting rancher"
  helm ls -A \
    | awk '/rancher/ {print $1}' \
    | xargsr helm uninstall -n cattle-system \
    || err_return $? "Could not delete rancher from helm" || return $? # return on pipefail

  log "Deleting CRDs from rancher"

  local crd_content=$(kubectl get crds --no-headers -o custom-columns=":metadata.name,:spec.group" | awk '/coreos.com|cattle.io/')

  while [ "$crd_content" ]
  do
    # remove finalizers from crds
    patch_k8s_resources crds ":metadata.name,:spec.group" "Could not remove finalizers from CustomResourceDefinitions in Rancher" '/coreos.com|cattle.io/ {print $1}' '{"metadata":{"finalizers":null}}' \
      || return $? # return on pipefail

    # delete crds
    # This process is backgrounded in order to timeout due to finalizers hanging
    delete_k8s_resources crds ":metadata.name,:spec.group" "Could not delete CustomResourceDefinitions from Rancher" '/coreos.com|cattle.io/ {print $1}' \
      || return $? &# return on pipefail
    sleep 30
    kill $! || true
    crd_content=$(kubectl get crds --no-headers -o custom-columns=":metadata.name,:spec.group" | awk '/coreos.com|cattle.io/')
  done

  # delete clusterrolebindings deployed by rancher
  log "Deleting ClusterRoleBindings"
  delete_k8s_resources clusterrolebinding ":metadata.name,:metadata.labels" "Could not delete ClusterRoleBindings from Rancher" '/cattle.io|app:rancher/ {print $1}' \
    || return $? # return on pipefail

  # delete clusterroles
  log "Deleting ClusterRoles"
  delete_k8s_resources clusterrole ":metadata.name,:metadata.labels" "Could not delete ClusterRoles from Rancher" '/cattle.io/ {print $1}' \
    || return $? # return on pipefail

  # delete rolebinding
  log "Deleting RoleBindings"
  local default_names=("default" "kube-node-lease" "kube-public" "kube-system")
  for namespace in "${default_names[@]}"
  do
    delete_k8s_resources rolebinding ":metadata.name" "Could not delete RoleBindings from Rancher in namespace ${namespace}" '/clusterrolebinding-/' "${namespace}" \
      || return $? # return on pipefail
  done

  # delete configmap in kube-system
  log "Deleting ConfigMap"
  kubectl delete configmap cattle-controllers -n kube-system  --ignore-not-found=true || err_return $? "Could not delete ConfigMap from Rancher in namespace kube-system" || return $?

  log "Deleting cattle namespaces"
  # delete namespace finalizers
  patch_k8s_resources namespaces ":metadata.name" "Could not remove finalizers from namespaces in Rancher" '/cattle-|local|p-|user-/ {print $1}' '{"metadata":{"finalizers":null}}' \
    || return $? # return on pipefail

  # delete cattle namespaces
  log "Delete rancher namespace"
  delete_k8s_resources namespaces ":metadata.name" "Could not delete namespaces from Rancher" '/cattle-|local|p-|user-/ {print $1}' \
    || return $? # return on pipefail

  # delete annotations left in kube-system secrets
  log "Delete Rancher Secret Annotations"
  for namespace in "${default_names[@]}"
  do
    kubectl get secret -n "${namespace}" --no-headers -o custom-columns=":metadata.name,:metadata.annotations" \
      | awk '/field.cattle.io\/projectId:/ {print $1}' \
      | xargsr -I resource kubectl annotate secret resource -n "${namespace}" field.cattle.io/projectId- \
      || err_return $? "Could not delete Annotations from Rancher" || return $? # return on pipefail
  done
}

action "Deleting Rancher Components" delete_rancher || exit 1
action "Deleting External DNS Components" delete_external_dns || exit 1
action "Deleting NGINX Components" delete_nginx || exit 1
action "Deleting Cert Manager Components" delete_cert_manager || exit 1
