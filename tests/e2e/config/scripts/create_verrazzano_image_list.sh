#!/bin/bash

set -e
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
WORKSPACE_ROOT=${SCRIPT_DIR}/../..
IMG_LIST_FILE=$1

if [ -z "${KUBECONFIG}" ] ; then
    echo "KUBECONFIG env var must be set!"
    exit 1
fi

source ${WORKSPACE_ROOT}/verrazzano/platform-operator/scripts/install/common.sh
# get image list from cluster and persist to output file
echo "Inspecting cluster pods for verrazzano release images"
kubectl get pods --all-namespaces -o jsonpath="{..image}" |\tr -s '[[:space:]]' '\n' |\sort |\uniq | grep verrazzano | grep / | cut -d/ -f2- >> ${IMG_LIST_FILE} || exit 1

# add the acme solver (short lived container image)
echo "adding acme solver image to list"
echo $CERT_MANAGER_SOLVER_IMAGE:$CERT_MANAGER_SOLVER_TAG | grep / | cut -d/ -f2- >> ${IMG_LIST_FILE} || exit 1
