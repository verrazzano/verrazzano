#!/bin/bash
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Script to allow users to load Verrazzano images into a private Docker registry.
# - Using the Verrazzano BOM, pull/tag/push images into a target registry/repository
# - Using a local cache of Verrazzano image tarballs, load into the local Docker registry and push to the remote registry/repo
# - Clean up the local registry
#
set -o pipefail
set -o errtrace

# variables
TO_REGISTRY=
TO_REPO=
BOM_FILE=./verrazzano-bom.json
USELOCAL=0
IMAGES_DIR=
INCREMENTAL_CLEAN=false
CLEAN_ALL=false
DRY_RUN=false
INCLUDE_COMPONENTS=
EXCLUDE_COMPONENTS=
LIST_IMAGES_ONLY=

function usage() {
  ec=${1:-0}
  echo """
This script is to help pushing Verrazzano container images into a private repository from their default locations
usage:

  $0 -t <docker-registry> [-l <archive-path> -r <repository-path>]
  $0 -t <docker-registry> [-b <path> -r <repository-path>]
  $0 -c [-b <path> | -l <archive-path>]

Options:
 -t <docker-registry>   Target docker registry to push to, e.g., iad.ocir.io
 -r <repository-path>   Repository name/prefix for each image, e.g \"path/to/my/image\"; if not specified the default will be used according to the BOM
 -b <path>              Bill of materials (BOM) of Verrazzano components; if not specified, defaults to ./verrazzano-bom.json
 -l <archive-dir>       Use the specified directory to load local Docker image tarballs from instead of pulling from
 -i <component>         Include the specified component in the operation (can be repeated for multiple components)
 -e <component>         Exclude the specified component from the operation (can be repeated for multiple components)
 -c                     Clean all local images/tags
 -z                     Incrementally clean each local image after it has been successfully pushed
 -d                     Dry-run only, do not perform Docker operations
 -o                     List components
 -m <outputfile>        List images to output file, do not process them

Examples:

  # Loads all images into registry 'myreg.io' using the default repository paths for each image in the BOM
  $0 -t myreg.io

  # Loads all Verrazzano images into registry 'myreg.io' in repository 'myrepo/user1'
  $0 -t myreg.io -r 'myrepo/user1'

  # Loads all Verrazzano images into registry 'myreg.io' in repository 'myrepo/user1' using the BOM /path/to/my-bom.json
  # and removes the locally downloaded image after a successful push
  $0 -c -t myreg.io -r 'myrepo/user1' -b /path/to/my-bom.json

  # Loads all Docker tarball images in the directory /path/to/exploded/tarball into registry 'myreg.io' in repository 'myrepo'
  $0 -t myreg.io -l /path/to/exploded/tarball -r myrepo

  # Do a dry-run with the tarball location /path/to/exploded/tarball with registry 'myreg.io' in repository 'myrepo'
  $0 -d -t myreg.io -l /path/to/exploded/tarball -r myrepo

  # List out the Verrazzano components in the default BOM file
  $0 -d -t myreg.io -o

  # List out the Verrazzano components in the specified BOM file
  $0 -d -t myreg.io -o -b /path/to/my-bom.json

  # Processes all Verrazzano images *except* verrazzano-platform-operator and verrazzano-application-operator
  $0 -t myreg.io -r 'myrepo/user1' -b /path/to/my-bom.json -e verrazzano-platform-operator -e verrazzano-application-operator

  # Processes *only* the images for the Verrazzano components cert-manager and istio
  $0 -t myreg.io -r 'myrepo/user1' -b /path/to/my-bom.json -i cert-manager -i istio

"""
  exit ${ec}
}

function exit_trap() {
  local rc=$?
  local lc="$BASH_COMMAND"

  if [[ $rc -ne 0 ]]; then
    echo "Command [$lc] exited with code [$rc]"
  fi
}

trap exit_trap EXIT

function run_docker() {
  local fatal=false
  if [ "${DRY_RUN}" != "true" ]; then
    tmpfile=$(mktemp -t vz-helper-docker-err.XXXXXX)
    docker $* 2>${tmpfile}
    local result=$?
    local denied=$(egrep -i "(Anonymous|permission_denied|403.*forbidden)" ${tmpfile})
    if [ -n "$denied" ]; then
       echo """

Permission error from Docker:

$(cat ${tmpfile})

Please log into the target registry and try again.

      """
      fatal=true
    elif [ "${result}" != "0" ]; then
      echo "An error occurred running docker command:"
      cat ${tmpfile}
    fi
  fi
  rm ${tmpfile}
  if [ "${fatal}" == "true" ]; then
    # Fatal error occurred, exit immeditately
    exit 1
  fi
  return ${result}
}

# Wrapper for Docker pull
function load() {
  archive=$1

  echo ">> Loading archive: ${archive}"
  run_docker load -i "${archive}"
}

# Wrapper for Docker pull
function pull() {
  image=$1

  echo ">> Pulling image: ${image}"
  run_docker pull "${image}"
}

