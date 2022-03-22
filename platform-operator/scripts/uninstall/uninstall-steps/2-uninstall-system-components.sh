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

function delete_cert_manager() {
  # uninstall cert manager deployment
  log "Deleting cert-manager"
  helm ls -n cert-manager \
    | awk '/cert-manager/ {print $1}' \
    | xargsr helm uninstall -n cert-manager \
    || err_return $? "Could not delete cert-manager from helm" || return $? # return on pipefail

  # delete the custom resource definition for cert manager
  log "Deleting the custom resource definition for cert manager"
  kubectl delete -f "${MANIFESTS_DIR}/cert-manager/cert-manager.crds.yaml" --ignore-not-found=true \
    || err_return $? "Could not delete CustomResourceDefinition from cert-manager" || return $?

  # delete cert manager config map
  log "Deleting config map for cert manager"
  kubectl delete configmap cert-manager-controller -n kube-system --ignore-not-found=true || err_return $? "Could not delete ConfigMap from cert-manager-controller" || return $?

  log "Deleting cert-manager namespace finalizers"
  # delete namespace finalizers
  patch_k8s_resources namespaces ":metadata.name" "Could not remove finalizers from namespace cert-manager" '/cert-manager/ {print $1}' '{"metadata":{"finalizers":null}}' \
    || return $? # return on pipefail

  # delete cainjector config map
  log "Deleting cainjector leader election configmap"
  kubectl delete configmap cert-manager-cainjector-leader-election -n kube-system --ignore-not-found=true || err_return $? "Could not delete ConfigMap from kube-system" || return $?

  # delete namespace
  log "Deleting cert-manager namespace"
  kubectl delete namespace cert-manager --ignore-not-found=true || err_return $? "Could not delete namespace cert-manager" || return $?
}

function cleanup_rancher_local_cluster() {
  if kubectl get cluster local > /dev/null 2>&1 ; then
    # Occasionally we have problems deleting 'local' Rancher cluster object nicely, and it gets stuck and puts any
    # re-installed cluster in a bad state.  So here we force the delete of the object, wait for it, and then
    # patch out any remaining finalizers and check one more time for delete success.
    log "Found 'local' cluster object still present, removing..."
    kubectl delete --wait=false clusters.management.cattle.io local || true
    kubectl wait --for=delete -n kube-system clusters.management.cattle.io/local --timeout=2m || true
    # Patch any dangling finalizers
    kubectl patch clusters.management.cattle.io local -p '{"metadata":{"finalizers":null}}' --type=merge || true
    kubectl wait --for=delete -n kube-system clusters.management.cattle.io/local --timeout=1m || true
    if kubectl get cluster local > /dev/null 2>&1 ; then
      log "Unable to delete Rancher 'local' cluster object"
    else
      log "Rancher 'local' cluster deleted successfully"
    fi
  fi
}

