#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# A script to download the source code for the images build from source.

set -e
set -o pipefail
set -o errtrace

# variables
IMAGES_TO_PUBLISH=
REPO_URL_PROPS=
ADDITIONAL_REPO_URLS=
SAVE_DIR=
DRY_RUN=false
CLEAN_SAVE_DIR=false

# Verrazzano repository, which is prefix for each of the entries in IMAGES_TO_PUBLISH
VZ_REPO_PREFIX="verrazzano/"

function exit_trap() {
  local rc=$?
  local lc="$BASH_COMMAND"

  if [[ $rc -ne 0 ]]; then
    echo "Command exited with code [$rc]"
  fi
}

trap exit_trap EXIT

function usage() {
  ec=${1:-0}
  echo """
A script to download the source code for the images used by Verrazzano..

Options:
-i Verrazzano image list
-r File containing the repository URLs to download the source
-a Addition source repository URLs
-s An empty directory to download the source
-d Dry run to ensure file specified by -r flag contains the URLs for all the images listed in the file specified by -i flag
-c Clean the directory to download the source

Examples:
  # Download source to /tmp/source_dir by reading the components from mydir/verrazzano_images.txt and repository URLs from mydir/repo_url.properties
  $0 -i mydir/verrazzano_images.txt -r mydir/repo_url.properties -s mydir/source_dir

"""
  exit ${ec}
}

# Validate the input flags
function validateFlags() {
  if [ -z "${ADDITIONAL_REPO_URLS}" ] || [ "${ADDITIONAL_REPO_URLS}" == "" ]; then
    if [ "${REPO_URL_PROPS}" == "" ]; then
      echo "The file containing the repository URLs to download the source is required, but not specified by flag -r"
      usage 1
    fi
    if [[ ! -f "${REPO_URL_PROPS}" ]]; then
      echo "The file containing the repository URLs specified by flag -r doesn't exist"
      usage 1
    fi
  fi

  if [ "${DRY_RUN}" == false ]; then
    if [ "${SAVE_DIR}" == "" ]; then
      echo "The directory to save the source is required, but not specified by flag -s"
      usage 1
    fi
  fi
}

# Initialize the source download
function initDownload() {
  # Create SAVE_DIR, if it doesn't exist
  if [[ ! -d "${SAVE_DIR}" ]] && [[ "$DRY_RUN" == false ]]; then
    mkdir -p ${SAVE_DIR}
  fi

  # Clean SAVE_DIR when $CLEAN_SAVE_DIR is true
  if [[ "$CLEAN_SAVE_DIR" == true ]] && [[ "$DRY_RUN" == false ]]; then
    rm -rf ${SAVE_DIR}/*
  fi
}

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
  local commitOrBranch=$2
  local repoUrl=$3
  if [ "${repoUrl}" = "" ]; then
    repoUrl=$(getRepoUrl "${key}")
    if [ "${repoUrl}" = "" ]; then
      if [ "$DRY_RUN" == true ]; then
        echo "The repository URL for the component ${compKey} is missing from file ${REPO_URL_PROPS}"
        exit 1
      else
        continue
      fi
    fi
  fi

  # When DRY_RUN is set to true, do not download the source
  if [ "$DRY_RUN" == true ] ; then
    continue
  fi

  # Consider the value SKIP_CLONE for a property, to ignore cloning the repository
  if [ "${repoUrl}" = "SKIP_CLONE" ]; then
    continue
  fi

  cd "${SAVE_DIR}"
  changeDir=$(getChangeDir "${repoUrl}")
  if [ -d "${SAVE_DIR}/${changeDir}" ]; then
    continue
  fi

  # Create a blobless clone, downloads all reachable commits and trees while fetching blobs on-demand.
  git clone --filter=blob:none "${repoUrl}"
  cd "${changeDir}"

  # -c advice.detachedHead=false is used to avoid the Git message for the detached HEAD
  git -c advice.detachedHead=false checkout "${commitOrBranch}"

  # Remove git history and other files
  rm -rf .git .gitignore .github .gitattributes
  printf "\n"
}

# Handle the examples repository as a special case and get the source from master/main branch
function downloadSourceExamples() {
  if [ "$DRY_RUN" == true ]; then
    return
  fi
  local repoUrl=$(getRepoUrl "examples")
  if [ "${repoUrl}" = "" ]; then
    return
  fi
  cd "${SAVE_DIR}"
  if [ -d "${SAVE_DIR}/examples" ]; then
    continue
  fi

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

# Download additional source for the component, which is not part of verrazzano-bom.json
function downloadAdditionalSource() {
  local additionalSource=$1
  if [ -f "${additionalSource}" ];
  then
    while IFS='=' read -r key value
    do
      # Skip empty lines and comments starting with #
      case $key in
       ''|\#*) continue ;;
      esac
      key=$(echo $key | tr '.' '_')
      url=$(echo "${value}"|cut -d':' -f2-)
      branchInfo=$(echo "${value}"|cut -d':' -f1)
      downloadSourceCode "$key" ${branchInfo} "${url}"
    done < "${additionalSource}"
  else
    echo "$additionalSource here not found."
  fi
}

# Read input flags
while getopts 'a:i:r:s:d:c:' flag; do
  case $flag in
  a)
    ADDITIONAL_REPO_URLS=$OPTARG
    ;;
  i)
    IMAGES_TO_PUBLISH=$OPTARG
    ;;
  r)
    REPO_URL_PROPS=$OPTARG
    ;;
  s)
    SAVE_DIR=$OPTARG
    ;;
  d)
    DRY_RUN=true
    ;;
  c)
    CLEAN_SAVE_DIR=true
    ;;
  h | ?)
    usage
    ;;
  esac
done

# Validate the command line flags
validateFlags
initDownload

if [ "${ADDITIONAL_REPO_URLS}" == "" ]; then
  processImagesToPublish "${IMAGES_TO_PUBLISH}"
  downloadSourceExamples
else
  downloadAdditionalSource "${ADDITIONAL_REPO_URLS}"
fi

if [ "$DRY_RUN" == true ] ; then
   echo "Completed running the script with DRY_RUN = true"
else
  echo "Completed archiving source code, take a look at the contents of ${SAVE_DIR}"
fi
