#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

set -e

if [ -z "${ADMIN_KUBECONFIG}" ] ; then
    echo "ADMIN_KUBECONFIG env var must be set!'"
    exit 1
fi
if [ -z "${MANAGED_CLUSTER_NAME}" ] ; then
    echo "MANAGED_CLUSTER_NAME env var must be set!'"
    exit 1
fi
if [ -z "${MANAGED_KUBECONFIG}" ] ; then
    echo "MANAGED_KUBECONFIG env var must be set!'"
    exit 1
fi

echo ADMIN_KUBECONFIG: ${ADMIN_KUBECONFIG}
echo MANAGED_CLUSTER_NAME: ${MANAGED_CLUSTER_NAME}
echo MANAGED_KUBECONFIG: ${MANAGED_KUBECONFIG}

function deleteRancherCluster() {
      # get the admin user token from the Rancher API
      RANCHER_URL=$(kubectl --kubeconfig ${ADMIN_KUBECONFIG} get vz -o jsonpath='{.items[0].status.instance.rancherUrl}')
      echo "RANCHER_URL: ${RANCHER_URL}"
      RANCHER_ADMIN_PASS=$(kubectl --kubeconfig ${ADMIN_KUBECONFIG} get secret -n cattle-system rancher-admin-secret -o jsonpath={.data.password} | base64 --decode)
      RANCHER_TOKEN=$(curl -s -k -X POST -H 'Content-Type: application/json' "${RANCHER_URL}/v3-public/localProviders/local?action=login"  -d "{\"username\":\"admin\", \"password\":\"${RANCHER_ADMIN_PASS}\"}"| jq -r ".token")
      if [ -z "${RANCHER_TOKEN}" ] ; then
        echo "Rancher token for admin user not found"
        exit 1
      fi

      # Use the token to retrieve Rancher cluster id and delete the cluster from Rancher
      RANCHER_CLUSTER_ID=$(curl -s -k -X GET -H "Authorization: Bearer ${RANCHER_TOKEN}" "${RANCHER_URL}/v3/clusters?name=${MANAGED_CLUSTER_NAME}" | jq -r '.data[0].id')
      echo "deleting RANCHER_CLUSTER_ID: ${RANCHER_CLUSTER_ID}"
      curl -s -k -X DELETE -H "Authorization: Bearer ${RANCHER_TOKEN}" "${RANCHER_URL}/v3/clusters/${RANCHER_CLUSTER_ID}"
}

# Delete the Rancher cluster explicitly here until VMC delete auto-triggers Rancher cluster delete (VZ-6451).
echo "Deleting Rancher cluster"
deleteRancherCluster
echo "Deleting VMC on admin cluster ${MANAGED_CLUSTER_NAME}"
kubectl --kubeconfig ${ADMIN_KUBECONFIG} -n verrazzano-mc delete vmc ${MANAGED_CLUSTER_NAME}
echo "Deleting cluster registration secret on managed cluster"
kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n verrazzano-system delete secret verrazzano-cluster-registration
