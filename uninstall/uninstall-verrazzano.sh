#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../install

. $INSTALL_DIR/common.sh

function usage() {
  error
    error "usage: $0 [-f] [-h]"
    error " -f    Force the uninstall and suppress warnings"
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
    read -r -t 30 resp
    case $resp in
      [Yy]* ) break;;
      [Nn]* ) exit;;
      * ) status 'Please answer yes or no'
    esac
  done

  if [ "$(kubectl get vb -A)" ] || [ "$(kubectl get vm -A)" ] ; then
    echo "$(tput bold)The following applications will be deleted upon uninstall:$(tput sgr0)" >&4
    echo "Verrazzano Models:" >&4
    echo "$(kubectl get vm -A --no-headers -o custom-columns=":metadata.name")" >&4
    echo "Verrazzano Bindings:" >&4
    echo "$(kubectl get vb -A --no-headers -o custom-columns=":metadata.name")" >&4

    while true
    do
      echo -n "$(tput bold)Are you sure you want to delete these applications? [y/n]:$(tput sgr0)" >&4
      read -r -t 30 resp
      case $resp in
        [Yy]* ) break;;
        [Nn]* ) exit;;
        * ) status 'Please answer yes or no'
      esac
    done
  fi
fi

section "Uninstalling Verrazzano Applications"
$SCRIPT_DIR/uninstall-steps/0-uninstall-applications.sh
section "Uninstalling Istio..."
$SCRIPT_DIR/uninstall-steps/1-uninstall-istio.sh
section "Uninstalling system components..."
$SCRIPT_DIR/uninstall-steps/2-uninstall-system-components.sh
section "Uninstalling Verrazzano..."
$SCRIPT_DIR/uninstall-steps/3-uninstall-verrazzano.sh
section "Uninstalling Keycloak..."
$SCRIPT_DIR/uninstall-steps/4-uninstall-keycloak.sh

section "Uninstallation of Verrazzano complete."