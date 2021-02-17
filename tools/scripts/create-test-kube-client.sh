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

if [[ -z "${TEST_SERVICEACCOUNT}" ]]; then
    echo "TEST_SERVICEACCOUNT is undefined."
    exit 1
fi

if [[ -z "${TEST_ROLE}" ]]; then
    echo "TEST_ROLE is undefined."
    exit 1
fi

if [[ -z "${TEST_ROLE_BINDING}" ]]; then
    echo "TEST_ROLE_BINDING is undefined."
    exit 1
fi

if ! role="$(kubectl get clusterrole "$TEST_ROLE" -o 'jsonpath={.metadata.name}' 2>/dev/null)"; then
  echo "clusterrole \"$TEST_ROLE\" not found."
  exit 2
fi

kubectl create ns ${TEST_NAMESPACE}
kubectl -n ${TEST_NAMESPACE} create serviceaccount ${TEST_SERVICEACCOUNT}
kubectl create clusterrolebinding ${TEST_ROLE_BINDING} --clusterrole=${TEST_ROLE} --serviceaccount=${TEST_NAMESPACE}:${TEST_SERVICEACCOUNT}


if ! secret="$(kubectl -n $TEST_NAMESPACE get serviceaccount "$TEST_SERVICEACCOUNT" -o 'jsonpath={.secrets[0].name}' 2>/dev/null)"; then
  echo "serviceaccounts \"$TEST_SERVICEACCOUNT\" not found."
  exit 2
fi

if [[ -z "$secret" ]]; then
  echo "serviceaccounts \"$TEST_SERVICEACCOUNT\" doesn't have a serviceaccount token."
  exit 2
fi

context="$(kubectl config current-context)"
cluster="$(kubectl config view -o "jsonpath={.contexts[?(@.name==\"$context\")].context.cluster}")"
server="$(kubectl config view -o "jsonpath={.clusters[?(@.name==\"$cluster\")].cluster.server}")"
ca_crt_data="$(kubectl -n $TEST_NAMESPACE get secret "$secret" -o "jsonpath={.data.ca\.crt}" | openssl enc -d -base64 -A)"
namespace="$(kubectl -n $TEST_NAMESPACE get secret "$secret" -o "jsonpath={.data.namespace}" | openssl enc -d -base64 -A)"
token="$(kubectl -n $TEST_NAMESPACE get secret "$secret" -o "jsonpath={.data.token}" | openssl enc -d -base64 -A)"

touch ${TEST_KUBECONFIG}
kubectl --kubeconfig=${TEST_KUBECONFIG} config set-credentials "$TEST_SERVICEACCOUNT" --token="$token" >/dev/null
ca_crt="$(mktemp)"; echo "$ca_crt_data" > $ca_crt
kubectl --kubeconfig=${TEST_KUBECONFIG} config set-cluster "$cluster" --server="$server" --certificate-authority="$ca_crt" --embed-certs >/dev/null
kubectl --kubeconfig=${TEST_KUBECONFIG} config set-context "$context" --cluster="$cluster" --namespace="$namespace" --user="$TEST_SERVICEACCOUNT" >/dev/null
kubectl --kubeconfig=${TEST_KUBECONFIG} config use-context "$context" >/dev/null

