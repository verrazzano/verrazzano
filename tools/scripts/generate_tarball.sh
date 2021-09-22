#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ ! -f "$1" ]; then
  echo "You must specify the images list BOM file as input"
  exit 1
fi
BOM_FILE=$1

if [ ! -d "$2" ]; then
  echo "Please specify temp directory"
  exit 1
fi

if [ -f "$3" ]; then
  echo "Output file already exists, please specify a new filename"
  exit 1
fi

if [ -d "$3" ]; then
  echo "Please specify a new filename not a directory"
  exit 1
fi

source $SCRIPT_DIR/bom_utils.sh

# Main driver for pulling/saving images based on the Verrazzano bill of materials (BOM)
function pull_and_save_images() {
  # Loop through registry components
  echo "Using image registry ${BOM_FILE}"
  local components=($(list_components))
  local global_registry=$(get_registry)
  for component in "${components[@]}"; do
    local sub_components=$(list_subcomponent_names ${component})
    for subcomponent in ${sub_components}; do
      local override_registry=$(get_subcomponent_registry ${component} ${subcomponent})
      local from_repository=$(get_subcomponent_repo ${component} ${subcomponent})
      local subcomponent_path=""
      if [ "$override_registry" == "null" ]; then
        subcomponent_path="$global_registry"
      else
        subcomponent_path="$override_registry"
      fi
      if [ ! -z "$from_repository" ] && [ "$from_repository" != "null" ]; then
        subcomponent_path="${subcomponent_path}/${from_repository}"
      fi
      local image_names=$(list_subcomponent_images ${component} ${subcomponent})
      for base_image in ${image_names}; do
        local from_image=${subcomponent_path}/${base_image}
        echo "Processing:  ${from_image}"
        local tarname=$(echo "$from_image.tar" | sed -e 's;/;_;g' -e 's/:/-/g')
        docker pull $from_image
        docker save -o $2/${tarname} ${from_image}
      done
    done
  done
  tar -czf $3 -C $2 .
}

pull_and_save_images $1 $2 $3
