#!/bin/bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# We enabled the experimental Docker CLI to be able to run 'docker pull'
export DOCKER_CLI_EXPERIMENTAL=enabled

echo Running OCR image checks ...
VZ_IMAGE_TXT=verrazzano_images.txt
SUCCESSFULLY_PULLED_IMAGES=()

echo Logging into Docker ...
echo "$OCR_CREDS_PSW" | docker login "$DOCKER_REPO" -u "$OCR_CREDS_USR" --password-stdin

echo Running docker image inspect ...
while IFS= read -r line
do
    IMAGE_NAME_AND_TAG=$(echo "$line")
    IMAGE_PULL=$(docker pull -q "$DOCKER_REPO"/"$IMAGE_NAME_AND_TAG")
    if [[ "$IMAGE_PULL" -eq 0 ]]; then
        echo Success Downloading image ...
        echo "$IMAGE_PULL"
        SUCCESSFULLY_PULLED_IMAGES+=("$IMAGE_PULL")
    fi

done < "$VZ_IMAGE_TXT"

echo "$SUCCESSFULLY_PULLED_IMAGES"