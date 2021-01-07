#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
PROJ_DIR=$(cd $(dirname "$0"); cd ../..; pwd -P)

. $SCRIPT_DIR/common.sh

# This makes an attempt to uninstall the Coherence Kubernetes operator, ignoring errors so that this can work
# in the case where there is a partial installation

function uninstall {
  log "Uninstalling Coherence Kubernetes operator"
  helm delete coherence --namespace verrazzano-system

  log "Deleting the Coherence CRD"
  kubectl delete crd coherence.coherence.oracle.com
  return 0
}

action "Uninstalling Coherence Kubernetes operator " uninstall
