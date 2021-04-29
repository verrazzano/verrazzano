#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

CALICO_DIR=$(cd $(dirname "$0"); pwd -P)

# Install Calico using the release bundle under CALICO_HOME. When the environment variable CALICO_HOME is set, the script
# expects the release-v3.18.1.tgz inside it. When the environment variable is not set, the script downloads the release
# bundle from the Calico release location.
#
function install_calico() {
    if [ -z "$CALICO_HOME" ]; then
      mkdir -p ${CALICO_DIR}/calico
      export CALICO_HOME=${CALICO_DIR}/calico
      curl -LJo ${CALICO_DIR}/calico/release-v3.18.1.tgz https://github.com/projectcalico/calico/releases/download/v3.18.1/release-v3.18.1.tgz
    fi

    # The $CALICO_HOME should have release-v3.18.1.tgz
    if [ -f "${CALICO_HOME}/release-v3.18.1.tgz" ]; then
      cd ${CALICO_HOME}
      tar -xzvf release-v3.18.1.tgz --strip-components=1
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
    else
      echo "File ${CALICO_HOME}/release-v3.18.1.tgz does not exist, Calico installation failed."
      return 1
    fi
    return 0
}

case "$1" in
    "") ;;
    install_calico) "$@"; exit;;
    *) echo "Unknown function: $1()"; exit 2;;
esac