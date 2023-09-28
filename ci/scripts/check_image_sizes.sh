#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Function to process the data from a file
process_file() {
  local IMAGE_FILE="$1"
  local IMAGENAME_ARRAY=()
  local IMAGESIZE_ARRAY=()

  while IFS= read -r line; do
    # Extract image name
    IMAGENAME=$(echo "$line" | cut -d ':' -f 1)
    # Extract image size
    IMAGESIZE=$(echo "$line" | cut -d ',' -f 2 | cut -d ',' -f 2)
    # Convert IMAGESIZE to an integer & remove spaces
    IMAGESIZE_INT=$(echo "$IMAGESIZE" | tr -d ' ')
    # Store the image name and size in separate arrays
    IMAGENAME_ARRAY+=("$IMAGENAME")
    IMAGESIZE_ARRAY+=("$IMAGESIZE_INT")
  done < "$IMAGE_FILE"

  for ((i = 0; i < ${#IMAGENAME_ARRAY[@]}; i++)); do
    echo "${IMAGENAME_ARRAY[$i]}:${IMAGESIZE_ARRAY[$i]}"
  done
}

# Extract Commit ID
extract_commit_id(){
  local FILENAME="$1"
  LAST_LINE=$(tail -n 1 "$FILENAME")
  COMMIT_ID=$(echo "$LAST_LINE" | cut -d '-' -f 2)
  echo "$COMMIT_ID">${WORKSPACE}/commitID.txt
}

oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}/image-list --file ${WORKSPACE}/image-sizes-objectstore.txt
if [ $? -ne  0 ] ; then
 echo "${CLEAN_BRANCH_NAME}/image-list not found"
 oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}/image-list --file ${WORKSPACE}/image-sizes.txt
 if [ $? -eq 0 ] ; then
     exit
 fi
 echo "os object put --file image-sizes.txt failed"
 exit
fi

declare -A IMAGENAME_SIZES_FILE_OS
declare -A IMAGENAME_SIZES_FILE_GENERATED

extract_commit_id "${WORKSPACE}/image-sizes-objectstore.txt"
IMAGE_DATA_OS=$(process_file "${WORKSPACE}/image-sizes-objectstore.txt")
IMAGE_DATA_GENERATED=$(process_file "${WORKSPACE}/image-sizes.txt")
IMAGE_SIZE_DIFF_FOUND="false"
NEW_IMAGE_FOUND="false"

# Exract image size & name. Populate the associative arrays for both files
while IFS=: read -r IMAGENAME IMAGESIZE; do
  IMAGENAME_SIZES_FILE_OS["$IMAGENAME"]=$IMAGESIZE
done <<< "$IMAGE_DATA_OS"

while IFS=: read -r IMAGENAME IMAGESIZE; do
  IMAGENAME_SIZES_FILE_GENERATED["$IMAGENAME"]=$IMAGESIZE
done <<< "$IMAGE_DATA_GENERATED"

# Check if imagenames in generated file are not in object store
for IMAGENAME in "${!IMAGENAME_SIZES_FILE_GENERATED[@]}"; do
  if [[ ! "${IMAGENAME_SIZES_FILE_OS[$IMAGENAME]}" ]]; then
    NEW_IMAGE_FOUND="true"
    echo "The image-sizes.txt base file contains an image with image name: $IMAGENAME that is not in the newly generated image-sizes.txt." >> ${WORKSPACE}/newimagefound.txt
  fi
done


# Compare sizes between the two files
for IMAGENAME in "${!IMAGENAME_SIZES_FILE_OS[@]}"; do
  FILE_SIZE_OS="${IMAGENAME_SIZES_FILE_OS[$IMAGENAME]}"
  FILE_SIZE_GENERATED="${IMAGENAME_SIZES_FILE_GENERATED[$IMAGENAME]}"

# Check if image size has increased by 0.1 MB or more - test
  if [ -n "$FILE_SIZE_OS" ] && [ -n "$FILE_SIZE_GENERATED" ] && [ "$FILE_SIZE_GENERATED" -gt 0 ] && [ "$((FILE_SIZE_GENERATED+1000000))" -gt "$((FILE_SIZE_OS+IMAGE_SIZE_INCREASE_THRESHOLD))" ]; then
        IMAGE_SIZE_DIFF_FOUND="true"
    echo "Image size for $IMAGENAME has increased from $((FILE_SIZE_OS/1000000))MB to $((FILE_SIZE_GENERATED/1000000))MB " >> ${WORKSPACE}/result.txt
  fi
done
if [ $IMAGE_SIZE_DIFF_FOUND == "true" ]; then
         echo "Image size difference found: "
         cat ${WORKSPACE}/result.txt
fi

if [ $NEW_IMAGE_FOUND == "true" ]; then
         cat ${WORKSPACE}/newimagefound.txt
fi