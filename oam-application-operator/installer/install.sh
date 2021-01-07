#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
PROJ_DIR=$(cd $(dirname "$0"); cd ..; pwd -P)

. $SCRIPT_DIR/scripts/common.sh

if [ -z "${VERRAZZANO_APP_OP_IMAGE:-}" -a -f "$PROJ_DIR/deploy/application-operator.txt" ] ; then
    export VERRAZZANO_APP_OP_IMAGE=$(cat $PROJ_DIR/deploy/application-operator.txt)
fi
log "Application operator image is ${VERRAZZANO_APP_OP_IMAGE}"

set -e
"$SCRIPT_DIR"/scripts/1-install-oam-runtime.sh
"$SCRIPT_DIR"/scripts/2-install-wls-operator.sh
"$SCRIPT_DIR"/scripts/3-install-coh-operator.sh
"$SCRIPT_DIR"/scripts/4-install-vz-oam.sh
