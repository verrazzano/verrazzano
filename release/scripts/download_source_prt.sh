#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# A script to download the source code for the images build from source.

set -e

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ -z "$1" ]; then
  echo "Specify the file containing the list of images"
  exit 1
fi

if [ -z "$2" ]; then
  echo "Specify the file containing the repository URLs"
  exit 1
fi

if [ -z "$3" ]; then
  echo "Specify an existing directory to download the source code from various repositories"
  exit 1
fi

IMAGES_TO_PUBLISH=$1
REPO_URL_PROPS=$2
SAVE_DIR=$3
DRY_RUN=${4:-false}

# Verrazzano repository, which is prefix for each of the entries in IMAGES_TO_PUBLISH
VZ_REPO_PREFIX="verrazzano/"

# Read the components from IMAGES_TO_PUBLISH file and download the source from the corresponding repositories
function processImagesToPublish() {
  local imagesToPublish=$1
  if [ -f "${imagesToPublish}" ];
  then
    while IFS=':' read -r key value
    do
      # Skip empty lines and comments starting with #
      case $key in
       ''|\#*) continue ;;
      esac
      key=$(echo $key | tr '.' '_')
      key=${key#$VZ_REPO_PREFIX}

      # Remove till last - from value to get the short commit
      value=${value##*-}

      downloadSourceCode "$key" "${value}"
    done < "${imagesToPublish}"
  else
    echo "$imagesToPublish here not found."
  fi
}

# Download source using git clone,
function downloadSourceCode() {
  local compKey=$1
  local shortCommit=$2
  local repoUrl=""
  local keyFound=false
  while IFS='=' read -r key value
  do
    key=$(echo $key | tr '.' '_')
    if [ "${compKey}" = "${key}" ]; then
      repoUrl=${value}
      keyFound=true
      break
    fi
  done < "${REPO_URL_PROPS}"

  # Fail when the property for the component is not defined in REPO_URL_PROPS
  if [ "${keyFound}" == false ]; then
    if [ "$DRY_RUN" == true ]; then
      echo "The repository URL for the component ${compKey} is missing from file ${REPO_URL_PROPS}"
      exit 1
    else
      continue
    fi
  fi

  # When DRY_RUN is set to true, do not download the source
  if [ "$DRY_RUN" == true ] ; then
    continue
  fi

  # Consider the value SKIP_CLONE for a property, to ignore cloning the repository
  # Also skip when there is no property defined for the compKey.
  if [ "${repoUrl}" = "SKIP_CLONE" ] || [ "${repoUrl}" = "" ]; then
    continue
  fi

  cd "${SAVE_DIR}"

  # Create a blobless clone, downloads all reachable commits and trees while fetching blobs on-demand.
  git clone --filter=blob:none "${value}"

  # In most of the cases, we should be able to cd ${key}, find change dir only when previous cd fails
  # Something like cd "${key}" || { changeDir=$(getChangeDir "${value}") cd "${changeDir} }
  # But need to find a way to avoid the error message when cd "${key}" fails
  changeDir=$(getChangeDir "${value}")
  cd "${changeDir}"

  # -c advice.detachedHead=false is used to avoid the Git message for the detached HEAD
  git -c advice.detachedHead=false checkout "${shortCommit}"

  # Remove git history and other files
  rm -rf .git .gitignore .github .gitattributes
  printf "\n"
}

# Handle the examples repository as a special case and get the source from master/main branch
function downloadSourceExamples() {
  local repoUrl=$(getRepoUrl "examples")
  if [ "${repoUrl}" = "" ]; then
    return
  fi
  cd "${SAVE_DIR}"
  git clone "${repoUrl}"
  cd examples
  # Remove git history and other files
  rm -rf .git .gitignore .github .gitattributes
  printf "\n"
}

# Fetch the URL for the repository based on the a given property
function getRepoUrl {
    local propKey=$1
    grep "^\\s*${propKey}=" "${REPO_URL_PROPS}"|cut -d'=' -f2
}

# Derive the directory from where to do - git checkout
function getChangeDir() {
  local repoUrl=$1
  repoUrl=${repoUrl##*/}
  echo "${repoUrl%%.git}"
}

# If SAVE_DIR exists, fail if it has files / directories
if [[ ! -d "${SAVE_DIR}" ]] && [[ "$DRY_RUN" == false ]]; then
  mkdir -p ${SAVE_DIR}
fi

if [[ ! -f "${REPO_URL_PROPS}" ]]; then
  echo "Input file ${REPO_URL_PROPS} doesn't exist"
  exit 1
fi

processImagesToPublish "${IMAGES_TO_PUBLISH}"
downloadSourceExamples

if [ "$DRY_RUN" == true ] ; then
   echo "Completed running the script with DRY_RUN = true"
else
  echo "Completed archiving source code, take a look at the contents of ${SAVE_DIR}"
fi
