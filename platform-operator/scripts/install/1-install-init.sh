#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/config.sh

INGRESS_TYPE=$(get_config_value ".ingress.type")

CONFIG_DIR=$SCRIPT_DIR/config
TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

function log_kube_version {
    local kubeVer=$(kubectl version -o json)
    log "------Begin Kubernetes Version Info----"
    log "$kubeVer"
    log "------End Kubernetes Version Info----"
    local servVer=$(echo $kubeVer | jq -r '.serverVersion.gitVersion')
    if [ "$servVer" == "null" ] || [ -z "$servVer" ]; then
        log "Could not retrieve Kubernetes server version"
        return 1
    fi
}

function check_helm_version {
    local helmVer=$(helm version --short | cut -d':' -f2 | tr -d " ")
    log "Helm version is $helmVer"
    local majorVer=$(echo $helmVer | cut -d'.' -f1)
    local minorVer=$(echo $helmVer | cut -d'.' -f2)
    if [ "$majorVer" != "v3" ]; then
        log "Helm major version is $majorVer, expected v3!"
        return 1
    fi
    return 0
}

function wait_for_nodes_to_exist {
    retries=0
    until kubectl get nodes | grep NAME; do
      retries=$(($retries+1))
      sleep 10
      if [ "$retries" -ge 30 ] ; then
        break
      fi
    done
    if [ "$retries" -ge 30 ] ; then
      log "Kubernetes nodes don't exist in cluster"
      return 1
    fi
}

function wait_for_istio {
  wait_for_deployment istio-system istiod
  return $?
}

set -ueo pipefail

action "Checking Kubernetes version" log_kube_version || exit 1
action "Checking Helm version" check_helm_version || (error "Helm version must be v3.x! Your Helm version is: $(helm version --short)"; exit 1)

# Wait for all cluster nodes to exist, and then to be ready
action "Waiting for all Kubernetes nodes to exist in cluster" wait_for_nodes_to_exist || exit 1

log "Kubernetes nodes exist"
action "Waiting for all Kubernetes nodes to be ready" \
    kubectl wait --for=condition=ready nodes --all || exit 1

# Label the kube-system namespace so that we can apply network policies
log "Adding label needed by network policies to kube-system namespace"
kubectl label namespace kube-system "verrazzano.io/namespace=kube-system" --overwrite
if [ $? -ne 0 ]; then
  echo "Failed to label kube-system namespace"
  exit 1
fi

# Wait for istio control plane to be ready
action "Waiting for istio control plane to be ready" wait_for_istio || exit 1
