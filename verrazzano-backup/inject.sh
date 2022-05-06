#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

OPENSEARCH_BINARY_PATH="/usr/share/opensearch/data/verrazzano-bin"
VZ_BINARY="verrazzano-backup"

function log () {
  echo $(date -u) $1
}

function check_command () {
  if [ $? != 0 ] ; then
    log "Command execution failed."
    exit 1
  fi
}

function copy_opensearch () {
  log "Copy file '${VZ_BINARY}' to '$1'"
  cp -f ${VZ_BINARY} $1
  check_command
}

STR=$HOSTNAME
case $STR in
  "vmi-system-es-master-0")
    log "Creating directory  ${OPENSEARCH_BINARY_PATH}"
    mkdir -p ${OPENSEARCH_BINARY_PATH}
    check_command
    copy_opensearch ${OPENSEARCH_BINARY_PATH}
esac


