#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname $0); pwd -P)
. $SCRIPT_DIR/common.sh

set -ue

if [[ -z "${KIND_CLUSTER_NAME}" ]]; then
  echo "KIND_CLUSTER_NAME environment variable must be set to KIND cluster name."
  exit
fi

function loadImage()
{
  image=$1
  if docker exec ${KIND_CLUSTER_NAME}-control-plane crictl images  -o json | jq '.images[].repoTags[]' | grep -q "$image";
  then
    echo  "$image is loaded"
  else
    echo  "loading $image"
    docker pull "$image"
    kind load docker-image --name $KIND_CLUSTER_NAME "$image"
  fi
}

for image in $(cat ${SCRIPT_DIR}/config/images); do
  loadImage "$image"
done
