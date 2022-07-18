#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -o pipefail

if [ -z "$BACKUP_OPENSEARCH" ] ||  [ -z "$VELERO_NAMESPACE" ] || [ -z "$VELERO_SECRET_NAME" ]
   [ -z "$BACKUP_STORAGE" ] || [ -z "$OCI_OS_NAMESPACE" ] ; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

ES_URL=$(kubectl get vz -o jsonpath={.items[].status.instance.elasticUrl})
VZ_PASSWORD=$(kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode)
BACKUP_ID=$(curl -ks "${ES_URL}/verrazzano-system/_search?from=0&size=1" -u verrazzano:${VZ_PASSWORD} | jq -r '.hits.hits[0]._id')

if [ $? -ne 0 ]; then
  echo "Unable to fetch backup id"
  exit 1
fi

echo "$BACKUP_ID"
exit 0