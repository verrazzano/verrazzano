#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
PROJ_DIR=$(cd $(dirname "$0"); cd ../..; pwd -P)
BUILD_DEPLOY=${PROJ_DIR}/build/deploy

VERRAZZANO_NS=verrazzano-system

. $SCRIPT_DIR/common.sh

# This makes an attempt to uninstall OAM, ignoring errors so that this can work
# in the case where there is a partial installation

if [ -z "${VERRAZZANO_APP_OP_IMAGE:-}" ] ; then
    export VERRAZZANO_APP_OP_IMAGE=$(cat $PROJ_DIR/deploy/application-operator.txt)
fi
if [ -z "${VERRAZZANO_APP_OP_IMAGE:-}" ] ; then
    error "Failed to determine Verrazzano OAM operator image."
    return 1
fi

function uninstall {
  log "Uninstalling Verrazzano application operator"
  kubectl delete -f ${BUILD_DEPLOY}/verrazzano.yaml

  log "Uninstalling Verrazzano application operator OAM extensions"
  kubectl delete -f ${PROJ_DIR}/deploy

  log "Uninstalling Verrazzano application operator CRD extensions"
  kubectl delete -f ${PROJ_DIR}/config/crd/bases

}

action "Uninstalling Verrazzano application operator" uninstall || fail "Failed to uninstall the Verrazzano application operator."
