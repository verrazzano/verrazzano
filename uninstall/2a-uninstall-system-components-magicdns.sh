#!/bin/bash

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../install

. $INSTALL_DIR/common.sh

function delete_rancher() {
  # grab repo for Rancher
  log "Add Rancher helm repository location"
  helm repo add rancher-stable https://releases.rancher.com/server-charts/stable
}