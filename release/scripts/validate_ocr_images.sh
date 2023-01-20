#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

IMAGES_FOUND_IN_OCR=("")
IMAGES_NOT_FOUND_IN_OCR=("")
OBJ_STORAGE_VZ_IMAGE_TXT=verrazzano_images.txt

printf "Logging into Crane ..."
crane auth login "$DOCKER_REPO" -u "$OCR_CREDS_USR" -p "$OCR_CREDS_PSW"

while IFS= read -r line
do  
    VZ_IMAGE_NAME=$(echo "$line")
    INSPECT_EXIT_CODE=$(crane manifest "$DOCKER_REPO/$VZ_IMAGE_NAME")
    if [[ $? -eq 0 ]]; then
        IMAGES_FOUND_IN_OCR+=("$VZ_IMAGE_NAME")
    else
        IMAGES_NOT_FOUND_IN_OCR+=("$VZ_IMAGE_NAME")
    fi
done < "$OBJ_STORAGE_VZ_IMAGE_TXT"

printf "\n\nThe following Images were NOT found in OCR ..."
for value in "${IMAGES_NOT_FOUND_IN_OCR[@]}"
do
     echo $value
done

printf "\n\nThe following Images were found in OCR ..."
for value in "${IMAGES_FOUND_IN_OCR[@]}"
do
     echo $value
done

if [[ "$FAIL_NOT_IN_OCR" = true ]]; then
      echo "Job Failed.\n A(n) image was not in OCR"
      exit 1
    fi
echo "Done."