# Wrapper for Docker tag
function tag() {
  from_image=$1
  to_image=$2

  echo ">> Tagging image: ${from_image} to ${to_image}"
  run_docker tag "${from_image}" "${to_image}"
}

# Wrapper for Docker push
function push() {
  image=$1

  echo ">> Pushing image: ${image}"
  run_docker push "${image}"
}

# Wrapper for Docker rmi
function remove() {
  local images=$*
  echo ">> Removing images: $*"
  run_docker rmi ${images}
}

# Perform requirements checks and validate arguments
function check() {
  if [ "${INCREMENTAL_CLEAN}" == "true" ] && [ "${CLEAN_ALL}" == "true" ]; then
    echo "Incremental clean and clean-all both specified, these can not be used together"
    usage 1
  fi

  if [ "$USELOCAL" -ne 0 ]; then
    echo "Use local images specified, ignoring -b if set"
    if [ -z "${IMAGES-DIR}" ]; then
      echo "Use local images specified, but no location specified"
      usage 1
    fi
  fi

  if [ -z "${TO_REGISTRY}" ]; then
    echo "Target registry not specified!"
    usage 1
  fi

  echo "Checking if docker is installed ..."
  if ! docker --help >/dev/null; then
    echo "[ERROR] docker is not installed, please install docker"
    usage 1
  fi

  echo "Checking if jq is installed ..."
  if ! jq --help >/dev/null; then
    echo "[ERROR] jq is not install ... please install jq"
    usage 1
  fi

}

# Process an image
# - Do pull (if necessary) and tag, and then push to the new registry
# - attempts up to 10 times before failing
# - Cleans up the locally downloaded/loaded image when done
function process_image() {
  local from_image=$1
  local to_image=$2

  if [ "${CLEAN_ALL}" == "true" ]; then
    remove ${to_image} ${from_image}
    return 0
  fi

  echo "Processing image ${from_image} to ${to_image}"
  local success=false
  for i in {1..10}; do
    # Only pull the image if we are not looking at local images
    if [ "$USELOCAL" -eq 0 ]; then
      pull "${from_image}"
      if [[ $? -ne 0 ]]; then
        sleep 30
        continue
      fi
    fi

    tag ${from_image} ${to_image}
    if [[ $? -ne 0 ]]; then
      sleep 30
      continue
    fi

    # push
    push ${to_image}
    if [[ $? -ne 0 ]]; then
      sleep 30
      continue
    fi

    success=true
    break
  done

  if [ "${INCREMENTAL_CLEAN}" == "true" ]; then
    remove ${to_image} ${from_image}
  fi

  if [[ "${success}" == "false" ]]; then
    echo "[ERROR] Failed to manage image [${from_image}]"
    exit 1
  fi
}

# Get the global Docker registry specified in the BOM
function get_bom_global_registry() {
  cat ${BOM_FILE} | jq -r '.registry'
}

# Get the list of component names in the BOM
function list_components() {
  cat ${BOM_FILE} | jq -r '.components[].name'
}

# List all the subcomponents for a component in the BOM
function list_subcomponents() {
  local compName=$1
  cat ${BOM_FILE} | jq -r --arg comp ${compName} '.components[] | select(.name == $comp) | .subcomponents[].name'
}

# Get the repository name for a subcomponent in the BOM
function get_subcomponent_repo() {
  local compName=$1
  local subcompName=$2
  cat ${BOM_FILE} | jq -r -c --arg comp ${compName} --arg subcomp ${subcompName} '.components[] | select(.name == $comp) | .subcomponents[] | select(.name == $subcomp) | .repository'
}

function get_subcomponent_registry() {
  local compName=$1
  local subcompName=$2
  local subcompRegistry=$(cat ${BOM_FILE} | jq -r -c --arg comp ${compName} --arg subcomp ${subcompName} '.components[] | select(.name == $comp) | .subcomponents[] | select(.name == $subcomp) | .registry')
  if [ -z "$subcompRegistry" ] || [ "$subcompRegistry" == "null" ]; then
    subcompRegistry=$(get_bom_global_registry)
  fi
  echo $subcompRegistry
}

# List the base image names for a subcomponent of a component in the BOM, in the form <image-name>:<tag>
function list_subcomponent_images() {
  local compName=$1
  local subcompName=$2
  cat ${BOM_FILE} | jq -r --arg comp ${compName} --arg subcomp ${subcompName} \
    '.components[] | select(.name == $comp) | .subcomponents[] | select(.name == $subcomp) | .images[] | "\(.image):\(.tag)"'
}

# Get the target repo if overridden, otherwise return the provided default
function get_target_repo() {
  local default_repo=$1
  local target_repo=${TO_REPO}
  if [ -z "${target_repo}" ]; then
    target_repo=${default_repo}
  fi
  echo "${target_repo}"
}

