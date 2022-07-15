#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -o pipefail

if [ -z "$OCI_OS_ACCESS_KEY" ] || [ -z "$OCI_OS_ACCESS_SECRET_KEY" ] || [ -z "$VELERO_NAMESPACE" ] || [ -z "$VELERO_SECRET_NAME" ]
   [ -z "$BACKUP_STORAGE" ] || [ -z "$OCI_OS_BUCKET_NAME" ] || [ -z "$OCI_OS_NAMESPACE" ] ; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

function waitForOpenSearch() {
  ES_URL=$(kubectl get vz -o jsonpath={.items[].status.instance.elasticUrl})
  VZ_PASSWORD=$(kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode)

  RETRY_COUNT=0
  CHECK_DONE=true
  while ${CHECK_DONE};
  do
    RESPONSE=$(curl -ks "${ES_URL}/_cluster/health" -u verrazzano:${VZ_PASSWORD} | jq -r .status)
    if [ "${RESPONSE}" == "green" ];then
      echo "Opensearch is healthy"
      CHECK_DONE=false
    else
        if [ "${RETRY_COUNT}" -gt 100 ];then
             echo "Opensearch health check failure. retry count exceeded !!"
             exit 1
        fi
        echo "Opensearch health is '${RESPONSE}'. Check after 10 seconds"
        sleep 10
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
  done

}

waitForOpenSearch

SECRETS_FILE=/tmp/os-creds.ini

cat <<EOF >> ${SECRETS_FILE}
   [default]
   aws_access_key_id=${OCI_OS_ACCESS_KEY}
   aws_secret_access_key=${OCI_OS_ACCESS_SECRET_KEY}
EOF

kubectl create secret generic -n ${VELERO_NAMESPACE} ${VELERO_SECRET_NAME} --from-file=cloud=${SECRETS_FILE}
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

if [ $? -ne 0 ]; then
  echo "Backup storage location creation failure"
  exit 1
fi

kubectl apply -f - <<EOF
    apiVersion: velero.io/v1
    kind: Backup
    metadata:
      name: ${BACKUP_OPENSEARCH}
      namespace: ${VELERO_NAMESPACE}
    spec:
      includedNamespaces:
        - verrazzano-system
      labelSelector:
        matchLabels:
          verrazzano-component: opensearch
      defaultVolumesToRestic: false
      storageLocation: ${BACKUP_STORAGE}
      hooks:
        resources:
          -
            name: ${BACKUP_RESOURCE}
            includedNamespaces:
              - verrazzano-system
            labelSelector:
              matchLabels:
                statefulset.kubernetes.io/pod-name: vmi-system-es-master-0
            post:
              -
                exec:
                  container: es-master
                  command:
                    - /usr/share/opensearch/bin/verrazzano-backup-hook
                    - -operation
                    - backup
                    - -velero-backup-name
                    - ${BACKUP_OPENSEARCH}
                  onError: Fail
                  timeout: 10m
EOF

if [ $? -ne 0 ]; then
  echo "Backup object creation failure"
  exit 1
fi

exit 0