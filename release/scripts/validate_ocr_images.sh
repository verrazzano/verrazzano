#!/bin/bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

echo Running OCR image checks ...
export DOCKER_CLI_EXPERIMENTAL=enabled
VZ_IMAGE_TXT=verrazzano_images.txt
IMAGES=$(cat "$VZ_IMAGE_TXT")

for image in "$IMAGES"; do
    IMAGE_NAME_AND_TAG=$(echo "$image" | awk -F '[/]' '{print $2}')
    # docker manifest inspect "$IMAGE_NAME_AND_TAG"
done

echo Running manifest inspect . . .
echo "$OCR_CREDS_PSW" | docker login "$DOCKER_REPO" -u "$OCR_CREDS_USR" --password-stdin

docker image inspect alertmanager:v0.24.0-20221206192620-fb73d30f
