#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Try to download specified Object store file from, default value is NONE
echo "Trying to download image.txt file for $OBJECT_STORE_FILE"
oci --region ${OCI_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${OBJECT_STORE_FILE} --file verrazzano_images.txt

if [[ $? -gt 0 ]]; then
    echo "Trying to downloading release-* object list ..."
    oci os object list --region us-phoenix-1 --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --all --stream-output --prefix release- > object_storage_images.json
    
    echo "Latest release-* object image.txt file is ..."
    LIST_OF_RELEASE_NUM=($(grep -i 'images.txt' object_storage_images.json))
    LATEST_RELEASE_NUMBER=$(echo ${LIST_OF_RELEASE_NUM[${#LIST_OF_RELEASE_NUM[@]}-1]} | cut -d  '"' -f 2)
    echo $LATEST_RELEASE_NUMBER

    echo "Trying to download image.txt file for $LATEST_RELEASE_NUMBER"
    oci --region ${OCI_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name $LATEST_RELEASE_NUMBER --file verrazzano_images.txt
fi