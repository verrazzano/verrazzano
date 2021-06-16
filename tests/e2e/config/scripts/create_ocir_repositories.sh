#!/bin/bash
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Create a Docker repository in a specific compartment using an exploded tarball of Verrazzano images; useful for
# scoping a repo someplace other than the root compartent.
#
set -u

usage() {
  echo """
Create OCIR repos for a set of exported Docker images tarballs located in a local directory in a specific
tenancy compartment.  This script reads the image repo information out of each tar file and uses that to create
a corresponding Docker repo in the target tenancy compartment, under that tenancy's namespace.  You can provide either
the full region name (e.g., "us-phoenix-1", or the region short name (e.g., "phx").

See https://docs.oracle.com/en-us/iaas/Content/Registry/Tasks/registrycreatingarepository.htm for details on
repository creation in an OCI tenancy.

Usage:

$0  -p <parent-repo> -c <compartment-id> [ -r <region> -d <path> ]

-r Region name (e.g., "us-phoenix-1")
-s Region short name (e.g., "phx", "lhr")
-p Parent repo, without the tenancy namespace
-c Compartment ID
-d Images directory

Example, to create a repo in compartment ocid.compartment.oc1..blah, where the desired docker path with tenancy namespace
to the image is "stevengreenberginc/testuser/myrepo/v8o", and the extracted tarball location is /tmp/exploded:

$0 -p testuser/myrepo/v8o -r uk-london-1 -c ocid.compartment.oc1..blah -d /tmp/exploded
  """
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

    # Build up the image name and target image names, and do a tag/push
    local from_image=$(tar xOvf $file manifest.json | jq -r '.[0].RepoTags[0]')
    local from_image_name=$(basename $from_image | cut -d \: -f 1)
    local from_repository=$(dirname $from_image | cut -d \/ -f 2-)

    local is_public="false"
    if [ "$from_repository" == "rancher" ]; then
      # Rancher repos must be public
      is_public="true"
    fi

    local repo_path=${from_repository}/${from_image_name}
    if [ -n "${PARENT_REPO}" ]; then
      repo_path=${PARENT_REPO}/${repo_path}
    fi

    echo "Creating repository ${repo_path} in ${REGION}, public: ${is_public}"
    oci --region ${REGION} artifacts container repository create --display-name ${repo_path} \
      --is-public ${is_public} --compartment-id ${COMPARTMENT_ID}
  done
}

IMAGES_DIR=.
REGION=""
REGION_SHORT_NAME=""

while getopts ":s:c:r:p:d:" opt; do
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

create_image_repos_from_archives
