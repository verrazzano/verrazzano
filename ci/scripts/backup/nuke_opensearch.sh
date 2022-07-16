#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -x
set -o pipefail


function nuke_os() {
  kubectl scale deploy -n verrazzano-system verrazzano-monitoring-operator --replicas=0
  kubectl delete sts -n verrazzano-system vmi-system-es-master --ignore-not-found=true
  kubectl delete deploy -n verrazzano-system \
  vmi-system-es-data-0 \
  vmi-system-es-data-1 \
  vmi-system-es-data-1 \
  vmi-system-es-ingest --ignore-not-found=true
  kubectl delete pvc -n verrazzano-system \
  elasticsearch-master-vmi-system-es-master-0 \
  elasticsearch-master-vmi-system-es-master-1 \
  elasticsearch-master-vmi-system-es-master-2 \
  vmi-system-es-data-0 \
  vmi-system-es-data-1 \
  vmi-system-es-data-1 --ignore-not-found=true
}
function check() {
    local k8sCommand=$1
    local resource=$2

    RETRY_COUNT=0
    CHECK_DONE=true
    while ${CHECK_DONE};
    do
      RESPONSE=$(${k8sCommand})
      if [ "${RESPONSE}" == "" ];then
        echo "No ${resource} found"
        CHECK_DONE=false
      else
          if [ "${RETRY_COUNT}" -gt 100 ];then
               echo "${resource} deletion failed. retry count exceeded !!"
               exit 1
          fi
          echo "${resource} deletion in progress. Check after 10 seconds"
          sleep 10
      fi
      RETRY_COUNT=$((RETRY_COUNT + 1))
    done

}

EXPECTED_MSG="No resources found in verrazzano-system namespace."
POD_CHECK_CMD="kubectl get pod -n verrazzano-system -l verrazzano-component=opensearch"
PVC_CHECK_CMD_DATA="kubectl get pvc  -n verrazzano-system -l verrazzano-component=opensearch"
PVC_CHECK_CMD_MASTER="kubectl get pvc  -n verrazzano-system -l app=system-es-master"

nuke_os
check "${POD_CHECK_CMD}" "pod"
check "${PVC_CHECK_CMD_DATA}" "pvc"
check "${PVC_CHECK_CMD_MASTER}" "pvc"

if [ $? -ne 0 ]; then
  echo "opensearch pods not cleaned up"
  exit 1
fi

echo "All opensearch related resources have been cleaned up"
exit 0

