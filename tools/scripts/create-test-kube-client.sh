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
kubectl -n ${TEST_NAMESPACE} create rolebinding ${TEST_ID}-${TEST_ROLE}-binding --clusterrole=${TEST_ROLE} --serviceaccount=${TEST_NAMESPACE}:${TEST_ID}-sa
kubectl -n ${TEST_NAMESPACE} create rolebinding ${TEST_ID}-${PROJECT_ADMIN_ROLE}-binding --clusterrole=${PROJECT_ADMIN_ROLE} --serviceaccount=${TEST_NAMESPACE}:${TEST_ID}-sa

if ! secret="$(kubectl -n $TEST_NAMESPACE get serviceaccount "${TEST_ID}-sa" -o 'jsonpath={.secrets[0].name}' 2>/dev/null)"; then
  echo "serviceaccounts \"${TEST_ID}-sa\" not found."
  exit 2
fi

if [[ -z "$secret" ]]; then
  echo "serviceaccounts \"${TEST_ID}-sa\" doesn't have a serviceaccount token."
  exit 2
fi

mkdir -p /tmp/${TEST_ID}-kubeconfig
cp ${KUBECONFIG} /tmp/${TEST_ID}-kubeconfig/kubeconfig
context="$(KUBECONFIG=/tmp/$TEST_ID-kubeconfig/kubeconfig kubectl config current-context)"
cluster="$(KUBECONFIG=/tmp/$TEST_ID-kubeconfig/kubeconfig kubectl config view -o "jsonpath={.contexts[?(@.name==\"$context\")].context.cluster}")"
server="$(KUBECONFIG=/tmp/$TEST_ID-kubeconfig/kubeconfig kubectl config view -o "jsonpath={.clusters[?(@.name==\"$cluster\")].cluster.server}")"
rm -rf /tmp/${TEST_ID}-kubeconfig

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
