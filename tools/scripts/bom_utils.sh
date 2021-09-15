#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

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
  cat ${BOM_FILE} | jq -r --arg comp ${compName} '.components[] | select(.name==$comp) | .repository'
}

# Get the subcomponent registry
function get_subcomponent_registry() {
  local compName=$1
  local subCompName=$2
  cat ${BOM_FILE} | jq -r --arg comp ${compName} --arg subcomp ${subCompName} '.components[] | select(.name==$comp) | .subcomponents[] | select(.name==$subcomp) | .registry'
}

# Get the repository name for a subcomponent in the BOM
function get_subcomponent_repo() {
  local compName=$1
  local subCompName=$2
  cat ${BOM_FILE} | jq -r --arg comp ${compName} --arg subcomp ${subCompName} '.components[] | select(.name==$comp) | .subcomponents[] | select(.name==$subcomp) | .repository'
}

# List the subcomponents names within a component in the BOM
function list_subcomponent_names() {
  local compName=$1
  cat ${BOM_FILE} | jq -r --arg comp ${compName} \
    '.components[] | select(.name==$comp) | .subcomponents[] | .name '
}

# List the base image names for all subcomponents of a component in the BOM, in the form <image-name>:<tag>
function list_subcomponent_images() {
  local compName=$1
  local subCompName=$2
  cat ${BOM_FILE} | jq -r --arg comp ${compName} --arg subcomp ${subCompName} \
    '.components[] | select(.name==$comp) | .subcomponents[] | select(.name==$subcomp) | .images[] | "\(.image):\(.tag)"'
}
