#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../install

set -ueo pipefail

if [ ! -d "${SCRIPT_DIR}/.verrazzano" ] ; then
  mkdir -p ${SCRIPT_DIR}/.verrazzano
fi

export CLUSTER_TYPE=OKE

. $INSTALL_DIR/common.sh

section "Uninstalling Istio..."
$SCRIPT_DIR/1-uninstall-istio.sh
section "Uninstalling system components..."
$SCRIPT_DIR/2-uninstall-system-components-magicdns.sh
section "Uninstalling Verrazzano..."
$SCRIPT_DIR/3-uninstall-verrazzano.sh
section "Uninstalling Keycloak..."
$SCRIPT_DIR/4-uninstall-keycloak.sh

status ""
section "Uninstallation of environment ${CLUSTER_TYPE} complete."