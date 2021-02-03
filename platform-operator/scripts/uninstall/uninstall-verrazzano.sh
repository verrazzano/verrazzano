#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../install

. $INSTALL_DIR/common.sh
. $SCRIPT_DIR/uninstall-utils.sh

set -o pipefail

APPLICATION_RESOURCES=""

function usage() {
  error
  error "usage: $0 [-f] [-h]"
  error " -f    Force the uninstall and suppress prompts"
  error " -h    Help"
  error
  exit
}

FORCE=false
while getopts fh flag
do
  case "${flag}" in
      f) FORCE=true;;
      h) usage;;
      *) usage;;
  esac
done

if [ "$FORCE" = false ] ; then
  while true
  do
    echo -n "$(tput bold)All Verrazzano resources and applications will be permanently removed from the cluster. Do you wish to proceed? [y/n]:$(tput sgr0)" >&4
    read -r resp
    case $resp in
      [Yy]* ) break;;
      [Nn]* ) exit;;
      * ) status 'Please answer yes or no'
    esac
  done
fi

function check_applications () {
  # check to make sure crds exist and grab them
  kubectl get crd verrazzanobindings.verrazzano.io || return 0
  kubectl get crd verrazzanomodels.verrazzano.io || return 0
  bindings=$(kubectl get vb) || err_return $? "Could not retrieve VerrazzanoBindings" || return $?
  models=$(kubectl get vm) || err_return $? "Could not retrieve VerrazzanoModels" || return $?

  if [ "$bindings" ] || [ "$models" ] ; then
    APPLICATION_RESOURCES="$(tput bold)The following applications will be deleted upon uninstall:$(tput sgr0)
    Verrazzano Models:
        $(kubectl get vm --no-headers -o custom-columns=":metadata.name" || err_return $? "Could not retrieve VerrazzanoModels" || return $?)
    Verrazzano Bindings:
        $(kubectl get vb --no-headers -o custom-columns=":metadata.name" || err_return $? "Could not retrieve VerrazzanoBindings" || return $?)\n"
  fi
}

function prompt_delete_applications () {
  # check to make sure crds exist and grab them
  kubectl get crd verrazzanobindings.verrazzano.io || return 0
  kubectl get crd verrazzanomodels.verrazzano.io || return 0
  bindings=$(kubectl get vb) || err_return $? "Could not retrieve VerrazzanoBindings" || return $?
  models=$(kubectl get vm) || err_return $? "Could not retrieve VerrazzanoModels" || return $?

  if [ "$bindings" ] || [ "$models" ] ; then
      if [ "$FORCE" = false ] ; then
      while true
      do
        echo -n "$(tput bold)Are you sure you want to delete these applications? [y/n]:$(tput sgr0)" >&4
        read -r resp
        case $resp in
          [Yy]* ) break;;
          [Nn]* ) exit;;
          * ) status 'Please answer yes or no'
        esac
      done
    fi
  fi
}

action "Retrieving Verrazzano Applications" check_applications || exit 1
echo -ne "$APPLICATION_RESOURCES" >&4
prompt_delete_applications || exit 1

section "Uninstalling Verrazzano Applications"
$SCRIPT_DIR/uninstall-steps/0-uninstall-applications.sh || exit 1
section "Uninstalling Keycloak..."
$SCRIPT_DIR/uninstall-steps/4-uninstall-keycloak.sh || exit 1
section "Uninstalling Verrazzano..."
$SCRIPT_DIR/uninstall-steps/3-uninstall-verrazzano.sh || exit 1
section "Uninstalling system components..."
$SCRIPT_DIR/uninstall-steps/2-uninstall-system-components.sh || exit 1
section "Uninstalling Istio..."
$SCRIPT_DIR/uninstall-steps/1-uninstall-istio.sh || exit 1

section "Uninstallation of Verrazzano complete."
