#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

CALICO_DIR=$(cd $(dirname "$0"); pwd -P)

CALICO_VERSION=$(grep 'calico-version=' ${SCRIPT_DIR}/../../.third-party-test-versions | sed 's/calico-version=//g')

download_calico() {
  mkdir -p ${CALICO_DIR}/calico/${CALICO_VERSION}
  curl -LJo ${CALICO_DIR}/calico/"${CALICO_VERSION}".tgz https://github.com/projectcalico/calico/releases/download/v"${CALICO_VERSION}"/release-v"${CALICO_VERSION}".tgz
  cd ${CALICO_DIR}/calico
  tar xzvf "${CALICO_VERSION}".tgz --strip-components=1 -C ${CALICO_DIR}/calico/${CALICO_VERSION}
  rm ${CALICO_VERSION}.tgz
  export CALICO_HOME=${CALICO_DIR}/calico
}

# Install Calico using the release bundle under CALICO_HOME. When the environment variable CALICO_HOME is set, the script
# expects the directory CALICO_VERSION inside it. When the environment variable is not set, the script downloads the
# bundle for version CALICO_VERSION from the Calico release location.
#
if [ -z "$CALICO_HOME" ]; then
  echo "CALICO_HOME is not set, downloading Calico release bundle."
  download_calico
fi

# Download the release bundle, if $CALICO_HOME/${CALICO_VERSION} doesn't exist
if [ ! -d "${CALICO_HOME}/${CALICO_VERSION}" ]; then
  echo "CALICO_HOME doesn't exist, downloading the Calico release bundle."
  download_calico
fi
