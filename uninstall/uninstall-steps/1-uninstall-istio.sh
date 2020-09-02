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
  local istio_crd_res=("$(kubectl get crd --no-headers -o custom-columns=":metadata.name" \
    | grep 'istio.io' || true)")

  printf "%s\n" "${istio_crd_res[@]}" \
    | awk '{print $1}' \
    | xargs kubectl delete crd \
    || return $? # return on pipefail

  # delete istio api services
  log "Deleting Istio API Services"
  local istio_api_res=("$(kubectl get apiservice --no-headers -o custom-columns=":metadata.name" \
    | grep 'istio.io' || true)")

  printf "%s\n" "${istio_api_res[@]}" \
    | awk '{print $1}' \
    | xargs kubectl delete apiservice \
    || return $? # return on pipefail

  # delete istio cluster role bindings
  log "Deleting Istio Cluster Role Bindings"
  local istio_crb_res=("$(kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'istio-system|istio-multi' || true)")

  printf "%s\n" "${istio_crb_res[@]}" \
    | awk '{print $1}' \
    | xargs kubectl delete clusterrolebinding \
    || return $? # return on pipefail

  # delete istio cluster roles
  log "Deleting Istio Cluster Roles"
  local istio_crb_res=("$(kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'istio-system|istio-reader|istiocoredns' || true)")

  printf "%s\n" "${istio_crb_res[@]}" \
    | awk '{print $1}' \
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
  local secret_res=("$(kubectl get secrets -n kube-system --no-headers -o custom-columns=":metadata.name,:metadata.annotations" \
  | grep "istio." || true)")

  printf "%s\n" "${secret_res[@]}" \
  | awk '{print $1}' \
  | xargs kubectl delete secret -n kube-system \
  || return $? # return on pipefail
}

function delete_istio_namepsace() {
  log "Deleting istio-system namespace"
  kubectl delete namespace istio-system --ignore-not-found=true || return $?
}

action "Deleting Istio Components" uninstall_istio || exit 1
action "Deleting Istio Secrets" delete_secrets || exit 1
action "Deleting Istio Namespace" delete_istio_namepsace || exit 1