#!/bin/bash
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Script for soft deletion images in the Harbor project.
# Garbage collection setting for actual deletion of data must be enabled in the Harbor instance separately.

set -u

# variables
REST_API_BASE_URL=
USERNAME=
PASSWORD=
PROJECT_NAME=
IMAGES_DIR=
IMAGE_REPO_SUBPATH_PREFIX=

function usage() {
  echo """
  Script for soft deletion images in the Harbor project. Garbage collection setting for actual deletion of data must be
  enabled in the Harbor instance separately.

  Usage:

  $0 -a <harbor-rest-api-base-url> -u <username> -p <password> -m <project-name> -l <images-directory> -i <image-repository-subpath-prefix>

  Options:
   -a <harbor-rest-api-base-url>          Base URL of the Harbor REST API (example https://<your-harbor-instance-domain>/api/v2.0)
   -u <username>                          Username with permissions to create a project in Harbor
   -p <password>                          Password for the corresponding username
   -m <project-name>                      The name of the project to be created
   -l <images-directory>                  Images directory
   -i <image-repository-subpath-prefix>   Subpath prefix for the image repository
   -h                                     Display help usage
  """
  exit 0
}

function delete_repositories() {
  local replacement="%252F"
  # Replace each occurrence of "/" with the replacement string
  # Example convert devname/vz-xxxx-abcdef/b14 to devname%252Fvz-xxxx-abcdef%252Fb14
  local imageSubPathPrefix="${IMAGE_REPO_SUBPATH_PREFIX////$replacement}"

  for file in ${IMAGES_DIR}/*.tar; do
    if [ ! -e ${file} ]; then
      echo "Image tar file ${file} does not exist!"
      exit 1
    fi

    local from_image=$(tar xOvf $file manifest.json | jq -r '.[0].RepoTags[0]')
    # Retrieve the image name without version information
    local imageSubPathSuffix=$(echo $from_image | cut -d \: -f 1)
    # Return the entire substring after the first occurrence of "/"
    # Example convert myreg.io/myproject/hello-world to myproject/hello-world
    imageSubPathSuffix=${imageSubPathSuffix#*/}
    # Replace each occurrence of "/" with the replacement string
    imageSubPathSuffix=${imageSubPathSuffix////$replacement}

    local imageSubPath="$imageSubPathPrefix$replacement$imageSubPathSuffix"

    local fullImageUrl="$REST_API_BASE_URL/projects/$PROJECT_NAME/repositories/$imageSubPath"
    echo "Proceeding to delete image: $fullImageUrl"
    curl --user $USERNAME:$PASSWORD -X DELETE $fullImageUrl -H "accept: application/json"
  done
}

while getopts 'ha:u:p:m:l:i:' opt; do
  case $opt in
  a)
    REST_API_BASE_URL=$OPTARG
    ;;
  u)
    USERNAME=$OPTARG
    ;;
  p)
    PASSWORD=$OPTARG
    ;;
  m)
    PROJECT_NAME=$OPTARG
    ;;
  l)
    IMAGES_DIR=$OPTARG
    ;;
  i)
    IMAGE_REPO_SUBPATH_PREFIX=$OPTARG
    ;;
  h | ?)
    usage
    ;;
  esac
done

delete_repositories