#!/bin/bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ ! -f "$1" ]; then
  echo "You must specify the BOM file as input"
  exit 1
fi
if [ -z "$2" ]; then
  echo "You must specify the output image list file name"
  exit 1
fi
BOM_FILE=$1
IMG_LIST_FILE=$2
REPOS="${3:-verrazzano}"

if [ -f "$IMG_LIST_FILE" ]; then
  echo "Output file $IMG_LIST_FILE already exists, please specify a new filename"
  exit 1
else
  touch $IMG_LIST_FILE
fi

source $SCRIPT_DIR/bom_utils.sh

function list_images() {
  local components=($(list_components))
  local global_registry=$(get_bom_global_registry)
  for component in "${components[@]}"; do
    local sub_components=$(list_subcomponent_names ${component})
    for subcomponent in ${sub_components}; do
      local override_registry=$(resolve_subcomponent_registry_from_bom ${component} ${subcomponent})
      local from_repository=$(get_subcomponent_repo ${component} ${subcomponent})
      if [[ "$REPOS" == *"$from_repository"* ]]; then
        local image_names=$(list_subcomponent_images ${component} ${subcomponent})
        for base_image in ${image_names}; do
          local from_image
          if [ "$override_registry" == "null" ]; then
              from_image=${from_repository}/${base_image}
          else
              from_image=${override_registry}/${from_repository}/${base_image}
          fi
          local existing=$(cat ${IMG_LIST_FILE} | grep ${from_image})
          if [ -z "$existing" ]; then
            echo "${from_image}" >> ${IMG_LIST_FILE}
          fi
        done
      fi
    done
  done
}

list_images $1 $2
