#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

CALICO_DIR=$(cd $(dirname "$0"); pwd -P)
CLUSTER_NAME=${1:-"kind"}
CALICO_VERSION=${2:-"3.18.1"}

download_calico() {
  mkdir -p ${CALICO_DIR}/calico/${CALICO_VERSION}
  curl -LJo ${CALICO_DIR}/calico/"${CALICO_VERSION}".tgz https://github.com/projectcalico/calico/releases/download/v"${CALICO_VERSION}"/release-v"${CALICO_VERSION}".tgz
  cd ${CALICO_DIR}/calico
  tar xzvf "${CALICO_VERSION}".tgz --strip-components=1 -C ${CALICO_DIR}/calico/${CALICO_VERSION}
  rm ${CALICO_VERSION}.tgz
  export CALICO_HOME=${CALICO_DIR}/calico/${CALICO_VERSION}
}

# Install Calico using the release bundle under CALICO_HOME. When the environment variable CALICO_HOME is set, the script
# expects the directory CALICO_VERSION inside it. When the environment variable is not set, the script downloads the
# bundle for version CALICO_VERSION from the Calico release location.
#
if [ -z "$CALICO_HOME" ]; then
  echo "CALICO_HOME is not set, downloading Calico release bundle."
  download_calico
fi

# Download the release bundle, if $CALICO_HOME doesn't exist
if [ ! -d "${CALICO_HOME}" ]; then
  echo "CALICO_HOME doesn't exist, downloading the calico release bundle."
  download_calico
fi

echo "Load the docker image from Calico archives at ${CALICO_HOME}/images."
cd ${CALICO_HOME}/images
kind load image-archive calico-cni.tar --name ${CLUSTER_NAME}
kind load image-archive calico-dikastes.tar --name ${CLUSTER_NAME}
kind load image-archive calico-flannel-migration-controller.tar --name ${CLUSTER_NAME}
kind load image-archive calico-kube-controllers.tar --name ${CLUSTER_NAME}
kind load image-archive calico-node.tar --name ${CLUSTER_NAME}
kind load image-archive calico-pod2daemon-flexvol.tar --name ${CLUSTER_NAME}
kind load image-archive calico-typha.tar --name ${CLUSTER_NAME}

echo "Apply ${CALICO_HOME}/k8s-manifests/calico.yaml."
cd ${CALICO_HOME}/k8s-manifests
kubectl apply -f calico.yaml