#!/bin/bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

echo Running OCR image checks ...
export DOCKER_CLI_EXPERIMENTAL=enabled
VZ_IMAGE_TXT=verrazzano_images.txt
IMAGES=$(cat "$VZ_IMAGE_TXT")

echo Logging into Docker ...
echo "$OCR_CREDS_PSW" | docker login "$DOCKER_REPO" -u "$OCR_CREDS_USR" --password-stdin

echo Running docker image inspect ...
for image in "$IMAGES"; do
    IMAGE_NAME_AND_TAG=$(echo "$image")
    docker pull "$DOCKER_REPO"/"$IMAGE_NAME_AND_TAG"
    # docker image inspect "$DOCKER_REPO"/"$IMAGE_NAME_AND_TAG"
done
