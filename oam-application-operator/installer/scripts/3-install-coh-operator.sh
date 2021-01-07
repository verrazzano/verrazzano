#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

. $SCRIPT_DIR/common.sh
TMP_DIR=$(mktemp -d)

function install_coh_operator {
  log "Add the Coherence helm repository"
  helm repo add coherence https://oracle.github.io/coherence-operator/charts
  if [ $? -ne 0 ]; then
    error "Failed to add the Coherence helm repository."
    return 1
  fi

  log "Update the helm repository"
  helm repo update
  if [ $? -ne 0 ]; then
    error "Failed to update the helm repository."
    return 1
  fi

  log "Install the Coherence Kubernetes operator"
  helm install --namespace verrazzano-system coherence coherence/coherence-operator --version 3.1.1 --wait
  if [ $? -ne 0 ]; then
    error "Failed to install the Coherence Kubernetes operator."
    return 1
  fi
}

action "Installing Coherence Kubernetes operator" install_coh_operator || fail "Failed to install Coherence Kubernetes operator."
