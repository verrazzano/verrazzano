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

TEST_OVERRIDE_CONFIGMAP_FILE="${TEST_SCRIPTS_DIR}/pre-install-overrides/test-overrides-configmap.yaml"
TEST_OVERRIDE_SECRET_FILE="${TEST_SCRIPTS_DIR}/pre-install-overrides/test-overrides-secret.yaml"

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
#cd ${GO_REPO_PATH}/verrazzano
${TEST_SCRIPTS_DIR}/install-metallb.sh

echo "Create Image Pull Secrets"
${TEST_SCRIPTS_DIR}/create-image-pull-secret.sh "${IMAGE_PULL_SECRET}" "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}"
# REVIEW: Do we need github-packages still?
${TEST_SCRIPTS_DIR}/create-image-pull-secret.sh github-packages "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}"
if [ -n "${OCR_REPO}" ] && [ -n "${OCR_CREDS_USR}" ] && [ -n "${OCR_CREDS_PSW}" ]; then
  echo "Creating Oracle Container Registry pull secret"
  ${TEST_SCRIPTS_DIR}/create-image-pull-secret.sh ocr "${OCR_REPO}" "${OCR_CREDS_USR}" "${OCR_CREDS_PSW}"
fi

if ! kubectl get cm test-overrides 2>&1 > /dev/null; then
  echo "Creating Override ConfigMap"
  kubectl create cm test-overrides --from-file=${TEST_OVERRIDE_CONFIGMAP_FILE}
  if [ $? -ne 0 ]; then
    echo "Could not create Override ConfigMap"
    exit 1
  fi
fi

if ! kubectl get secret test-overrides 2>&1 > /dev/null; then
  echo "Creating Override Secret"
  kubectl create secret generic test-overrides --from-file=${TEST_OVERRIDE_SECRET_FILE}
  if [ $? -ne 0 ]; then
    echo "Could not create Override Secret"
    exit 1
  fi
fi

# optionally create a cluster dump snapshot for verifying uninstalls
if [ -n "${CLUSTER_SNAPSHOT_DIR}" ]; then
  ${TEST_SCRIPTS_DIR}/looping-test/dump_cluster.sh ${CLUSTER_SNAPSHOT_DIR}
fi

echo "Install Platform Operator"
if [ -z "$OPERATOR_YAML" ] && [ "" = "${OPERATOR_YAML}" ]; then
  # Derive the name of the wls.yaml file, copy or generate the file, then install
  if [ "NONE" = "${VERRAZZANO_OPERATOR_IMAGE}" ]; then
      echo "Using operator.yaml from object storage location ${OCI_OS_LOCATION}"
      curl -s -L https://objectstorage.us-phoenix-1.oraclecloud.com/n/${OCI_OS_NAMESPACE}/b/${OCI_OS_COMMIT_BUCKET}/o/${OCI_OS_LOCATION}/wls.yaml > ${WORKSPACE}/downloaded-wls.yaml
      cp ${WORKSPACE}/downloaded-wls.yaml ${WORKSPACE}/acceptance-test-wls.yaml
  else
      echo "Generating operator.yaml based on image name provided: ${VERRAZZANO_OPERATOR_IMAGE}"
      env IMAGE_PULL_SECRETS=verrazzano-container-registry DOCKER_IMAGE=${VERRAZZANO_OPERATOR_IMAGE} ${VZ_ROOT}/tools/scripts/generate_operator_yaml.sh > ${WORKSPACE}/acceptance-test-wls.yaml
  fi
  kubectl apply -f ${WORKSPACE}/acceptance-test-wls.yaml
else
  # The wls.yaml filename was provided, install using that file.
  echo "Using provided operator.yaml file: " ${OPERATOR_YAML}
  kubectl apply -f ${OPERATOR_YAML}
fi

# make sure ns exists
${TEST_SCRIPTS_DIR}/check_verrazzano_ns_exists.sh verrazzano-install

# create secret in verrazzano-install ns
${TEST_SCRIPTS_DIR}/create-image-pull-secret.sh "${IMAGE_PULL_SECRET}" "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}" "verrazzano-install"

echo "Wait for Operator to be ready"
kubectl -n verrazzano-install rollout status deployment/verrazzano-platform-wls
if [ $? -ne 0 ]; then
  echo "Operator is not ready"
  exit 1
fi

exit 0
