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

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

CONFIG_DIR=$INSTALL_DIR/config

function uninstall_istio() {
  # delete webhook configurations
  log "Removing Istio Webhook Configurations"
  kubectl delete MutatingWebhookConfiguration istio-sidecar-injector --ignore-not-found=true || err_exit $? "Could not delete MutatingWebhookConfiguration from Istio"
  kubectl delete ValidatingWebhookConfiguration istio-galley --ignore-not-found=true || err_exit $? "Could not delete ValidatingWebhookConfiguration from Istio"

  # delete istio crds
  log "Deleting Istio Custom Resource Definitions"
  kubectl get crd --no-headers -o custom-columns=":metadata.name" \
    | awk '/istio.io/ {print $1}' \
    | xargsr kubectl delete crd \
    || err_exit $? "Could not delete CustomResourceDefinition from Istio" # return on pipefail

  # delete istio api services
  log "Deleting Istio API Services"
  kubectl get apiservice --no-headers -o custom-columns=":metadata.name" \
    | awk '/istio.io/ {print $1}' \
    | xargsr kubectl delete apiservice \
    || err_exit $? "Could not delete APIServices from Istio" # return on pipefail

  # delete istio cluster role bindings
  log "Deleting Istio Cluster Role Bindings"
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | awk '/istio-system|istio-multi/ {print $1}' \
    | xargsr kubectl delete clusterrolebinding \
    || err_exit $? "Could not delete ClusterRoleBindings from Istio" # return on pipefail

  # delete istio cluster roles
  log "Deleting Istio Cluster Roles"
  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name" \
    | awk '/istio-system|istio-reader|istiocoredns/ {print $1}' \
    | xargsr kubectl delete clusterrole \
    || err_exit $? "Could not delete ClusterRoles from Istio" # return on pipefail
}

function delete_secrets() {
  # Delete istio.default in all namespaces
  log "Collecting istio secrets for deletion"
  kubectl delete secret istio.default --ignore-not-found=true || err_exit $? "Could not delete secret from Istio in namespace default"
  kubectl delete secret istio.default -n kube-public --ignore-not-found=true || err_exit $? "Could not delete secret from Istio in namespace kube-public"
  kubectl delete secret istio.default -n kube-node-lease --ignore-not-found=true || err_exit $? "Could not delete secret from Istio in namespace kuce-node-lease"

  # delete secrets left over in kube-system
  kubectl get secrets -n kube-system --no-headers -o custom-columns=":metadata.name,:metadata.annotations" \
  | awk '/istio./ {print $1}' \
  | xargsr kubectl delete secret -n kube-system \
  || err_exit $? "Could not delete secrets from Istio in namespace kube-system" # return on pipefail
}

function delete_istio_namepsace() {
  log "Deleting istio-system finalizers"
  kubectl get namespaces --no-headers -o custom-columns=":metadata.name" \
    | awk '/istio-system/ {print $1}' \
    | xargsr kubectl patch namespace -p '{"metadata":{"finalizers":null}}' --type=merge  \
    || err_exit $? "Could not remove finalizers from namespace istio-system"# return on pipefail

  log "Deleting istio-system namespace"
  kubectl delete namespace istio-system --ignore-not-found=true || err_exit $? "Could not delete namespace istio-system"
}

function finalize() {
  # Removing possible reference to verrazzano in clusterroles and clusterrolebindings
  log "Removing Verrazzano ClusterRoles and ClusterRoleBindings"
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | awk '/verrazzano/' \
    | xargsr kubectl delete clusterrolebinding \
    || err_exit $? "Could not delete ClusterRoleBindings" # return on pipefail

  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name" \
    | awk '/verrazzano/' \
    | xargsr kubectl delete clusterrole \
    || err_exit $? "Could not delete ClusterRoles" # return on pipefail

  # Grab all leftover Helm repos and delete resources
  log "Deleting Helm repos"
  helm repo ls \
    | awk '/istio.io|stable|jetstack|rancher-stable|codecentric/ {print $1}' \
    | xargsr -I name helm repo remove name \
    || err_exit $? "Could not delete helm repos" # return on pipefail
}

action "Deleting Istio Components" uninstall_istio || exit 1
action "Deleting Istio Secrets" delete_secrets || exit 1
action "Deleting Istio Namespace" delete_istio_namepsace || exit 1
action "Finalizing Uninstall" finalize || exit 1