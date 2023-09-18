#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Function to process/parse the data from a file
process_file() {
  local IMAGE_FILE="$1"
  local IMAGENAME_ARRAY=()
  local IMAGESIZE_ARRAY=()
  local SERIALIZED_DATA=""

  while IFS= read -r line; do
    # Extract image name
    IMAGENAME=$(echo "$line" | cut -d ':' -f 1)
    # Extract image size
    IMAGESIZE=$(echo "$line" | cut -d ',' -f 2 | cut -d ',' -f 2)
    # Convert imagesize to an integer & remove spaces
    IMAGESIZE_INT=$(echo "$IMAGESIZE" | tr -d ' ')
    # Store the image name and size in separate arrays
    IMAGENAME_ARRAY+=("$IMAGENAME")
    IMAGESIZE_ARRAY+=("$IMAGESIZE_INT")
  done < "$IMAGE_FILE"

  for ((i = 0; i < ${#IMAGENAME_ARRAY[@]}; i++)); do
    SERIALIZED_DATA+="${IMAGENAME_ARRAY[$i]}:${IMAGESIZE_ARRAY[$i]}"
  done
}

oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}/image-list --file ${WORKSPACE}/image-sizes-objectstore.txt
if [ $? -ne  0 ] ; then
  echo "image-sizes.txt not found"
  oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}/image-list --file ${WORKSPACE}/image-sizes.txt
  if [ $? -eq 0 ] ; then
      exit
  fi
  echo "os object put --file image-sizes.txt failed"
  exit
fi

declare -A IMAGENAME_SIZES_FILE_OS
declare -A IMAGENAME_SIZES_FILE_GENERATED

IMAGENAME_SIZES_FILE_OS=$(process_file "${WORKSPACE}/image-sizes-objectstore.txt")
IMAGENAME_SIZES_FILE_GENERATED=$(process_file "${WORKSPACE}/image-sizes.txt")

IMAGE_SIZE_DIFF_FOUND="false"

# Exract image size & name. Populate the associative arrays for both files
while IFS=: read -r IMAGENAME IMAGESIZE; do
  IMAGENAME_SIZES_FILE_OS["$IMAGENAME"]=$IMAGESIZE
done <<< "$IMAGENAME_SIZES_FILE_OS"

while IFS=: read -r IMAGENAME IMAGESIZE; do
  IMAGENAME_SIZES_FILE_GENERATED["$IMAGENAME"]=$IMAGESIZE
done <<< "$IMAGENAME_SIZES_FILE_GENERATED"


# Check if imagenames in filenew are not in fileprev
for IMAGENAME in "${!IMAGENAME_SIZES_FILE_GENERATED[@]}"; do
  if [[ ! "${IMAGENAME_SIZES_FILE_OS[$IMAGENAME]}" ]]; then
    echo "The image-sizes.txt base file contains an image with image name: $IMAGENAME that is not in the newly generated image-sizes.txt."
  fi
done

# Compare sizes between the two files
for IMAGENAME in "${!IMAGENAME_SIZES_FILE_OS[@]}"; do
  FILE_SIZE_OS="${IMAGENAME_SIZES_FILE_OS[$IMAGENAME]}"
  FILE_SIZE_GENERATED="${IMAGENAME_SIZES_FILE_GENERATED[$IMAGENAME]}"

# Check if image size has increased by 0.1MB
  if [ -n "$FILE_SIZE_OS" ] && [ -n "$FILE_SIZE_GENERATED" ] && [ "$FILE_SIZE_GENERATED" -gt 0 ] && [ "$FILE_SIZE_GENERATED" -gt "$((FILE_SIZE_OS+100000))" ]; then
    IMAGE_SIZE_DIFF_FOUND="true"
    echo "Image size for $IMAGENAME has increased from $((FILE_SIZE_OS/1000000))MB to $((FILE_SIZE_GENERATED/1000000))MB "
  fi
done

if [ $IMAGE_SIZE_DIFF_FOUND == "true" ]; then
    exit 1
fi