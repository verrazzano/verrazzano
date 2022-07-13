#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -x
set -o pipefail

if [ -z "$OCI_OS_ACCESS_KEY" ] || [ -z "$OCI_OS_ACCESS_SECRET_KEY" ] || [ -z "$VELERO_NAMESPACE" ] || [ -z "$VELERO_SECRET_NAME" ]
   [ -z "$BACKUP_STORAGE" ] || [ -z "$OCI_OS_BUCKET_NAME" ] || [ -z "$OCI_OS_NAMESPACE" ] ; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

SECRETS_FILE=/tmp/os-creds.ini

cat <<EOF >> ${SECRETS_FILE}
   [default]
   aws_access_key_id=${OCI_OS_ACCESS_KEY}
   aws_secret_access_key=${OCI_OS_ACCESS_SECRET_KEY}
EOF

echo "List secrets under '${VELERO_NAMESPACE}' namespace"
kubectl get secrets -n ${VELERO_NAMESPACE}

kubectl create secret generic -n ${VELERO_NAMESPACE} ${VELERO_SECRET_NAME} --from-file=${SECRETS_FILE}

echo "List secrets under '${VELERO_NAMESPACE}' namespace after secret is created"
kubectl get secrets -n ${VELERO_NAMESPACE}

rm -rf ${SECRETS_FILE}

kubectl apply -f - <<EOF
    apiVersion: velero.io/v1
    kind: BackupStorageLocation
    metadata:
      name: ${BACKUP_STORAGE}
      namespace: ${VELERO_NAMESPACE}
    spec:
      provider: aws
      objectStorage:
        bucket: ${OCI_OS_BUCKET_NAME}
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
BSL=$(kubectl get bsl ${BACKUP_STORAGE} -n ${VELERO_NAMESPACE} --no-headers -o custom-columns=":metadata.name")

# debug
kubectl get bsl ${BACKUP_STORAGE} -n ${VELERO_NAMESPACE} -o yaml

if [ $BSL == ${BACKUP_STORAGE} ]; then
  RESULT=Success
fi
echo "$RESULT"
