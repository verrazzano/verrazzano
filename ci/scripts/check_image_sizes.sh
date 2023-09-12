#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#


#diff ${WORKSPACE}/image-sizes-objectstore.txt ${WORKSPACE}/image-sizes.txt > returnvalue.txt
#if [ $? -eq 0 ] ; then
 # rm returnvalue.txt
#fi

# Function to process the data from a file
process_file() {
  local filename="$1"
  local imagename_array=()
  local imagesize_array=()

  while IFS= read -r line; do
    # Extract image name
    imagename=$(echo "$line" | cut -d ':' -f 1)
    # Extract image size
    imagesize=$(echo "$line" | cut -d ',' -f 2 | cut -d ',' -f 2)
    # Convert imagesize to an integer & remove spaces
    imagesize_int=$(echo "$imagesize" | tr -d ' ')
    # Store the image name and size in separate arrays
    imagename_array+=("$imagename")
    imagesize_array+=("$imagesize_int")
  done < "$filename"

  for ((i = 0; i < ${#imagename_array[@]}; i++)); do
    echo "${imagename_array[$i]}:${imagesize_array[$i]}"
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

declare -A imagename_sizes_fileprev
declare -A imagename_sizes_filenew

image_data_fileprev=$(process_file "${WORKSPACE}/image-sizes-objectstore.txt")
image_data_filenew=$(process_file "${WORKSPACE}/image-sizes.txt")

IMAGE_SIZE_DIFF_FOUND="false"

# Exract image size & name. Populate the associative arrays for both files
while IFS=: read -r imagename imagesize; do
  imagename_sizes_fileprev["$imagename"]=$imagesize
done <<< "$image_data_fileprev"

while IFS=: read -r imagename imagesize; do
  imagename_sizes_filenew["$imagename"]=$imagesize
done <<< "$image_data_filenew"

# Print the contents of the associative arrays for the files
#echo "Image sizes for ${WORKSPACE}/image-sizes-objectstore.txt:"
#for imagename in "${!imagename_sizes_fileprev[@]}"; do
#  echo "$imagename: ${imagename_sizes_fileprev[$imagename]}"
#done

#echo "Image sizes for ${WORKSPACE}/image-sizes.txt:"
#for imagename in "${!imagename_sizes_filenew[@]}"; do
#  echo "$imagename: ${imagename_sizes_filenew[$imagename]}"
#done

# Check if imagenames in filenew are not in fileprev
for imagename in "${!imagename_sizes_filenew[@]}"; do
  if [[ ! "${imagename_sizes_fileprev[$imagename]}" ]]; then
    echo "The image-sizes.txt base file contains an image with image name: $imagename that is not in the newly generated image-sizes.txt."
  fi
done

# Compare sizes between the two files
for imagename in "${!imagename_sizes_fileprev[@]}"; do
  size1="${imagename_sizes_fileprev[$imagename]}"
  size2="${imagename_sizes_filenew[$imagename]}"

# Check if image size has increased by 0.1MB
  if [ -n "$size1" ] && [ -n "$size2" ] && [ "$size2" -gt 0 ] && [ "$((size2+100000))" -gt "$size1" ]; then
    IMAGE_SIZE_DIFF_FOUND="true"
    echo "Image size for $imagename has increased from $((size1/1000000))MB to $((size2/1000000))MB "
  fi
done

if [ $IMAGE_SIZE_DIFF_FOUND == "true" ]; then
    exit 1
fi
