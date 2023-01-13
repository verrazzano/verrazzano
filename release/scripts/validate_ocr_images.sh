#!/bin/bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# We enabled the experimental Docker CLI to be able to run 'docker pull'
export DOCKER_CLI_EXPERIMENTAL=enabled

echo Running OCR image checks ...
OBJ_STORAGE_VZ_IMAGE_TXT=verrazzano_images.txt
SUCCESSFULLY_PULLED_IMAGES=("")

echo Logging into Docker ...
echo "$OCR_CREDS_PSW" | docker login "$DOCKER_REPO" -u "$OCR_CREDS_USR" --password-stdin

echo Pulling images from OCR ... 
while IFS= read -r line
do  
    VZ_IMAGE_NAME=$(echo "$line")
    docker pull "$DOCKER_REPO"/"$VZ_IMAGE_NAME"
done < "$OBJ_STORAGE_VZ_IMAGE_TXT"

# while IFS= read -r line
# do  
#     VZ_IMAGE_NAME=$(echo "$line")
#     # IMAGE_NAME_AND_TAG=$(echo "$line" | awk -F '/' '{print $2}')
#     docker image inspect "$DOCKER_REPO"/"$VZ_IMAGE_NAME"

# done < "$OBJ_STORAGE_VZ_IMAGE_TXT"
echo Exit code of image inspect . . .
EXIT_CODE=$(docker image inspect "$DOCKER_REPO"/container-registry.oracle.com/verrazzano/velero:v1.9.1-20220928065349-147272cf)
echo $?
# echo List of images that were successfully pulled ...
# for image_name in "${SUCCESSFULLY_PULLED_IMAGES[@]}"
# do
#     IMAGE_FOUND_OR_NOT=$(grep -i "$image_name" "$OBJ_STORAGE_VZ_IMAGE_TXT")
#     if [[ $? -eq 0 ]]; then
#         echo Image "$image_name" found
#     else
#         echo Image "$image_name" not found
#     fi
# done

