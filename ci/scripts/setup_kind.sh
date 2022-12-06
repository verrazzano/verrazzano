#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# $1 Boolean indicates whether to setup and install Calico or not

set -o pipefail

#set -xv

if [ -z "$GO_REPO_PATH" ] ; then
  echo "GO_REPO_PATH must be set"
  exit 1
fi
if [ -z "$TESTS_EXECUTED_FILE" ]; then
  echo "TESTS_EXECUTED_FILE mark file path not set, required to indicate if tests have run"
  exit 1
fi
if [ -z "$WORKSPACE" ]; then
  echo "WORKSPACE must be set"
  exit 1
fi
if [ -z "$TEST_SCRIPTS_DIR" ]; then
  echo "TEST_SCRIPTS_DIR must be set to the E2E test script directory location"
  exit 1
fi

scriptHome=$(dirname ${BASH_SOURCE[0]})

set -e

if [ -n "${VZ_TEST_DEBUG}" ]; then
  set -xv
fi

export KUBECONFIG=${KUBECONFIG:-"${WORKSPACE}/test_kubeconfig"}
export KUBERNETES_CLUSTER_VERSION=${KUBERNETES_CLUSTER_VERSION:-"1.22"}

CONNECT_JENKINS_RUNNER_TO_NETWORK=false
if [ -n "${JENKINS_URL}" ]; then
  echo "Running in Jenkins, URL=${JENKINS_URL}"
  CONNECT_JENKINS_RUNNER_TO_NETWORK=true
fi

INSTALL_CALICO=${1:-false}
CLUSTER_NAME=${CLUSTER_NAME:="kind"}
KIND_NODE_COUNT=${KIND_NODE_COUNT:-1}
VERRAZZANO_OPERATOR_IMAGE=${VERRAZZANO_OPERATOR_IMAGE:-"NONE"}

BRANCH_NAME=${BRANCH_NAME:-$(git branch --show-current)}
SHORT_COMMIT_HASH=${SHORT_COMMIT_HASH:-$(git rev-parse --short=8 HEAD)}
OCI_OS_LOCATION=${OCI_OS_LOCATION:-ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH}}

KIND_CACHING=${KIND_CACHING:="false"}
KIND_NODE_COUNT=${KIND_NODE_COUNT:-1}

mkdir -p $WORKSPACE || true

echo "tests will execute" > ${TESTS_EXECUTED_FILE}
echo "Create Kind cluster"
${scriptHome}/create_kind_clusters.sh "${CLUSTER_NAME}" "${GO_REPO_PATH}/verrazzano/platform-operator" "${KUBECONFIG}" "${KUBERNETES_CLUSTER_VERSION}" true ${CONNECT_JENKINS_RUNNER_TO_NETWORK} true $INSTALL_CALICO "NONE" ${KIND_NODE_COUNT}
if [ $? -ne 0 ]; then
    mkdir -p $WORKSPACE/kind-logs``
    kind export logs $WORKSPACE/kind-logs
    echo "Kind cluster creation failed"
    exit 1
fi

if [ $INSTALL_CALICO == true ]; then
    echo "Install Calico"
    #cd ${GO_REPO_PATH}/verrazzano
    ${scriptHome}/install_calico.sh "${CLUSTER_NAME}"
fi

# With the Calico configuration to set disableDefaultCNI to true in the KIND configuration, the control plane node will
# be ready only after applying calico.yaml. So wait for the KIND control plane node to be ready, before proceeding further,
# with maximum wait period of 5 minutes.
kubectl  wait --for=condition=ready nodes/${CLUSTER_NAME}-control-plane --timeout=5m --all
kubectl wait --for=condition=ready pods/kube-controller-manager-${CLUSTER_NAME}-control-plane -n kube-system --timeout=5m
echo "Listing pods in kube-system namespace ..."
kubectl get pods -n kube-system

echo "Install metallb"
METALLB_ADDRESS_RANGE=${METALLB_ADDRESS_RANGE:-"172.18.0.230-172.18.0.254"}
${TEST_SCRIPTS_DIR}/install-metallb.sh "${METALLB_ADDRESS_RANGE}"

exit 0
