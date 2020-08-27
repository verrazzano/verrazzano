#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install

. $INSTALL_DIR/common.sh

set -o pipefail

function delete_bindings {
  binding_crd=$(kubectl get crd | grep "verrazzanobinding" || true)
  if [ -z "$binding_crd" ] ; then
    return
  fi
  kubectl get VerrazzanoBindings --no-headers -o custom-columns=":metadata.name" \
    | xargs kubectl delete VerrazzanoBindings \
    || return $? # return on pipefail
}

function delete_models {
  model_crd=$(kubectl get crd | grep "verrazzanomodel" || true)
  if [ -z "$binding_crd" ] ; then
    return
  fi
  kubectl get VerrazzanoModels --no-headers -o custom-columns=":metadata.name" \
    | xargs kubectl delete VerrazzanoModels \
    || return $? # return on pipefail
}

action "Deleting Verrazzano Bindings" delete_bindings || exit 1
action "Deleting Verrazzano Models" delete_models || exit 1