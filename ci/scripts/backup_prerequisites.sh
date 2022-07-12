#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -o pipefail

if [ -z "$OCI_OS_ACCESS_KEY" ] || [ -z "$OCI_OS_ACCESS_SECRET_KEY" ] || [ -z "$VELERO_NAMESPACE" ] || [ -z "$VELERO_SECRET_NAME" ]
   [ -z "$BACKUP_STORAGE" ] || [ -z "$VELERO_BUCKET_NAME" ] || [ -z "$OCI_OS_NAMESPACE" ] ; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

cat <<EOF >> velero-creds.ini
   [default]
   aws_access_key_id=${OCI_OS_ACCESS_KEY_PSW}
   aws_secret_access_key=${OCI_OS_ACCESS_SECRET_KEY_PSW}
EOF

kubectl create secret generic -n ${VELERO_NAMESPACE} ${VELERO_SECRET_NAME} --from-file=cloud=velero-creds.ini

kubectl apply -f - <<EOF
    apiVersion: velero.io/v1
    kind: BackupStorageLocation
    metadata:
      name: ${BACKUP_STORAGE}
      namespace: ${VELERO_NAMESPACE}
    spec:
      provider: aws
      objectStorage:
        bucket: ${VELERO_BUCKET_NAME}
        prefix: opensearch
      credential:
        name: ${VELERO_SECRET_NAME}
        key: cloud
      config:
        region: us-phoenix-1
        s3ForcePathStyle: "true"
        s3Url: https://${OCI_OS_NAMESPACE}.compat.objectstorage.us-phoenix-1.oraclecloud.com
EOF

RESULT=Failed
BSL=$(kubectl get bsl ${BACKUP_STORAGE} -n ${VELERO_NAMESPACE})
if [ $BSL != "" ]; then
  RESULT=Sucess
fi
echo "$RESULT"
