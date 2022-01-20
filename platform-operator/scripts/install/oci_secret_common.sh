#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

if [ -z "${KUBECONFIG:-}" ] ; then
  echo "Environment variable KUBECONFIG must be set an point to a valid kube config file"
  exit 1
fi

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true' EXIT

# read a config item from a specified section of an oci config file
function read_config() {
  if [[ $# -lt 2 || ! -f $1 ]]; then
    echo "usage: iniget <file> [--list|<SECTION> [key]]"
    return 1
  fi
  local ocifile=$1

  if [ "$2" == "--list" ]; then
    for SECTION in $(cat $ocifile | grep "\[" | sed -e "s#\[##g" | sed -e "s#\]##g"); do
      echo $SECTION
    done
    return 0
  fi

  local SECTION=$2
  local key
  [ $# -eq 3 ] && key=$3

 local lines=$(awk '/\[/{prefix=$0; next} $1{print prefix $0}' $ocifile)
  for line in $lines; do
    if [[ "$line" = \[$SECTION\]* ]]; then
      local keyval=$(echo $line | sed -e "s/^\[$SECTION\]//")
      if [[ -z "$key" ]]; then
        echo $keyval
      else
        if [[ "$keyval" = $key=* ]]; then
          echo $(echo $keyval | sed -e "s/^$key=//")
        fi
      fi
    fi
  done
}

