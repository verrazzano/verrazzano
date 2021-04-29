#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

CALICO_DIR=$(cd $(dirname "$0"); pwd -P)
CALICO_VERSION=${1:-"3.18.1"}

# Install Calico using the release bundle under CALICO_HOME. When the environment variable CALICO_HOME is set, the script
# expects the release-v3.18.1.tgz inside it. When the environment variable is not set, the script downloads the release
# bundle from the Calico release location.
#
if [ -z "$CALICO_HOME" ]; then
  echo "CALICO_HOME is not set, downloading Calico release bundle ..."
  mkdir -p ${CALICO_DIR}/calico
  export CALICO_HOME=${CALICO_DIR}/calico
  curl -LJo ${CALICO_DIR}/calico/release-v"${CALICO_VERSION}".tgz https://github.com/projectcalico/calico/releases/download/v"${CALICO_VERSION}"/release-v"${CALICO_VERSION}".tgz
fi

# Download the release bundle, if $CALICO_HOME doesn't contain the requested/required version
if [[ ! -f "${CALICO_HOME}/release-v${CALICO_VERSION}.tgz" ]]; then
  echo "CALICO_HOME doesn't contain the release bundle for version ${CALICO_VERSION}, downloading it ..."
  curl -LJo ${CALICO_DIR}/calico/release-v"${CALICO_VERSION}".tgz https://github.com/projectcalico/calico/releases/download/v"${CALICO_VERSION}"/release-v"${CALICO_VERSION}".tgz
fi

echo "Extract Calico release bundle, load the images and apply calico.yaml"
cd ${CALICO_HOME}
tar -xzvf release-v${CALICO_VERSION}.tgz --strip-components=1
cd ${CALICO_HOME}/images
kind load image-archive calico-cni.tar
kind load image-archive calico-dikastes.tar
kind load image-archive calico-flannel-migration-controller.tar
kind load image-archive calico-kube-controllers.tar
kind load image-archive calico-node.tar
kind load image-archive calico-pod2daemon-flexvol.tar
kind load image-archive calico-typha.tar
cd ${CALICO_HOME}/k8s-manifests
kubectl apply -f calico.yaml
