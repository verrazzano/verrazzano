#!/bin/bash

set -u

usage() {
  echo """
usage:

$0  -p <parent-repo> -c <compartment-id> [ -r <region> -d <path> ]

-r Region name
-p Parent repo
-c Compartment ID
-d Images directory
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

    local repo_path=${from_repository}/${from_image_name}
    if [ -n "${PARENT_REPO}" ]; then
      repo_path=${PARENT_REPO}/${repo_path}
    fi
    set -x
    oci --region ${REGION} artifacts container repository create --display-name ${repo_path} --compartment-id ${COMPARTMENT_ID}
    set +x
  done
}

IMAGES_DIR=.
REGION=us-phoenix-1

while getopts ":c:r:p:d:" opt; do
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
  \?)
    usage
    ;;
  esac
done
shift $((OPTIND - 1))

if [ -z "${PARENT_REPO}" ]; then
  echo "Repository pattern not provided"
fi
if [ -z "${COMPARTMENT_ID}" ]; then
  echo "Compartment ID not provided"
fi

create_image_repos_from_archives
