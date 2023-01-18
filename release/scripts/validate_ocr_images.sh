#!/bin/bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# We enabled the experimental Docker CLI to be able to run 'docker pull'
export DOCKER_CLI_EXPERIMENTAL=enabled
IMAGES_FOUND_IN_OCR=("")
IMAGES_NOT_FOUND_IN_OCR=("")

echo "Running OCR image checks ..."
OBJ_STORAGE_VZ_IMAGE_TXT=verrazzano_images.txt

echo "Logging into Docker ..."
echo "$OCR_CREDS_PSW" | docker login "$DOCKER_REPO" -u "$OCR_CREDS_USR" --password-stdin

echo "Logging into Skopeo ..."
docker run quay.io/skopeo/stable:latest login "$DOCKER_REPO"

echo "Logging into Crane ..."
crane auth login "$DOCKER_REPO" -u "$OCR_CREDS_USR" -p "$OCR_CREDS_PSW"

# echo "Pulling images from OCR ..." 
# while IFS= read -r line
# do  
#     VZ_IMAGE_NAME=$(echo "$line")
#     docker pull "$DOCKER_REPO"/"$VZ_IMAGE_NAME"
# done < "$OBJ_STORAGE_VZ_IMAGE_TXT"

# printf "\n\nThe following Images were NOT found in OCR ..."
# while IFS= read -r line
# do  
#     VZ_IMAGE_NAME=$(echo "$line")
#     INSPECT_EXIT_CODE=$(docker image inspect "$DOCKER_REPO"/"$VZ_IMAGE_NAME")
#     if [[ $? -eq 1 ]]; then
#         echo "$VZ_IMAGE_NAME" NOT found
#     else
#         echo "$VZ_IMAGE_NAME" was found
#     fi
# done < "$OBJ_STORAGE_VZ_IMAGE_TXT"

while IFS= read -r line
do  
    VZ_IMAGE_NAME=$(echo "$line")
    # INSPECT_EXIT_CODE=$(docker run quay.io/skopeo/stable:latest inspect docker://"$DOCKER_REPO"/"$VZ_IMAGE_NAME")
    # INSPECT_EXIT_CODE=$(docker run quay.io/skopeo/stable:latest inspect --username="$OCR_CREDS_USR" --password="$OCR_CREDS_PSW" docker://"$DOCKER_REPO"/"$VZ_IMAGE_NAME")
    INSPECT_EXIT_CODE=$(crane manifest "$DOCKER_REPO/$VZ_IMAGE_NAME")
    if [[ $? -eq 0 ]]; then
        IMAGES_FOUND_IN_OCR+=("$VZ_IMAGE_NAME")
    else
        IMAGES_NOT_FOUND_IN_OCR+=("$VZ_IMAGE_NAME")
    fi
done < "$OBJ_STORAGE_VZ_IMAGE_TXT"

# Print Images NOT found in OCR
printf "\n\nThe following Images were NOT found in OCR ..."
for value in "${IMAGES_NOT_FOUND_IN_OCR[@]}"
do
     echo $value
done

# Print Images found in OCR
printf "\n\nThe following Images were found in OCR ..."
for value in "${IMAGES_FOUND_IN_OCR[@]}"
do
     echo $value
done

# docker run --rm quay.io/skopeo/stable:latest inspect --authfile "$AUTHFILE"/auth.json docker://"$DOCKER_REPO"/verrazzano/example-bobbys-coherence:1.0.0-1-20210728181814-eb1e622
# docker run --rm quay.io/skopeo/stable:latest inspect docker://"$DOCKER_REPO"/verrazzano/example-bobbys-coherence:1.0.0-1-20210728181814-eb1e622
# docker run --rm quay.io/skopeo/stable:latest inspect docker://"$DOCKER_REPO"/verrazzano/velero:v1.9.1-20220928065349-147272cf


echo "Done."