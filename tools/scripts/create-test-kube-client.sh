#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

if [[ -z "${TEST_KUBECONFIG}" ]]; then
    echo "TEST_KUBECONFIG is undefined."
    exit 1
fi

if [[ -z "${TEST_NAMESPACE}" ]]; then
    echo "TEST_NAMESPACE is undefined."
    exit 1
fi

if [[ -z "${TEST_ID}" ]]; then
    echo "TEST_ID is undefined."
    exit 1
fi

if [[ -z "${PROJECT_ADMIN_ROLE}" ]]; then
    echo "PROJECT_ADMIN_ROLE is undefined."
    exit 1
fi

if [[ -z "${TEST_ROLE}" ]]; then
    echo "TEST_ROLE is undefined."
    exit 1
fi

if ! role="$(kubectl get clusterrole "$TEST_ROLE" -o 'jsonpath={.metadata.name}' 2>/dev/null)"; then
  echo "clusterrole \"$TEST_ROLE\" not found."
  exit 2
fi

if ! role="$(kubectl get clusterrole "$PROJECT_ADMIN_ROLE" -o 'jsonpath={.metadata.name}' 2>/dev/null)"; then
  echo "clusterrole \"$PROJECT_ADMIN_ROLE\" not found."
  exit 2
fi

kubectl create ns ${TEST_NAMESPACE}
kubectl -n ${TEST_NAMESPACE} create serviceaccount ${TEST_ID}-sa
kubectl -n verrazzano-system create rolebinding ${TEST_ID}-${TEST_ROLE}-binding --clusterrole=${TEST_ROLE} --serviceaccount=${TEST_NAMESPACE}:${TEST_ID}-sa
kubectl -n istio-system create rolebinding ${TEST_ID}-${TEST_ROLE}-binding --clusterrole=${TEST_ROLE} --serviceaccount=${TEST_NAMESPACE}:${TEST_ID}-sa
kubectl -n cert-manager create rolebinding ${TEST_ID}-${TEST_ROLE}-binding --clusterrole=${TEST_ROLE} --serviceaccount=${TEST_NAMESPACE}:${TEST_ID}-sa
kubectl -n cattle-system create rolebinding ${TEST_ID}-${TEST_ROLE}-binding --clusterrole=${TEST_ROLE} --serviceaccount=${TEST_NAMESPACE}:${TEST_ID}-sa
kubectl -n ingress-nginx create rolebinding ${TEST_ID}-${TEST_ROLE}-binding --clusterrole=${TEST_ROLE} --serviceaccount=${TEST_NAMESPACE}:${TEST_ID}-sa
kubectl -n keycloak create rolebinding ${TEST_ID}-${TEST_ROLE}-binding --clusterrole=${TEST_ROLE} --serviceaccount=${TEST_NAMESPACE}:${TEST_ID}-sa
kubectl -n ${TEST_NAMESPACE} create rolebinding ${TEST_ID}-${TEST_ROLE}-binding --clusterrole=${TEST_ROLE} --serviceaccount=${TEST_NAMESPACE}:${TEST_ID}-sa
kubectl -n ${TEST_NAMESPACE} create rolebinding ${TEST_ID}-${PROJECT_ADMIN_ROLE}-binding --clusterrole=${PROJECT_ADMIN_ROLE} --serviceaccount=${TEST_NAMESPACE}:${TEST_ID}-sa
# In k8s 1.24 and later, secret is not created for service account. Create a service account token secret and get the
#	token from the same.
secret=${TEST_ID}-sa-token
kubectl -n ${TEST_NAMESPACE} apply -f <<EOF -
  apiVersion: v1
  kind: Secret
  metadata:
    name: ${secret}
    annotations:
      kubernetes.io/service-account.name: ${TEST_ID}-sa
  type: kubernetes.io/service-account-token
EOF

echo "Creating test kubeconfig at ${TEST_KUBECONFIG}"
export OLD_KUBECONFIG=${KUBECONFIG}
cp ${KUBECONFIG} /tmp/${TEST_ID}-kubeconfig
export KUBECONFIG=/tmp/${TEST_ID}-kubeconfig
context="$(kubectl config current-context)"
cluster="$(kubectl config view -o "jsonpath={.contexts[?(@.name==\"$context\")].context.cluster}")"
server="$(kubectl config view -o "jsonpath={.clusters[?(@.name==\"$cluster\")].cluster.server}")"
ca_crt_data="$(kubectl -n $TEST_NAMESPACE get secret "$secret" -o "jsonpath={.data.ca\.crt}" | openssl enc -d -base64 -A)"
namespace="$(kubectl -n $TEST_NAMESPACE get secret "$secret" -o "jsonpath={.data.namespace}" | openssl enc -d -base64 -A)"
token="$(kubectl -n $TEST_NAMESPACE get secret "$secret" -o "jsonpath={.data.token}" | openssl enc -d -base64 -A)"

touch ${TEST_KUBECONFIG}
kubectl --kubeconfig=${TEST_KUBECONFIG} config set-credentials "${TEST_ID}-sa" --token="$token" >/dev/null
ca_crt="$(mktemp)"; echo "$ca_crt_data" > $ca_crt
kubectl --kubeconfig=${TEST_KUBECONFIG} config set-cluster "$cluster" --server="$server" --certificate-authority="$ca_crt" --embed-certs >/dev/null
kubectl --kubeconfig=${TEST_KUBECONFIG} config set-context "$context" --cluster="$cluster" --namespace="$namespace" --user="${TEST_ID}-sa" >/dev/null
kubectl --kubeconfig=${TEST_KUBECONFIG} config use-context "$context" >/dev/null
echo "Test kubeconfig ${TEST_KUBECONFIG} created."
rm -rf /tmp/${TEST_ID}-kubeconfig
export KUBECONFIG=$OLD_KUBECONFIG