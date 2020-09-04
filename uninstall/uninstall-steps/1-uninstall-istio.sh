#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install

. $INSTALL_DIR/common.sh

set -o pipefail

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

CONFIG_DIR=$INSTALL_DIR/config

function uninstall_istio() {
  # delete webhook configurations
  log "Removing Istio Webhook Configurations"
  kubectl delete MutatingWebhookConfiguration istio-sidecar-injector --ignore-not-found=true || return $?
  kubectl delete ValidatingWebhookConfiguration istio-galley --ignore-not-found=true || return $?

  # delete istio crds
  log "Deleting Istio Custom Resource Definitions"
  kubectl get crd --no-headers -o custom-columns=":metadata.name" \
    | awk '/istio.io/ {print $1}' \
    | xargs kubectl delete crd \
    || return $? # return on pipefail

  # delete istio api services
  log "Deleting Istio API Services"
  kubectl get apiservice --no-headers -o custom-columns=":metadata.name" \
    | awk '/istio.io/ {print $1}' \
    | xargs kubectl delete apiservice \
    || return $? # return on pipefail

  # delete istio cluster role bindings
  log "Deleting Istio Cluster Role Bindings"
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | awk '/istio-system|istio-multi/ {print $1}' \
    | xargs kubectl delete clusterrolebinding \
    || return $? # return on pipefail

  # delete istio cluster roles
  log "Deleting Istio Cluster Roles"
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | awk '/istio-system|istio-reader|istiocoredns/ {print $1}' \
    | xargs kubectl delete clusterrole \
    || return $? # return on pipefail
}

function delete_secrets() {
  # Delete istio.default in all namespaces
  log "Collecting istio secrets for deletion"
  kubectl delete secret istio.default --ignore-not-found=true || return $?
  kubectl delete secret istio.default -n kube-public --ignore-not-found=true || return $?
  kubectl delete secret istio.default -n kube-node-lease --ignore-not-found=true || return $?

  # delete secrets left over in kube-system
  kubectl get secrets -n kube-system --no-headers -o custom-columns=":metadata.name,:metadata.annotations" \
  | awk '/istio./ {print $1}' \
  | xargs kubectl delete secret -n kube-system \
  || return $? # return on pipefail
}

function delete_istio_namepsace() {
  kubectl get namespaces --no-headers -o custom-columns=":metadata.name" \
    | awk '/istio-system/ {print $1}' \
    | xargs kubectl patch namespace -p '{"metadata":{"finalizers":null}}' --type=merge  \
    || return $? # return on pipefail

  log "Deleting istio-system namespace"
  kubectl delete namespace istio-system --ignore-not-found=true || return $?
}

function finalize() {
  # Grab all leftover Helm repos and delete resources
  log "Deleting Helm repos"
  helm repo ls || true \
    | awk 'NR>1 {print $1}' \
    | xargs -I name helm repo remove name \
    || return $? # return on pipefail

  # Removing possible reference to verrazzano in clusterroles and clusterrolebindings
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | awk '/verrazzano/' \
    | xargs kubectl delete clusterrolebinding \
    || return $? # return on pipefail

  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name" \
    | awk '/verrazzano/' \
    | xargs kubectl delete clusterrole \
    || return $? # return on pipefail
}

action "Deleting Istio Components" uninstall_istio || exit 1
action "Deleting Istio Secrets" delete_secrets || exit 1
action "Deleting Istio Namespace" delete_istio_namepsace || exit 1
action "Finalizing Uninstall" finalize || exit 1