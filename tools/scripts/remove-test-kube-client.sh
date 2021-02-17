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

set -e

kubectl delete clusterrolebinding $TEST_ROLE_BINDING
kubectl -n $TEST_NAMESPACE delete serviceaccount $TEST_SERVICEACCOUNT
kubectl delete ns $TEST_NAMESPACE
rm -rf $TEST_KUBECONFIG

set +e
