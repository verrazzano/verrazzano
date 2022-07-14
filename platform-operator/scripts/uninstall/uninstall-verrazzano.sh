#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
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

section "Uninstalling Verrazzano..."
$SCRIPT_DIR/uninstall-steps/3-uninstall-verrazzano.sh || exit 1
section "Uninstalling system components..."
$SCRIPT_DIR/uninstall-steps/2-uninstall-system-components.sh || exit 1

section "Uninstallation of Verrazzano complete."
