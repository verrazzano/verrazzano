#!/bin/bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

echo Running OCR image checks ...
VZ_IMAGE_TXT=verrazzano_images.txt
LOCAL_IMAGES_TXT=/Users/abrherna/Downloads/release-1.4_verrazzano_1.4.3-images.txt

IMAGES=$(cat "$LOCAL_IMAGES_TXT")

for image in "$IMAGES"; do
    echo "$image" | awk -F '[/]' '{print $2}'
done
