#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

set -ueo pipefail

if [ -z "${VERRAZZANO_KUBECONFIG:-}" ] ; then
  echo "Environment variable VERRAZZANO_KUBECONFIG must be set an point to a valid kubeconfig"
  exit 1
fi

if [ ! -d "${SCRIPT_DIR}/.verrazzano" ] ; then
  mkdir -p ${SCRIPT_DIR}/.verrazzano
fi

export INSTALL_CONFIG_FILE="$SCRIPT_DIR/config/config_defaults.json"

. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/config.sh

section "Installing Istio..."
$SCRIPT_DIR/1-install-istio.sh
section "Installing system components..."
$SCRIPT_DIR/2a-install-system-components-magicdns.sh
section "Installing Verrazzano..."
$SCRIPT_DIR/3-install-verrazzano.sh
section "Installing Keycloak..."
$SCRIPT_DIR/4-install-keycloak.sh

function wait_for_env_ready() {
  kubectl -n verrazzano-system wait --for=condition=ready pods -l app=verrazzano-operator --timeout 2m
  kubectl -n verrazzano-system wait --for=condition=ready pods -l name=verrazzano-validation --timeout 2m
}
action "Wait for environment to be ready" wait_for_env_ready || fail "Environment not ready"

status ""
section "Installation of environment $(get_config_value ".environmentName") complete."

