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

# Get the global Docker registry specified in the BOM
function get_registry() {
  cat ${BOM_FILE} | jq -r '.registry'
}

# Get the list of component names in the BOM
function list_components() {
  cat ${BOM_FILE} | jq -r '.components[].name'
}

# Get the repository name for a component in the BOM
function get_component_repo() {
  local compName=$1
  cat ${BOM_FILE} | jq -r --arg comp ${compName} '.components[] | select(.name|test($comp)) | .repository'
}

# List the base image names for all subcomponents of a component in the BOM, in the form <image-name>:<tag>
function list_subcomponent_images() {
  local compName=$1
  cat ${BOM_FILE} | jq -r --arg comp ${compName} \
    '.components[] | select(.name|test($comp)) | .subcomponents[].images[] | "\(.image):\(.tag)"'
}

# Main driver for pulling/saving images based on the Verrazzano bill of materials (BOM)
function pull_and_save_images() {
  # Loop through registry components
  echo "Using image registry ${BOM_FILE}"
  local from_registry=$(get_registry)
  local components=($(list_components))
  for component in "${components[@]}"; do
    echo "Processing images for Verrazzano component ${component}"
    # Load the repository and base image names for the component
    local from_repository=$(get_component_repo $component)
    local image_names=$(list_subcomponent_images $component)
    for base_image in ${image_names}; do
      # Build up the image name and target image name, and do a pull/tag/push
      local from_image=${from_registry}/${from_repository}/${base_image}
      echo "DEBUG: ${from_image}"
      local tarname=$(echo "$from_image.tar" | sed -e 's;/;_;g' -e 's/:/-/g')
      docker pull $from_image
      docker save -o $2/${tarname} ${from_image}
    done
  done
  tar -czf ${3} -C ${2} .
}

pull_and_save_images
