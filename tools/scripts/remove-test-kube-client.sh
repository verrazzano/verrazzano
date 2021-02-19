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

kubectl -n ${TEST_NAMESPACE} delete rolebinding ${TEST_ID}-${TEST_ROLE}-binding || true
kubectl -n verrazzano-system delete rolebinding ${TEST_ID}-${TEST_ROLE}-binding || true
kubectl -n istio-system delete rolebinding ${TEST_ID}-${TEST_ROLE}-binding || true
kubectl -n ${TEST_NAMESPACE} delete rolebinding ${TEST_ID}-${PROJECT_ADMIN_ROLE}-binding || true
kubectl -n ${TEST_NAMESPACE} delete serviceaccount $TEST_ID-sa || true
kubectl delete ns $TEST_NAMESPACE || true
rm -rf $TEST_KUBECONFIG || true
echo "Test kubeconfig ${TEST_KUBECONFIG} deleted."


