#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# $1 The docker image to pull (required)
# $2 Retry limit (optional defaults to 5)
# $3 Sleep time (optional defaults to 5 seconds)
function docker_pull_retry () {
  if [ -z "$1" ]; then
    echo "Image to pull needs to be specified"
    return 1
  fi
  local attempts=0
  local retry_limit=${2:-"5"}
  local sleep_time=${3:-"5"}
  until [ ${attempts} -ge ${retry_limit} ]
  do
    docker pull $1 && break
    let attempts=attempts+1
    if [ ${attempts} -ge ${retry_limit} ]; then
      echo "docker pull failed after ${ATTEMPTS} retry attempts"
      return 1
    fi
    sleep ${sleep_time}
  done
  return 0
}
