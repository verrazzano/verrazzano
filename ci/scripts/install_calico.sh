#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
CLUSTER_NAME=${1:-"kind"}
CALICO_VERSION=${2:-"3.18.1"}

$SCRIPT_DIR/download_calico.sh ${CALICO_VERSION}

echo "Load the docker image from Calico archives at ${CALICO_HOME}/${CALICO_VERSION}/images."
cd ${CALICO_HOME}/${CALICO_VERSION}/images
for image_archive in *.tar; do
    echo "Loading image archive $image_archive ..."
    kind load image-archive "$image_archive" --name "${CLUSTER_NAME}"
done

echo "Apply ${CALICO_HOME}/${CALICO_VERSION}/k8s-manifests/calico.yaml."
cd ${CALICO_HOME}/${CALICO_VERSION}/k8s-manifests
kubectl apply -f calico.yaml