function delete_rancher() {
  local rancher_exists=$(kubectl get namespace cattle-system --ignore-not-found)
  if [ -z "$rancher_exists" ] ; then
    return 0
  fi

  # Clean up the local rancher cluster object if necessary
  cleanup_rancher_local_cluster

  # Deleting rancher components
  log "Deleting rancher"
  helm ls -n fleet-system | awk '/fleet/ {print $1}' | xargsr helm uninstall -n fleet-system \
    || err_return $? "Could not delete fleet-system charts from helm" || return $? # return on pipefail
    helm ls -n fleet-system | awk '/fleet/ {print $1}' | xargsr helm -n fleet-system uninstall fleet-crd \
    || err_return $? "Could not delete fleet-system charts from helm" || return $? # return on pipefail
  helm ls -n rancher-operator-system | awk '/rancher/ {print $1}' | xargsr helm uninstall -n rancher-operator-system \
    || err_return $? "Could not delete rancher-operator-system charts from helm" || return $? # return on pipefail
  helm ls -n cattle-system | awk '/rancher/ {print $1}' | xargsr helm uninstall -n cattle-system \
    || err_return $? "Could not delete cattle-system from helm" || return $? # return on pipefail

  log "Delete the additional CA secret for Rancher"
  kubectl -n cattle-system delete secret tls-ca-additional 2>&1 > /dev/null || true
  kubectl -n cattle-system delete secret tls-ca --ignore-not-found=true

  log "Deleting CRs from rancher"
  kubectl api-resources --api-group=management.cattle.io --verbs=delete -o name \
    | xargsr -n 1 kubectl get --all-namespaces --ignore-not-found -o custom-columns=":kind,:metadata.name,:metadata.namespace" \
    | awk '{res="";if ($1 != "") res=tolower($1)".management.cattle.io "tolower($2); if ($3 != "<none>" && res != "") res=res" -n "$3; if (res != "") cmd="kubectl patch "res" -p \x027{\"metadata\":{\"finalizers\":null}}\x027 --type=merge;kubectl delete "res; print cmd}' \
    | sh \
    || err_return $? "Could not delete rancher CRs" || return $? # return on pipefail

  log "Deleting CRDs from rancher"

  local crd_content=$(kubectl get crds --no-headers -o custom-columns=":metadata.name,:spec.group" | awk '/coreos.com|cattle.io/')

  while [ "$crd_content" ]
  do
    # remove finalizers from crds
    # Ignore patch failures and attempt to delete the resources anyway.
    patch_k8s_resources crds ":metadata.name,:spec.group" "Could not remove finalizers from CustomResourceDefinitions in Rancher" '/coreos.com|cattle.io/ {print $1}' '{"metadata":{"finalizers":null}}' \
      || true

    # delete crds
    # This process is backgrounded in order to timeout due to finalizers hanging
    delete_k8s_resources crds ":metadata.name,:spec.group" "Could not delete CustomResourceDefinitions from Rancher" '/coreos.com|management.cattle.io|cattle.io|fleet/ {print $1}' \
      || return $? &# return on pipefail
    sleep 30
    kill $! || true
    crd_content=$(kubectl get crds --no-headers -o custom-columns=":metadata.name,:spec.group" | awk '/coreos.com|cattle.io/')
  done

  # delete clusterrolebindings deployed by rancher
  log "Deleting ClusterRoleBindings"
  delete_k8s_resources clusterrolebinding ":metadata.name,:metadata.labels" "Could not delete ClusterRoleBindings from Rancher" '/cattle.io|app:rancher|fleetworkspace-|fleet-|gitjob/ {print $1}' \
    || return $? # return on pipefail

  # delete clusterroles
  log "Deleting ClusterRoles"
  delete_k8s_resources clusterrole ":metadata.name,:metadata.labels" "Could not delete ClusterRoles from Rancher" '/cattle.io|app:rancher|fleetworkspace-|fleet-|gitjob/ {print $1}' \
    || return $? # return on pipefail

  # delete rolebinding
  log "Deleting RoleBindings"
  local default_names=("default" "kube-node-lease" "kube-public" "kube-system")
  for namespace in "${default_names[@]}"
  do
    delete_k8s_resources rolebinding ":metadata.name" "Could not delete RoleBindings from Rancher in namespace ${namespace}" '/clusterrolebinding-/' "${namespace}" \
      || return $? # return on pipefail
    delete_k8s_resources rolebinding ":metadata.name" "Could not delete RoleBindings from Rancher in namespace ${namespace}" '/^rb-/' "${namespace}" \
      || return $? # return on pipefail
  done

  # delete configmap in kube-system
  log "Deleting ConfigMap"
  kubectl delete configmap cattle-controllers -n kube-system  --ignore-not-found=true || err_return $? "Could not delete ConfigMap from Rancher in namespace kube-system" || return $?
  kubectl delete configmap rancher-controller-lock -n kube-system --ignore-not-found=true || err_return $? "Could not delete ConfigMap rancher-controller-lock in namespace kube-system" || return $?

  log "Removing Rancher namespace finalizers"
  # delete namespace finalizers
  patch_k8s_resources namespaces ":metadata.name" "Could not remove finalizers from namespaces in Rancher" '/cattle-|local|p-|user-|fleet|rancher/ {print $1}' '{"metadata":{"finalizers":null}}' \
    || return $? # return on pipefail

  log "Delete the Rancher service accounts"
  if kubectl get serviceaccount -n cattle-system rancher > /dev/null 2>&1 ; then
    if ! kubectl delete serviceaccount -n cattle-system rancher ; then
      error "Failed to delete the service account rancher in namespace cattle-system."
    fi
  fi

  if kubectl get serviceaccount -n cattle-system rancher-webhook > /dev/null 2>&1 ; then
    if ! kubectl delete serviceaccount -n cattle-system rancher-webhook ; then
      error "Failed to delete the service account rancher-webhook in namespace cattle-system."
    fi
  fi

  if kubectl get serviceaccount -n cattle-system default > /dev/null 2>&1 ; then
    if ! kubectl delete serviceaccount -n cattle-system default ; then
      error "Failed to delete the service account default in namespace cattle-system."
    fi
  fi

  # delete cattle namespaces
  log "Delete rancher namespace"
  delete_k8s_resources namespaces ":metadata.name" "Could not delete namespaces from Rancher" '/cattle-|local|p-|user-|fleet|rancher/ {print $1}' \
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

  # Remove the Rancher namespace finalizers; do it here since Rancher is not guaranteed to have been installed by VZ
  # (it can now be opted-out by the user)
  log "Removing Rancher Namespace Finalizers"
  kubectl get namespaces --no-headers -o custom-columns=":metadata.name,:metadata.finalizers" \
    | awk '/controller.cattle.io/ {print $1}' \
    | xargsr kubectl patch namespace -p '{"metadata":{"finalizers":null}}' --type=merge \
    || err_return $? "Could not remove Rancher finalizers from all namespaces" || return $? # return on pipefail

  log "Removing Rancher MutatingWebhookConfiguration"
  kubectl delete mutatingwebhookconfigurations.admissionregistration.k8s.io rancher.cattle.io --ignore-not-found \
    || err_return $? "Failed to delete MutatingWebhookConfiguration rancher.cattle.io"
}

action "Deleting Rancher Components" delete_rancher || exit 1
action "Deleting NGINX Components" delete_nginx || exit 1
action "Deleting Cert Manager Components" delete_cert_manager || exit 1
