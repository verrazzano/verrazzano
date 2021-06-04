#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

set -e

if [ -z "${ADMIN_KUBECONFIG}" ] ; then
    echo "ADMIN_KUBECONFIG env var must be set!'"
    exit 1
fi
if [ -z "${MANAGED_CLUSTER_DIR}" ] ; then
    echo "MANAGED_CLUSTER_DIR env var must be set!'"
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

# create configmap "verrazzano-admin-cluster" on admin
if ! kubectl --kubeconfig ${ADMIN_KUBECONFIG} -n verrazzano-mc get configmap verrazzano-admin-cluster ; then
  export ADMIN_K8S_SERVER_ADDRESS=$(cat ${ADMIN_KUBECONFIG} | grep "server:" | awk '{ print $2 }')
  kubectl --kubeconfig ${ADMIN_KUBECONFIG} -n verrazzano-mc create configmap verrazzano-admin-cluster --from-literal=server=${ADMIN_K8S_SERVER_ADDRESS}
fi

# create managed cluster prometheus secret yaml on managed
PROMETHEUS_SECRET_FILE=${MANAGED_CLUSTER_NAME}.yaml
TLS_SECRET=$(kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n verrazzano-system get secret system-tls -o json | jq -r '.data."ca.crt"')
if [ ! -z "${TLS_SECRET%%*( )}" ] && [ "null" != "${TLS_SECRET}" ] ; then
  CA_CERT=$(kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n verrazzano-system get secret system-tls -o json | jq -r '.data."ca.crt"' | base64 --decode)
fi
HOST=$(kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n verrazzano-system get ing vmi-system-prometheus -o jsonpath='{.spec.tls[0].hosts[0]}')
echo "prometheus:" > ${PROMETHEUS_SECRET_FILE}
echo "  host: $HOST" >> ${PROMETHEUS_SECRET_FILE}
if [ ! -z "${CA_CERT}" ] ; then
   echo "  cacrt: |" >> ${PROMETHEUS_SECRET_FILE}
   echo -e "$CA_CERT" | sed 's/^/    /' >> ${PROMETHEUS_SECRET_FILE}
fi

# create managed cluster prometheus secret on admin
kubectl --kubeconfig ${ADMIN_KUBECONFIG} -n verrazzano-mc create secret generic prometheus-${MANAGED_CLUSTER_NAME} --from-file=${PROMETHEUS_SECRET_FILE}

# create VerrazzanoManagedCluster on admin
kubectl --kubeconfig ${ADMIN_KUBECONFIG} apply -f <<EOF -
apiVersion: clusters.verrazzano.io/v1alpha1
kind: VerrazzanoManagedCluster
metadata:
  name: ${MANAGED_CLUSTER_NAME}
  namespace: verrazzano-mc
spec:
  description: "VerrazzanoManagedCluster object for prometheus-${MANAGED_CLUSTER_NAME}"
  prometheusSecret: prometheus-${MANAGED_CLUSTER_NAME}
EOF

# wait for VMC to be ready - that means the manifest has been created
echo "Creating VMC for ${MANAGED_CLUSTER_NAME}"
kubectl --kubeconfig ${ADMIN_KUBECONFIG} wait --for=condition=Ready --timeout=60s vmc ${MANAGED_CLUSTER_NAME} -n verrazzano-mc
if [ $? -ne 0 ]; then
  echo "VMC ${MANAGED_CLUSTER_NAME} not ready after 60 seconds. Registration failed."
  exit 1
fi

# export manifest on admin
kubectl --kubeconfig ${ADMIN_KUBECONFIG} get secret verrazzano-cluster-${MANAGED_CLUSTER_NAME}-manifest -n verrazzano-mc -o jsonpath={.data.yaml} | base64 --decode > register-${MANAGED_CLUSTER_NAME}.yaml

# obtain permission-constrained version of kubeconfig to be used by managed cluster
kubectl --kubeconfig ${ADMIN_KUBECONFIG} get secret verrazzano-cluster-${MANAGED_CLUSTER_NAME}-agent -n verrazzano-mc -o jsonpath={.data.admin\-kubeconfig} | base64 --decode > ${MANAGED_CLUSTER_DIR}/managed_kube_config

echo "----------BEGIN register-${MANAGED_CLUSTER_NAME}.yaml contents----------"
cat register-${MANAGED_CLUSTER_NAME}.yaml
echo "----------END register-${MANAGED_CLUSTER_NAME}.yaml contents----------"

echo "Applying register-${MANAGED_CLUSTER_NAME}.yaml"
# register using the manifest on managed
kubectl --kubeconfig ${MANAGED_KUBECONFIG} apply -f register-${MANAGED_CLUSTER_NAME}.yaml