# Main driver for processing images from a locally downloaded set of tarballs
function process_local_archives() {
  # Loop through tar files
  echo "Using local image downloads"
  for file in ${IMAGES_DIR}/*.tar; do
    if [ ! -e ${file} ]; then
      echo "Image tar file ${file} does not exist!"
      exit 1
    fi
    # load tar file into the local Docker registry
    load $file

    # Build up the image name and target image names, and do a tag/push
    local from_image=$(tar xOvf $file manifest.json | jq -r '.[0].RepoTags[0]')
    local from_image_name=$(basename $from_image)
    local from_repository=$(dirname $from_image | cut -d \/ -f 2-)

    local target_repo=$(get_target_repo ${from_repository})
    local to_image=${TO_REGISTRY}/${target_repo}/${from_repository}/${from_image_name}
    if [ ! -z "$LIST_IMAGES_ONLY" ]; then
      echo ${to_image} >> $LIST_IMAGES_ONLY
    else
      process_image ${from_image} ${to_image}
    fi
  done
}

# Returns 0 if the specified component is in the excludes list, 1 otherwise
function is_component_excluded() {
    local seeking=$1
    local excludes=(${EXCLUDE_COMPONENTS})
    local in=1
    for comp in "${excludes[@]}"; do
        if [[ "$comp" == "$seeking" ]]; then
            in=0
            break
        fi
    done
    return $in
}

# Main driver for pulling/tagging/pushing images based on the Verrazzano bill of materials (BOM)
function process_images_from_bom() {
  # Loop through registry components
  echo "Using image registry ${BOM_FILE}"

  local components=(${INCLUDE_COMPONENTS})
  if [ "${#components[@]}" == "0" ]; then
    components=($(list_components))
  fi

  echo "Components: ${components[*]}"

#  local components=($(list_components))
  for component in "${components[@]}"; do
    if is_component_excluded ${component} ; then
      echo "Component ${component} excluded"
      continue
    fi
    local subcomponents=($(list_subcomponents ${component}))
    for subcomp in "${subcomponents[@]}"; do
      echo "Processing images for Verrazzano subcomponent ${component}/${subcomp}"
      # Load the repository and base image names for the component
      local from_registry=$(get_subcomponent_registry $component $subcomp)
      local from_repository=$(get_subcomponent_repo $component $subcomp)
      local image_names=$(list_subcomponent_images $component $subcomp)

      local from_image_prefix=${from_registry}
      if [ -n "${from_repository}" ] && [ "${from_repository}" != "null" ]; then
        from_image_prefix=${from_image_prefix}/${from_repository}
      fi

      local to_image_prefix=${TO_REGISTRY}
      if [ -n "${TO_REPO}" ]; then
        to_image_prefix=${to_image_prefix}/${TO_REPO}
      fi
      to_image_prefix=${to_image_prefix}/${from_repository}

      for base_image in ${image_names}; do
        # Build up the image name and target image name, and do a pull/tag/push
        local from_image=${from_image_prefix}/${base_image}
        local to_image=${to_image_prefix}/${base_image}
        if [ ! -z "$LIST_IMAGES_ONLY" ]; then
          echo ${to_image} >> $LIST_IMAGES_ONLY
        else
          process_image ${from_image} ${to_image}
        fi
      done
    done
  done
}

output_bom_components() {
  echo """
Verrazzano components in BOM file ${BOM_FILE}:

$(list_components)
  """
}

# Main fn
function main() {
  if [ "$USELOCAL" != "0" ]; then
    process_local_archives
  else
    process_images_from_bom
  fi
  if [ "${CLEAN_ALL}" == "true" ]; then
    echo "[SUCCESS] All local images cleaned"
  else
    if [ ! -z "$LIST_IMAGES_ONLY" ]; then
      echo "[SUCCESS] All images listed"
    else
      echo "[SUCCESS] All images pushed to [${TO_REGISTRY}]"
    fi
  fi
}

while getopts 'hzcdom:b:t:f:r:l:i:e:' opt; do
  case $opt in
  d)
    DRY_RUN=true
    ;;
  b)
    BOM_FILE=$OPTARG
    ;;
  d)
    DB_DUMP=$OPTARG
    ;;
  e)
    echo "Exclude component: ${OPTARG}"
    EXCLUDE_COMPONENTS="${EXCLUDE_COMPONENTS} ${OPTARG}"
    ;;
  i)
    echo "Include component: ${OPTARG}"
    INCLUDE_COMPONENTS="${INCLUDE_COMPONENTS} ${OPTARG}"
    ;;
  r)
    TO_REPO=$OPTARG
    ;;
  t)
    TO_REGISTRY=$OPTARG
    ;;
  f)
    TARBALL=$OPTARG
    ;;
  z)
    INCREMENTAL_CLEAN=true
    ;;
  c)
    CLEAN_ALL=true
    ;;
  o)
    output_bom_components
    exit 0
    ;;
  m)
    LIST_IMAGES_ONLY="${OPTARG}"
    ;;
  l)
    USELOCAL=1
    IMAGES_DIR="${OPTARG}"
    ;;
  h | ?)
    usage
    ;;
  esac
done

# Check the system requirements and arguments
check
# Exec main
main
