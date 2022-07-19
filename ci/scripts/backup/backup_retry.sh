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

function cleanup() {
    kubectl delete backup.velero.io -n ${VELERO_NAMESPACE}  ${BACKUP_OPENSEARCH} --ignore-not-found=true
    kubectl delete pod -n verrazzano-system  vmi-system-es-master-0 --grace-period=0 --force
    kubectl wait --namespace verrazzano-system --for=condition=ready pod --all --timeout=300s
}

function create_os_backup_object() {
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
}

function create_rancher_backup_object() {
kubectl apply -f - <<EOF
  apiVersion: resources.cattle.io/v1
  kind: Backup
  metadata:
    name: rancher-backup-test
  spec:
    storageLocation:
      s3:
        credentialSecretName: rancher-backup-creds
        credentialSecretNamespace: verrazzano-backup
        bucketName: aamitra-v80dev-bucket
        folder: v8odevbackup-kind
        region: us-ashburn-1
        endpoint: odsbuilddev.compat.objectstorage.us-ashburn-1.oraclecloud.com
    resourceSetName: rancher-resource-set
EOF
}

cleanup
create_os_backup_object
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
    echo "Backup operation is in progress. Check after 10 seconds"
    sleep 10
  else
      echo "Backup progress changed to  $RESPONSE"
      CHECK_DONE=false
  fi
  RETRY_COUNT=$((RETRY_COUNT + 1))
done

if [ "${RESPONSE}" != "Completed" ]; then
    exit 1
fi

exit 0
}