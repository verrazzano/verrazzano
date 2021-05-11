#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ ! -f "$1" ]; then
  echo "You must specify the images list CSV file as input"
  exit 1
fi

if [ ! -d "$2" ]; then
  echo "Please specify temp directory"
  exit 1
fi

if [ -f "$3" ]; then
  echo "Output file already exists, please specify a new filename"
  exit 1
fi

if [ -d "$3" ]; then
  echo "Please specify a new filename not a directory"
  exit 1
fi

# The CSV file is the output of the cluster-dump, which is based on the images found on the cluster nodes.
# There currently is only one column from this format which is the entire image to pull
while read csv_line; do
  image=$(echo "$csv_line" | cut -d, -f"1")
  image_alt=$(echo "$csv_line" | cut -d, -f"2")
  if [ -z "$image" ]; then
    echo "Skipping empty line"
    continue
  fi

  # Skip comments
  if [[ "$image" == "#"* ]]; then
    echo "Skipping comment"
    continue
  fi

  # Excluding images pulled from region specific OCIR locations
  if [[ "$image" == *"-1.ocir.io"* ]]; then
    echo "This was a region specific image, skipping: $image"
    continue
  fi

  # FIXME: Some of the CSV entries from the cluster-dump have a second row and that is the image without the hash, need
  # to get that output consistent so we can use it here (ie: for our purposes image without the hash in the first column
  # is what we need and there are maybe a few that don't fit that)
  tarname=$(echo "$image.tar" | sed -e 's;/;_;g' -e 's/:/-/g')

  docker pull $image
  docker save -o $2/${tarname} ${image}
done <$1

tar -czf ${3} ${2}/*.tar
