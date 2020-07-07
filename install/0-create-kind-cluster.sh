#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle Corporation and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname $0); pwd -P)

export CLUSTER_TYPE=KIND

set -u

. $SCRIPT_DIR/common.sh

KIND_IMAGE="kindest/node:v1.16.9@sha256:7175872357bc85847ec4b1aba46ed1d12fa054c83ac7a8a11f5c268957fd5765"

command -v kind >/dev/null 2>&1 || {
    consoleerr "kind is required but cannot be found on the path";
    consoleerr "Please install kind and try again: https://kind.sigs.k8s.io/docs/user/quick-start#installation"
    exit 1;
}

${SCRIPT_DIR}/5-delete-kind-cluster.sh >&5 6>&5

action "Creating kind cluster ${KIND_CLUSTER_NAME}..." \
    kind create cluster \
        --wait 30s \
        --image ${KIND_IMAGE} \
        --name ${KIND_CLUSTER_NAME} \
        --config ${SCRIPT_DIR}/config/kind-config.yaml \
        --kubeconfig ${KUBECONFIG}

action "Loading Docker images into kind..." $SCRIPT_DIR/load-images.sh
