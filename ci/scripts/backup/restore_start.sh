#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -x
set -o pipefail

if [ -z "$BACKUP_OPENSEARCH" ] ||  [ -z "$VELERO_NAMESPACE" ] ||  [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$RESTORE_NAME" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

function cleanup() {
    kubectl delete restore.velero.io -n ${VELERO_NAMESPACE} ${RESTORE_NAME} --ignore-not-found=true
    sleep 30
}

function create_restore() {
kubectl apply -f - <<EOF
    apiVersion: velero.io/v1
        kind: Restore
        metadata:
          name: ${RESTORE_NAME}
          namespace: ${VELERO_NAMESPACE}
        spec:
          backupName: ${BACKUP_OPENSEARCH}
          includedNamespaces:
            - verrazzano-system
          labelSelector:
            matchLabels:
              verrazzano-component: opensearch
          restorePVs: false
          hooks:
            resources:
              - name: ${BACKUP_RESOURCE}
                includedNamespaces:
                  - verrazzano-system
                labelSelector:
                  matchLabels:
                    statefulset.kubernetes.io/pod-name: vmi-system-es-master-0
                postHooks:
                  - exec:
                      container: es-master
                      command:
                        - /usr/share/opensearch/bin/verrazzano-backup-hook
                        - -operation
                        - restore
                        - -velero-backup-name
                        - ${BACKUP_OPENSEARCH}
                      waitTimeout: 30m
                      execTimeout: 30m
                      onError: Fail
EOF
}

cleanup
create_restore
RETRY_COUNT=0
CHECK_DONE=true
while ${CHECK_DONE};
do
  RESPONSE=`(kubectl get restore.velero.io -n ${VELERO_NAMESPACE} ${RESTORE_NAME} -o jsonpath={.status.phase})`
  if [ "${RESPONSE}" == "InProgress" ];then
    if [ "${RETRY_COUNT}" -gt 100 ];then
       echo "Restore failed. retry count exceeded !!"
       exit 1
    fi
    echo "Restore operation in progress. Check after 30 seconds"
    sleep 30
  else
      echo "Restore progress changed to  $RESPONSE"
      CHECK_DONE=false
  fi
  RETRY_COUNT=$((RETRY_COUNT + 1))
done

if [ "${RESPONSE}" != "Completed" ]; then
    exit 1
fi

exit 0










