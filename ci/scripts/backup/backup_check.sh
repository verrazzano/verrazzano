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

RETRY_COUNT=0
CHECK_DONE=true
while ${CHECK_DONE};
do
  RESPONSE=`(kubectl get backup.velero.io -n ${VELERO_NAMESPACE} ${BACKUP_OPENSEARCH} -o jsonpath={.status.phase})`
  if [ "${RESPONSE}" == "InProgress" ];then
    if [ "${RETRY_COUNT}" -gt 100 ];then
       echo "Backup failed. retry count exceeded !!"
       exit 1
    fi
    #echo "Backup operation is in progress. Check after 10 seconds"
    sleep 10
  else
      #echo "Backup progress changed to  $RESPONSE"
      CHECK_DONE=false
  fi
  RETRY_COUNT=$((RETRY_COUNT + 1))
done

echo "$BACKUP_ID"
exit 0