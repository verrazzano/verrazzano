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
if [ ! -z "${TLS_SECRET%%*( )}" ] ; then
  CA_CERT=$(kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n verrazzano-system get secret system-tls -o json | jq -r '.data."ca.crt"' | base64 --decode)
fi
AUTH_PASSWORD=$(kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n verrazzano-system get secret verrazzano -o jsonpath='{.data.password}' | base64 --decode)
HOST=$(kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n verrazzano-system get ing vmi-system-prometheus -o jsonpath='{.spec.tls[0].hosts[0]}')
echo "prometheus:" > ${PROMETHEUS_SECRET_FILE}
echo "  authpasswd: $AUTH_PASSWORD" >> ${PROMETHEUS_SECRET_FILE}
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

# wait for manifest to be created
retries=0
until kubectl --kubeconfig ${ADMIN_KUBECONFIG} -n verrazzano-mc get secret | grep verrazzano-cluster-${MANAGED_CLUSTER_NAME}-manifest; do
  retries=$(($retries+1))
  sleep 1
  if [ "$retries" -ge 10 ] ; then
    break
  fi
done

# export manifest on admin
kubectl --kubeconfig ${ADMIN_KUBECONFIG} get secret verrazzano-cluster-${MANAGED_CLUSTER_NAME}-manifest -n verrazzano-mc -o jsonpath={.data.yaml} | base64 --decode > register-${MANAGED_CLUSTER_NAME}.yaml

# register using the manifest on managed
kubectl --kubeconfig ${MANAGED_KUBECONFIG} apply -f register-${MANAGED_CLUSTER_NAME}.yaml

# the following is not related to registering managed cluster, but to working around xip.io resolution problem
set +e
retries=0
until [ "$retries" -ge 30 ]
do
  kubectl --kubeconfig ${ADMIN_KUBECONFIG} -n verrazzano-system get ing vmi-system-es-ingest && break
  retries=$((retries+1))
  sleep 5
done
ES_HOST=$(kubectl --kubeconfig ${ADMIN_KUBECONFIG} -n verrazzano-system get ing vmi-system-es-ingest -o jsonpath='{.spec.rules[0].host}')
if [[ "${ES_HOST}" == *.xip.io ]]; then
  # wait until secret verrazzano-cluster-registration is present
  retries=0
  until [ "$retries" -ge 30 ]
  do
    kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n verrazzano-system get secret verrazzano-cluster-registration && break
    retries=$((retries+1))
    sleep 5
  done
  REGISTRATION_VERSION=$(kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n verrazzano-system get secret verrazzano-cluster-registration -o jsonpath='{.metadata.resourceVersion}')
  # wait until verrazzano-operator deployment has the same REGISTRATION_SECRET_VERSION as verrazzano-cluster-registration
  retries=0
  until [ "$retries" -ge 30 ]
  do
    ENV_VERSION=$(kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n verrazzano-system get deployment verrazzano-operator -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="REGISTRATION_SECRET_VERSION")].value}')
    if [[ ${REGISTRATION_VERSION} = ${ENV_VERSION} ]]; then
      break
    fi
    retries=$((retries+1))
    sleep 5
  done
  # patch /etc/hosts in the managed cluster beats pods by setting hostAliases
  kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n verrazzano-system rollout status deploy verrazzano-operator
  kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n logging rollout status daemonset filebeat
  kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n logging rollout status daemonset journalbeat
  ES_IP=$(kubectl --kubeconfig ${ADMIN_KUBECONFIG} -n verrazzano-system get ing vmi-system-es-ingest -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
  patch_data='{"spec":{"template":{"spec":{"hostAliases":[{"hostnames":["'"${ES_HOST}"'"],"ip":"'"${ES_IP}"'"}]}}}}'
  kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n logging patch daemonset filebeat --patch ${patch_data}
  kubectl --kubeconfig ${MANAGED_KUBECONFIG} -n logging patch daemonset journalbeat --patch ${patch_data}
fi
