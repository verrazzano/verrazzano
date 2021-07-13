#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

set -e

if [ -z "${ADMIN_KUBECONFIG}" ]; then
  echo "ADMIN_KUBECONFIG env var must be set!'"
  exit 1
fi
if [ -z "${MANAGED_CLUSTER_DIR}" ]; then
  echo "MANAGED_CLUSTER_DIR env var must be set!'"
  exit 1
fi
if [ -z "${MANAGED_CLUSTER_NAME}" ]; then
  echo "MANAGED_CLUSTER_NAME env var must be set!'"
  exit 1
fi
if [ -z "${MANAGED_KUBECONFIG}" ]; then
  echo "MANAGED_KUBECONFIG env var must be set!'"
  exit 1
fi
echo ADMIN_KUBECONFIG: ${ADMIN_KUBECONFIG}
echo MANAGED_CLUSTER_NAME: ${MANAGED_CLUSTER_NAME}
echo MANAGED_KUBECONFIG: ${MANAGED_KUBECONFIG}

# create configmap "verrazzano-admin-cluster" on admin
if ! kubectl --kubeconfig ${ADMIN_KUBECONFIG} -n verrazzano-mc get configmap verrazzano-admin-cluster; then
  export ADMIN_K8S_SERVER_ADDRESS=$(cat ${ADMIN_KUBECONFIG} | grep "server:" | awk '{ print $2 }')
  kubectl --kubeconfig ${ADMIN_KUBECONFIG} -n verrazzano-mc create configmap verrazzano-admin-cluster --from-literal=server=${ADMIN_K8S_SERVER_ADDRESS}
fi

# create managed cluster ca secret yaml on managed
CA_SECRET_FILE=${MANAGED_CLUSTER_NAME}.yaml
TLS_SECRET=$(kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n verrazzano-system get secret system-tls -o json | jq -r '.data."ca.crt"')
if [ ! -z "${TLS_SECRET%%*( )}" ] && [ "null" != "${TLS_SECRET}" ]; then
  CA_CERT=$(kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n verrazzano-system get secret system-tls -o json | jq -r '.data."ca.crt"' | base64 --decode)
fi
if [ ! -z "${CA_CERT}" ]; then
  kubectl create secret generic "ca-secret-${MANAGED_CLUSTER_NAME}" -n verrazzano-mc --from-literal=cacrt="$CA_CERT" --dry-run=client -o yaml >>${CA_SECRET_FILE}
fi

# create managed cluster ca secret on admin
kubectl --kubeconfig ${ADMIN_KUBECONFIG} apply -f ${CA_SECRET_FILE}

# check whether vz is built or not
if ! vz; then
  echo "CLI not built"
  exit 1
fi

#create VerrazzanoMangedCLuster on admin
echo "vz cluster register ${MANAGED_CLUSTER_NAME}"
vz cluster register ${MANAGED_CLUSTER_NAME} -d "VerrazzanoManagedCluster object for ${MANAGED_CLUSTER_NAME}" -c "ca-secret-${MANAGED_CLUSTER_NAME}"

# wait for VMC to be ready - that means the manifest has been created
kubectl --kubeconfig ${ADMIN_KUBECONFIG} wait --for=condition=Ready --timeout=60s vmc ${MANAGED_CLUSTER_NAME} -n verrazzano-mc
if [ $? -ne 0 ]; then
  echo "VMC ${MANAGED_CLUSTER_NAME} not ready after 60 seconds. Registration failed."
  exit 1
fi

#export manifest on admin
echo "vz cluster get-registration-manifest ${MANAGED_CLUSTER_NAME}"
vz cluster get-registration-manifest ${MANAGED_CLUSTER_NAME} >register-${MANAGED_CLUSTER_NAME}.yaml

# obtain permission-constrained version of kubeconfig to be used by managed cluster
kubectl --kubeconfig ${ADMIN_KUBECONFIG} get secret verrazzano-cluster-${MANAGED_CLUSTER_NAME}-agent -n verrazzano-mc -o jsonpath={.data.admin\-kubeconfig} | base64 --decode >${MANAGED_CLUSTER_DIR}/managed_kube_config

echo "----------BEGIN register-${MANAGED_CLUSTER_NAME}.yaml contents----------"
cat register-${MANAGED_CLUSTER_NAME}.yaml
echo "----------END register-${MANAGED_CLUSTER_NAME}.yaml contents----------"

echo "Applying register-${MANAGED_CLUSTER_NAME}.yaml"
# register using the manifest on managed
kubectl --kubeconfig ${MANAGED_KUBECONFIG} apply -f register-${MANAGED_CLUSTER_NAME}.yaml

# run verify-register test
cd ${GO_REPO_PATH}/verrazzano/tests/e2e
ginkgo -p --randomizeAllSpecs -v -keepGoing --noColor multicluster/verify-register/...

echo "vz cluster list"
vz cluster list

echo "vz cluster get ${MANAGED_CLUSTER_NAME}"
vz cluster get ${MANAGED_CLUSTER_NAME} -o yaml

# deregister managed cluster
echo "vz cluster deregister ${MANAGED_CLUSTER_NAME}"
vz cluster deregister ${MANAGED_CLUSTER_NAME}