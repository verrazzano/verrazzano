#!/bin/bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

echo Running OCR image checks ...
VZ_IMAGE_TXT=verrazzano_images.txt
IMAGES=$(cat "$VZ_IMAGE_TXT")

for image in "$IMAGES";
do
    echo "$image"
done

docker manifest inspect alertmanager:v0.24.0-20221206192620-fb73d30f