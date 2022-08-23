#!/bin/bash
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Script utilities for interacting with the Verrazzano BOM
#

# variables
BOM_FILE=./verrazzano-bom.json

# Get the global Docker registry specified in the BOM
function get_bom_global_registry() {
  cat ${BOM_FILE} | jq -r '.registry'
}

# Get the list of component names in the BOM
function list_components() {
  cat ${BOM_FILE} | jq -r '.components[].name'
}

# List all the subcomponents for a component in the BOM
function list_subcomponent_names() {
  local compName=$1
  cat ${BOM_FILE} | jq -r --arg comp ${compName} '.components[] | select(.name == $comp) | .subcomponents[].name'
}

# Get the repository name for a subcomponent in the BOM
function get_subcomponent_repo() {
  local compName=$1
  local subcompName=$2
  cat ${BOM_FILE} | jq -r -c --arg comp ${compName} --arg subcomp ${subcompName} '.components[] | select(.name == $comp) | .subcomponents[] | select(.name == $subcomp) | .repository'
}

# Get the repository override for an image in the BOM
function get_image_repo_override() {
  local compName=$1
  local subcompName=$2
  local imageNameTag=$3
  local imageName=$(echo $imageNameTag | cut -d \: -f 1)
  local imageTag=$(echo $imageNameTag | cut -d \: -f 2)
  cat ${BOM_FILE} | jq -r -c --arg comp ${compName} --arg subcomp ${subcompName} --arg image ${imageName} --arg tag ${imageTag} \
    '.components[] | select(.name == $comp) | .subcomponents[] | select(.name == $subcomp) | .images[] | select(.image == $image and .tag == $tag) | .repository'
}

# Resolves the repo for an image in the BOM; precedence is the image definition, followed by the subcomponent definition
function resolve_image_repo_from_bom() {
  local compName=$1
  local subcompName=$2
  local imageNameTag=$3

  local resolvedRepo=$(get_subcomponent_repo $compName $subcompName)
  local imageRepoOverride=$(get_image_repo_override $compName $subcompName $imageNameTag)
  if [ -n "${imageRepoOverride}" ] && [ "${imageRepoOverride}" != "null" ]; then
    resolvedRepo=${imageRepoOverride}
  fi
  echo $resolvedRepo
}

# Resolves the registry location for a subcomponent in the BOM; precedence is
# - the subcomponent definition
# - the global BOM definition
function resolve_subcomponent_registry_from_bom() {
  local compName=$1
  local subcompName=$2
  local subcompRegistry=$(cat ${BOM_FILE} | jq -r -c --arg comp ${compName} --arg subcomp ${subcompName} '.components[] | select(.name == $comp) | .subcomponents[] | select(.name == $subcomp) | .registry')
  if [ -z "$subcompRegistry" ] || [ "$subcompRegistry" == "null" ]; then
    subcompRegistry=$(get_bom_global_registry)
  fi
  echo $subcompRegistry
}

# Get the registry override for an image in the BOM
function get_image_registry_override() {
  local compName=$1
  local subcompName=$2
  local imageNameTag=$3
  local imageName=$(echo $imageNameTag | cut -d \: -f 1)
  local imageTag=$(echo $imageNameTag | cut -d \: -f 2)
  cat ${BOM_FILE} | jq -r -c --arg comp ${compName} --arg subcomp ${subcompName} --arg image ${imageName} --arg tag ${imageTag} \
    '.components[] | select(.name == $comp) | .subcomponents[] | select(.name == $subcomp) | .images[] | select(.image == $image and .tag == $tag) | .registry'
}

# Resolves the registry location for an image in the BOM; precedence is
# - the image definition
# - the subcomponent definition
# - the global BOM definition
function resolve_image_registry_from_bom() {
  local compName=$1
  local subcompName=$2
  local imageNameTag=$3

  local resolvedRegistry=$(resolve_subcomponent_registry_from_bom $compName $subcompName)
  local imageRegistryOverride=$(get_image_registry_override $compName $subcompName $imageNameTag)
  if [ -n "${imageRegistryOverride}" ] && [ "${imageRegistryOverride}" != "null" ]; then
    resolvedRegistry=${imageRegistryOverride}
  fi
  echo $resolvedRegistry
}

# List the base image names for a subcomponent of a component in the BOM, in the form <image-name>:<tag>
function list_subcomponent_images() {
  local compName=$1
  local subcompName=$2
  cat ${BOM_FILE} | jq -r --arg comp ${compName} --arg subcomp ${subcompName} \
    '.components[] | select(.name == $comp) | .subcomponents[] | select(.name == $subcomp) | .images[] | "\(.image):\(.tag)"'
}
