#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -x
set -o pipefail

if [ -z "$BACKUP_OPENSEARCH" ] ||  [ -z "$VELERO_NAMESPACE" ] || [ -z "$VELERO_SECRET_NAME" ]
   [ -z "$BACKUP_STORAGE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$RESTORE_NAME" ] || [ -z "$BACKUP_ID" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

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


RETRY_COUNT=0
CHECK_DONE=true
while ${CHECK_DONE};
do
  RESPONSE=`(kubectl get restore.velero.io -n ${VELERO_NAMESPACE} ${RESTORE_NAME} -o jsonpath={.status.phase})`
  if [ "${RESPONSE}" == "InProgress" ];then
    if [ "${RETRY_COUNT}" -gt 50 ];then
       echo "Restore failed. retry count exceeded !!"
       exit 1
    fi
    echo "Restore in progress. Check after 10 seconds"
    sleep 10
  else
      echo "Restore progress changed to  $RESPONSE"
      CHECK_DONE=false
  fi
  RETRY_COUNT=$((RETRY_COUNT + 1))
done

ES_URL=$(kubectl get vz -o jsonpath={.items[].status.instance.elasticUrl})
VZ_PASSWORD=$(kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode)

REQUEST_JSON_BODY=/tmp/input.json
cat <<EOF >> ${REQUEST_JSON_BODY}
      {
        "query": {
          "terms": {
            "_id": ["${BACKUP_ID}"]
          }
        }
      }
EOF
CHECK_BACKUP_ID=$(curl -ks "${ES_URL}/verrazzano-system/_search?" -u verrazzano:${VZ_PASSWORD} -d @${REQUEST_JSON_BODY} | jq -r '.hits.hits[0]._id')

if [ ${CHECK_BACKUP_ID} == ${BACKUP_ID} ]; then
  echo "True"
else
  echo "False"
fi










