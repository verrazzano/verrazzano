#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

echo "Latest release number is ..."
LIST_OF_RELEASE_NUM=($(grep -i 'images.txt' object_storage_images.json))
LATEST_RELEASE_NUMBER=$(echo ${LIST_OF_RELEASE_NUM[${#LIST_OF_RELEASE_NUM[@]}-1]} | cut -d  '"' -f 2)
echo $LATEST_RELEASE_NUMBER
oci --region ${OCI_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name $LATEST_RELEASE_NUMBER --file verrazzano_images.txt