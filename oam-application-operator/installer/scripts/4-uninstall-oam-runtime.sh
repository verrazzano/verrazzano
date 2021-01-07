#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

. $SCRIPT_DIR/common.sh

# This makes an attempt to uninstall OAM, ignoring errors so that this can work
# in the case where there is a partial installation
function uninstall_oam {

  log "Uninstall OAM"
  helm delete oam --namespace verrazzano-system 

#  log "Delete OAM roles"
   kubectl delete clusterrolebinding cluster-admin-binding-oam

}

action "Uninstalling OAM runtime" uninstall_oam
