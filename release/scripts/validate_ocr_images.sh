#!/bin/bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# We enabled the experimental Docker CLI to be able to run 'docker pull'
export DOCKER_CLI_EXPERIMENTAL=enabled

echo Running OCR image checks ...
VZ_IMAGE_TXT=verrazzano_images.txt
SUCCESSFULLY_PULLED_IMAGES=("")

echo Logging into Docker ...
echo "$OCR_CREDS_PSW" | docker login "$DOCKER_REPO" -u "$OCR_CREDS_USR" --password-stdin

echo Pulling images from OCR ... 
while IFS= read -r line
do
    IMAGE_NAME_AND_TAG=$(echo "$line")
    IMAGE_PULL=$(docker pull "$DOCKER_REPO"/"$IMAGE_NAME_AND_TAG")
    if [[ "$IMAGE_PULL" -eq 0 ]]; then
        echo Success Downloading image ...
        SUCCESSFULLY_PULLED_IMAGES+=("$IMAGE_NAME_AND_TAG")
    fi
done < "$VZ_IMAGE_TXT"

echo List of images that were successfully pulled ...
for image_name in "${SUCCESSFULLY_PULLED_IMAGES[@]}"
do
    IMAGE_FOUND_OR_NOT=$(grep -i "$image_name" "$VZ_IMAGE_TXT")
    if [[ $? -eq 0 ]]; then
        echo Image "$image_name" found
    fi
    else
        echo Image "$image_name" not found
    fi
done

