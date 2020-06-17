#!/bin/bash

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

set -eu

export CLUSTER_TYPE=KIND

. $SCRIPT_DIR/common.sh


$SCRIPT_DIR/0-create-kind-cluster.sh >&$CONSOLE_STDOUT 2>&$CONSOLE_STDERR
$SCRIPT_DIR/1-install-istio.sh >&$CONSOLE_STDOUT 2>&$CONSOLE_STDERR
$SCRIPT_DIR/2a-install-system-components-magicdns.sh >&$CONSOLE_STDOUT 2>&$CONSOLE_STDERR
$SCRIPT_DIR/3-install-verrazzano.sh >&$CONSOLE_STDOUT 2>&$CONSOLE_STDERR
$SCRIPT_DIR/4-install-keycloak.sh >&$CONSOLE_STDOUT 2>&$CONSOLE_STDERR

#
# Wait for environment to be ready
kubectl -n verrazzano-system wait --for=condition=ready pods -l app=verrazzano-operator --timeout 2m
kubectl -n verrazzano-system wait --for=condition=ready pods -l name=verrazzano-validation --timeout 2m

consoleout
consoleout "Installation of cluster ${CLUSTER_TYPE} completed"
