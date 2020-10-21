#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/../install/common.sh

set -eu

function usage {
    consoleerr
    consoleerr "usage: $0 [-i image_list_file] [-s source_repo ] [-l source_repo_login] [-c source_repo_pwd] [-d destination_repo] [-u destination_repo_user] [-p destination_repo_pwd]"
    consoleerr "  -i image list file		The file that contains the list of images to pull/push. Defaults to verrazzano_img_list.txt"	
    consoleerr "  -s source_repo 		Source repository.  Defaults to ghcr.io."
    consoleerr "  -l source_repo_login 		Source repository user."
    consoleerr "  -c source_repo_pwd 		Source repository password."
    consoleerr "  -d destination_repo   	Destination repository.  Defaults to container-registry-admin.oraclecorp.com"
    consoleerr "  -u destination_repo_user   	Destination repository username."
    consoleerr "  -p destination_repo_pwd   	Destination repository password."
    consoleerr "  -h             		Help"
    consoleerr
    exit 0 
}

IMG_LIST_FILE=verrazzano_img_list.txt
SOURCE_REPO=ghcr.io
SOURCE_REPO_USER=foo
SOURCE_REPO_PWD=bar
DEST_REPO=container-registry-admin.oraclecorp.com
DEST_REPO_USER=foo
DEST_REPO_PWD=bar

while getopts i:s:l:c:d:u:p:h flag
do
    case "${flag}" in
        i) IMG_LIST_FILE=${OPTARG};;
        s) SOURCE_REPO=${OPTARG};;
        l) SOURCE_REPO_USER=${OPTARG};;
        c) SOURCE_REPO_PWD=${OPTARG};;
        d) DEST_REPO=${OPTARG};;
        u) DEST_REPO_USER=${OPTARG};;
        p) DEST_REPO_PWD=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done

log "logging in to $SOURCE_REPO"
docker login --username $SOURCE_REPO_USER --password $SOURCE_REPO_PWD $SOURCE_REPO
while IFS="" read -r p || [ -n "$p" ]
do
  log "pulling and tagging $SOURCE_REPO/$p"
  if ! docker pull $SOURCE_REPO/$p
  then
    consoleerr "$SOURCE_REPO/$p not found in repository"
    echo "$SOURCE_REPO/$p" >> skipped_images.txt
    continue
  else
    echo "$SOURCE_REPO/$p" >> pushed_images.txt
  fi
  docker tag $SOURCE_REPO/$p $DEST_REPO/$p
done < $IMG_LIST_FILE

docker logout $SOURCE_REPO

log "logging in to $DEST_REPO"
docker login --username $DEST_REPO_USER --password $DEST_REPO_PWD $DEST_REPO
# while IFS="" read -r p || [ -n "$p" ]
# do
#   log "pushing $DEST_REPO/$p"
#   docker push $DEST_REPO/$p
# done < pushed_images.txt

docker logout $DEST_REPO

