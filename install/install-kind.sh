#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

set -eu

export CLUSTER_TYPE=KIND

. $SCRIPT_DIR/common.sh

status "Creating cluster..."
$SCRIPT_DIR/0-create-kind-cluster.sh
status "Installing Istio..."
$SCRIPT_DIR/1-install-istio.sh
status "Installing system components..."
$SCRIPT_DIR/2a-install-system-components-magicdns.sh
status "Installing Verrazzano..."
$SCRIPT_DIR/3-install-verrazzano.sh
status "Installing Keycloak..."
$SCRIPT_DIR/4-install-keycloak.sh

function wait_for_env_ready() {
  kubectl -n verrazzano-system wait --for=condition=ready pods -l app=verrazzano-operator --timeout 2m
  kubectl -n verrazzano-system wait --for=condition=ready pods -l name=verrazzano-validation --timeout 2m
}
action "Wait for environment to be ready" wait_for_env_ready || fail "Environment not ready"

status ""
status "Installation of cluster ${CLUSTER_TYPE} completed"
