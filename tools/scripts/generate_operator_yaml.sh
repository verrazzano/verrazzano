#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

IMAGE_PULL_SECRETS=${IMAGE_PULL_SECRETS:-}

set -ueo pipefail

command -v helm >/dev/null 2>&1 || {
  fail "helm is required but cannot be found on the path. Aborting."
}

function check_helm_version {
    local helm_version=$(helm version --short | cut -d':' -f2 | tr -d " ")
    local major_version=$(echo $helm_version | cut -d'.' -f1)
    local minor_version=$(echo $helm_version | cut -d'.' -f2)
    if [ "$major_version" != "v3" ]; then
        echo "Helm version is $helm_version, expected v3!" >&2
        return 1
    fi
    return 0
}

check_helm_version || exit 1

DOCKER_IMAGE=${DOCKER_IMAGE:-}
if [ -z "${DOCKER_IMAGE}" ] ; then
    echo "DOCKER_IMAGE environment variable must be set"
    exit 1
fi

IMAGE_PULL_SECRET_ARG=
if [ -n "${IMAGE_PULL_SECRETS}" ] ; then
    IMAGE_PULL_SECRET_ARG="--set global.imagePullSecrets={${IMAGE_PULL_SECRETS}}"
fi

helm template \
    --include-crds \
    ${IMAGE_PULL_SECRET_ARG} \
    --set image=${DOCKER_IMAGE} \
    $SCRIPT_DIR/../../platform-operator/helm_config/charts/verrazzano-platform-operator

exit $?
