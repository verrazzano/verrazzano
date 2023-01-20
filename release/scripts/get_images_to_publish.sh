#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Try to download specified Object store file from, default value is NONE
printf "\nTrying to download image.txt file for $OBJECT_STORE_FILE"
oci --region ${OCI_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${OBJECT_STORE_FILE} --file verrazzano_images.txt

if [[ $? -gt 0 ]]; then
# This "object list" command grabs the list of all objects in the OCI_OS_NAMESPACE that is a release object. Then trys to download the most up-to-date images.txt file
# This only runs if the "object get" for the specified version fails or does not exist
    printf "\nTrying to downloading release-* object list ..."
    oci os object list --region us-phoenix-1 --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --all --stream-output --prefix release- > object_storage_images.json

    LIST_OF_RELEASE_NUM=($(grep -i 'images.txt' object_storage_images.json))
    LATEST_RELEASE_NUMBER=$(echo ${LIST_OF_RELEASE_NUM[${#LIST_OF_RELEASE_NUM[@]}-1]} | cut -d  '"' -f 2)
    printf "\nLatest release-* object image.txt file is ... $LATEST_RELEASE_NUMBER"

    printf "\nTrying to download image.txt file for $LATEST_RELEASE_NUMBER"
    oci --region ${OCI_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name $LATEST_RELEASE_NUMBER --file verrazzano_images.txt
fi