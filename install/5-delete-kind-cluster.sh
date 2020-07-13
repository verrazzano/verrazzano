#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname $0); pwd -P)
export CLUSTER_TYPE=KIND

. $SCRIPT_DIR/common.sh

set -u

command -v kind >/dev/null 2>&1 || {
    consoleerr "kind is required but cannot be found on the path";
    consoleerr "Please install kind and try again: https://kind.sigs.k8s.io/docs/user/quick-start#installation"
    exit 1;
}

action "Deleting kind cluster ${KIND_CLUSTER_NAME}..." kind delete cluster --name="${KIND_CLUSTER_NAME}" --kubeconfig "${KUBECONFIG}"
