#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../install

. $INSTALL_DIR/common.sh

if [ "$(kubectl get vb -A)" ] || [ "$(kubectl get vm -A)" ] ; then
  error "Please delete all Verrazzano Models and Verrazzano Bindings before continuing the uninstall"
  exit 1
fi

section "Uninstalling Istio..."
$SCRIPT_DIR/uninstall-steps/1-uninstall-istio.sh
section "Uninstalling system components..."
$SCRIPT_DIR/uninstall-steps/2-uninstall-system-components.sh
section "Uninstalling Verrazzano..."
$SCRIPT_DIR/uninstall-steps/3-uninstall-verrazzano.sh
section "Uninstalling Keycloak..."
$SCRIPT_DIR/uninstall-steps/4-uninstall-keycloak.sh

section "Uninstallation of Verrazzano complete."