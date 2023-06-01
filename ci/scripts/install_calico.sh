#!/bin/bash
#
# Copyright (c) 2021, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ -z $1 ]; then
    echo "Cluster name is required to be supplied"
    exit 1
fi
CLUSTER_NAME=$1

CALICO_VERSION=$(grep 'calico-version=' ${SCRIPT_DIR}/../../.third-party-test-versions | sed 's/calico-version=//g')

. $SCRIPT_DIR/download_calico.sh

echo "Load the docker image from Calico archives at ${CALICO_HOME}/${CALICO_VERSION}/images."
cd ${CALICO_HOME}/${CALICO_VERSION}/images
for image_archive in *.tar; do
    echo "Loading image archive $image_archive ..."
    kind load image-archive "$image_archive" --name "${CLUSTER_NAME}"
done

echo "Apply ${CALICO_HOME}/${CALICO_VERSION}/manifests/calico.yaml."
cd ${CALICO_HOME}/${CALICO_VERSION}/manifests
kubectl apply -f calico.yaml
