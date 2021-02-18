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

if [[ -z "${TEST_ROLE_BINDING}" ]]; then
    echo "TEST_ROLE_BINDING is undefined."
    exit 1
fi

kubectl delete clusterrolebinding $TEST_ROLE_BINDING || true
kubectl -n ${TEST_NAMESPACE} delete rolebinding podreader-binding || true
kubectl -n istio-system delete rolebinding istioservicereader-binding || true
kubectl -n verrazzano-system delete rolebinding verrazzanosecretreader-binding || true
kubectl -n verrazzano-system delete rolebinding verrazzanoingressreader-binding || true
kubectl -n ${TEST_NAMESPACE} delete role podreader || true
kubectl -n istio-system delete role istioservicereader || true
kubectl -n verrazzano-system delete role verrazzanosecretreader || true
kubectl -n verrazzano-system delete role verrazzanoingressreader || true
kubectl -n $TEST_NAMESPACE delete serviceaccount $TEST_SERVICEACCOUNT || true
kubectl delete ns $TEST_NAMESPACE || true
rm -rf $TEST_KUBECONFIG || true


