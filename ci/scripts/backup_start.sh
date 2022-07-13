#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -x
set -o pipefail

if [ -z "$BACKUP_OPENSEARCH" ] ||  [ -z "$VELERO_NAMESPACE" ] || [ -z "$VELERO_SECRET_NAME" ]
   [ -z "$BACKUP_STORAGE" ] || [ -z "$OCI_OS_NAMESPACE" ] ; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
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

 ES_URL=$(kubectl get vz -o jsonpath={.items[].status.instance.elasticUrl})
 VZ_PASSWORD=$(kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode)
 BACKUP_ID=$(curl -ks "${ES_URL}/verrazzano-system/_search?from=0&size=1" -u verrazzano:${VZ_PASSWORD} | jq -r '.hits.hits[0]._id')

 backup=$(kubectl get backup.velero.io -n ${VELERO_NAMESPACE} ${BACKUP_OPENSEARCH} -o yaml)
 echo ${backup}

 RETRY_COUNT=0
 CHECK_DONE=true
 while ${CHECK_DONE};
  do
    RESPONSE=`(kubectl get backup.velero.io -n ${VELERO_NAMESPACE} ${BACKUP_OPENSEARCH} -o jsonpath={.status.phase})`
    if [ "${RESPONSE}" == "InProgress" ];then
      if [ "${RETRY_COUNT}" -gt 50 ];then
         echo "Backup failed. retry count exceeded !!"
         exit 1
      fi
      echo "Backup in progress. Check after 10 seconds"
      sleep 10
    else
        echo "Snapshot progress changed to  $RESPONSE"
        CHECK_DONE=false
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
  done

echo "$BACKUP_ID"

