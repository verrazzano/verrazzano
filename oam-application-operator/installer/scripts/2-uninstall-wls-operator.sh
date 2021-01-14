#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
PROJ_DIR=$(cd $(dirname "$0"); cd ../..; pwd -P)

. $SCRIPT_DIR/common.sh

# This makes an attempt to uninstall the WebLogic Kubernetes operator, ignoring errors so that this can work
# in the case where there is a partial installation

function uninstall {
  log "Uninstalling WebLogic Kubernetes operator"
  helm delete weblogic-operator --namespace verrazzano-system

  log "Deleting weblogic-operator serviceaccount"
  kubectl delete serviceaccount -n verrazzano-system weblogic-operator-sa

  return 0
}

action "Uninstalling WebLogic Kubernetes operator " uninstall
