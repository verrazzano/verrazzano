#!/bin/bash
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Create a Docker repository in a specific compartment using the BOM file or an exploded tarball of Verrazzano images;
# useful for scoping a repo someplace other than the root compartment.
#
set -u

usage() {
  echo """
Create OCIR repos in a specific tenancy compartment for a set of images described in the Verrazzaon BOM file, or for a
set of exported Docker images tarballs located in a local directory .  This script reads the image repo information out
of each tar file and uses that to create a corresponding Docker repo in the target tenancy compartment, under that
tenancy's namespace.  You can provide either the full region name (e.g., "us-phoenix-1", or the region short name
(e.g., "phx").

See https://docs.oracle.com/en-us/iaas/Content/Registry/Tasks/registrycreatingarepository.htm for details on
repository creation in an OCI tenancy.

Usage:

$0  -p <parent-repo> -c <compartment-id> [ -r <region> ] [-d <path> | -b <bom-path>]

-r Region name (e.g., "us-phoenix-1")
-s Region short name (e.g., "phx", "lhr")
-p Parent repo, without the tenancy namespace
-c Compartment ID
-b BOM file location (default ./verrazzano-bom.json)
-d Images directory

Example, to create a repo in compartment ocid.compartment.oc1..blah, where the desired docker path with tenancy namespace
to the image is "myreporoot/testuser/myrepo/v8o", and the extracted tarball location is /tmp/exploded:

$0 -p myreporoot/testuser/myrepo/v8o -r uk-london-1 -c ocid.compartment.oc1..blah -d /tmp/exploded
  """
}

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
    '.components[] | select(.name == $comp) | .subcomponents[] | select(.name == $subcomp) | .images[] | "\(.image)"'
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
function create_image_repos_from_archives() {
  # Loop through tar files
  echo "Using local image downloads"
  for file in ${IMAGES_DIR}/*.tar; do
    if [ ! -e ${file} ]; then
      echo "Image tar file ${file} does not exist!"
      exit 1
    fi

    # Build up the image name and target image names, and create the repo
    local from_image=$(tar xOvf $file manifest.json | jq -r '.[0].RepoTags[0]')
    local from_image_name=$(basename $from_image | cut -d \: -f 1)
    local from_repository=$(dirname $from_image | cut -d \/ -f 2-)

    create_repo $from_repository $from_image_name
  done
}

function create_image_repos_from_bom() {
  # Loop through registry components
  echo "Creating image repos from ${BOM_FILE}"

  local components=($(list_components))
  echo "Components: ${components[*]}"

  for component in "${components[@]}"; do
    local subcomponents=($(list_subcomponents ${component}))
    for subcomp in "${subcomponents[@]}"; do
      echo "Processing images for Verrazzano subcomponent ${component}/${subcomp}"
      # Load the repository and base image names for the component
      local from_repository=$(get_subcomponent_repo $component $subcomp)
      local image_names=$(list_subcomponent_images $component $subcomp)

      for base_image in ${image_names}; do
        # Create a repo for each image
        create_repo $from_repository ${base_image}
      done
    done
  done
}

function create_repo() {
  local from_repository=$1
  local from_image_name=$2

  local repo_path=${from_repository}/${from_image_name}
  if [ -n "${PARENT_REPO}" ]; then
    repo_path=${PARENT_REPO}/${repo_path}
  fi

  local is_public="false"
  if [ "$from_repository" == "rancher" ] || [ "$from_image_name" == "verrazzano-platform-operator" ]; then
    # Rancher repos must be public
    is_public="true"
  fi

  echo "Creating repository ${repo_path} in ${REGION}, public: ${is_public}"
  oci --region ${REGION} artifacts container repository create --display-name ${repo_path}  \
    --is-public ${is_public} --compartment-id ${COMPARTMENT_ID}
  local rc=$?
  if [ "$?" != "0" ] && [ "$?" != "409" ]; then
    exit 1
  fi
}

IMAGES_DIR=.
REGION=""
REGION_SHORT_NAME=""
USE_LOCAL_IMAGES="false"

while getopts ":s:c:r:p:d:b:" opt; do
  case ${opt} in
  c) # compartment ID
    COMPARTMENT_ID=${OPTARG}
    ;;
  r) # region
    REGION=${OPTARG}
    ;;
  p) # parent repo
    PARENT_REPO=${OPTARG}
    ;;
  d) # images dir
    IMAGES_DIR="${OPTARG}"
    USE_LOCAL_IMAGES="true"
    ;;
  b) # BOM file
    BOM_FILE="${OPTARG}"
    ;;
  s )
    REGION_SHORT_NAME="${OPTARG}"
    ;;
  \?)
    usage
    ;;
  esac
done
shift $((OPTIND - 1))

if [ -z "${REGION}" ]; then
  if [ -z "${REGION_SHORT_NAME}" ]; then
    echo "Must provide either the full or the short region name"
    usage
    exit 1
  fi
  echo "REGION_SHORT_NAME=$REGION_SHORT_NAME"
  REGION=$(oci --region us-phoenix-1 iam region list | jq -r  --arg regionAbbr ${REGION_SHORT_NAME} '.data[] | select(.key|test($regionAbbr;"i")) | .name')
  if [ -z "${REGION}" ] || [ "null" == "${REGION}" ]; then
    echo "Invalid short region name ${REGION_SHORT_NAME}"
    usage
    exit 1
  fi
fi

if [ -z "${PARENT_REPO}" ]; then
  echo "Repository pattern not provided"
  usage
  exit 1
fi
if [ -z "${COMPARTMENT_ID}" ]; then
  echo "Compartment ID not provided"
  usage
  exit 1
fi

if [ "${USE_LOCAL_IMAGES}" == "true" ]; then
  create_image_repos_from_archives
else
  create_image_repos_from_bom
fi
