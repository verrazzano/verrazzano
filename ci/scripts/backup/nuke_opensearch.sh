#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -x
set -o pipefail


function nuke_os() {
  kubectl scale deploy -n verrazzano-system verrazzano-monitoring-operator --replicas=0
  kubectl delete sts -n verrazzano-system vmi-system-es-master
  kubectl delete deploy -n verrazzano-system vmi-system-es-data-0
  kubectl delete deploy -n verrazzano-system vmi-system-es-data-1
  kubectl delete deploy -n verrazzano-system vmi-system-es-data-2
  kubectl delete deploy -n verrazzano-system vmi-system-es-ingest
  kubectl delete pvc -n verrazzano-system vmi-system-es-data
  kubectl delete pvc -n verrazzano-system vmi-system-es-data-1
  kubectl delete pvc -n verrazzano-system vmi-system-es-data-2
}
function check() {
    local k8sCommand=$1
    local expected_msg=$2
    local resource=$3

    RETRY_COUNT=0
    CHECK_DONE=true
    while ${CHECK_DONE};
    do
      RESPONSE=$(${k8sCommand})
      if [ "${RESPONSE}" == "${expected_msg}" ];then
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
check ${POD_CHECK_CMD} ${EXPECTED_MSG} "pod"
check ${PVC_CHECK_CMD_DATA} ${EXPECTED_MSG} "pvc"
check ${PVC_CHECK_CMD_MASTER} ${EXPECTED_MSG} "pvc"


