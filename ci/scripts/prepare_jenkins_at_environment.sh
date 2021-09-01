#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# $1 Boolean indicates whether to setup and install Calico or not

set -o pipefail

set -xv

if [ -z "$JENKINS_URL" ] || [ -z "$GO_REPO_PATH" ] || [ -z "$TESTS_EXECUTED_FILE" ] || [ -z "$WORKSPACE" ] || [ -z "$VERRAZZANO_OPERATOR_IMAGE" ] || [ -z "$INSTALL_CONFIG_FILE_KIND" ] || [ -z "$OCI_OS_LOCATION" ] || [ -z "$TEST_SCRIPTS_DIR" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

INSTALL_CALICO=${1:-false}
WILDCARD_DNS_DOMAIN=${2:-"x=nip.io"}

cd ${GO_REPO_PATH}/verrazzano
echo "tests will execute" > ${TESTS_EXECUTED_FILE}
echo "Create Kind cluster"
cd ${TEST_SCRIPTS_DIR}
./create_kind_cluster.sh "${CLUSTER_NAME}" "${GO_REPO_PATH}/verrazzano/platform-operator" "${KUBECONFIG}" "${KIND_KUBERNETES_CLUSTER_VERSION}" true true true $INSTALL_CALICO
if [ $? -ne 0 ]; then
    mkdir $WORKSPACE/kind-logs
    kind export logs $WORKSPACE/kind-logs
    echo "Kind cluster creation failed"
    exit 1
fi

if [ $INSTALL_CALICO == true ]; then
    echo "Install Calico"
    cd ${GO_REPO_PATH}/verrazzano
    ./ci/scripts/install_calico.sh "${CLUSTER_NAME}"
fi

# With the Calico configuration to set disableDefaultCNI to true in the KIND configuration, the control plane node will
# be ready only after applying calico.yaml. So wait for the KIND control plane node to be ready, before proceeding further,
# with maximum wait period of 5 minutes.
kubectl wait --for=condition=ready nodes/${CLUSTER_NAME}-control-plane --timeout=5m --all
kubectl wait --for=condition=ready pods/kube-controller-manager-${CLUSTER_NAME}-control-plane -n kube-system --timeout=5m
echo "Listing pods in kube-system namespace ..."
kubectl get pods -n kube-system

echo "Install metallb"
cd ${GO_REPO_PATH}/verrazzano
./tests/e2e/config/scripts/install-metallb.sh

echo "Create Image Pull Secrets"
cd ${GO_REPO_PATH}/verrazzano
./tests/e2e/config/scripts/create-image-pull-secret.sh "${IMAGE_PULL_SECRET}" "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}"
./tests/e2e/config/scripts/create-image-pull-secret.sh github-packages "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}"
./tests/e2e/config/scripts/create-image-pull-secret.sh ocr "${OCR_REPO}" "${OCR_CREDS_USR}" "${OCR_CREDS_PSW}"

echo "Install Platform Operator"
cd ${GO_REPO_PATH}/verrazzano

if [ -z "$OPERATOR_YAML" ] && [ "" = "${OPERATOR_YAML}" ]; then
  # Derive the name of the operator.yaml file, copy or generate the file, then install
  if [ "NONE" = "${VERRAZZANO_OPERATOR_IMAGE}" ]; then
      echo "Using operator.yaml from object storage"
      oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${OCI_OS_LOCATION}/operator.yaml --file ${WORKSPACE}/downloaded-operator.yaml
      cp ${WORKSPACE}/downloaded-operator.yaml ${WORKSPACE}/acceptance-test-operator.yaml
  else
      echo "Generating operator.yaml based on image name provided: ${VERRAZZANO_OPERATOR_IMAGE}"
      env IMAGE_PULL_SECRETS=verrazzano-container-registry DOCKER_IMAGE=${VERRAZZANO_OPERATOR_IMAGE} ./tools/scripts/generate_operator_yaml.sh > ${WORKSPACE}/acceptance-test-operator.yaml
  fi
  kubectl apply -f ${WORKSPACE}/acceptance-test-operator.yaml
else
  # The operator.yaml filename was provided, install using that file.
  echo "Using provided operator.yaml file: " ${OPERATOR_YAML}
  kubectl apply -f ${OPERATOR_YAML}
fi



# make sure ns exists
./tests/e2e/config/scripts/check_verrazzano_ns_exists.sh verrazzano-install

# create secret in verrazzano-install ns
./tests/e2e/config/scripts/create-image-pull-secret.sh "${IMAGE_PULL_SECRET}" "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}" "verrazzano-install"

# Configure the custom resource to install verrazzano on Kind
./tests/e2e/config/scripts/process_kind_install_yaml.sh ${INSTALL_CONFIG_FILE_KIND} ${WILDCARD_DNS_DOMAIN}

echo "Wait for Operator to be ready"
cd ${GO_REPO_PATH}/verrazzano
kubectl -n verrazzano-install rollout status deployment/verrazzano-platform-operator
if [ $? -ne 0 ]; then
  echo "Operator is not ready"
  exit 1
fi

echo "Installing Verrazzano on Kind"
install_retries=0
until kubectl apply -f ${INSTALL_CONFIG_FILE_KIND}; do
  install_retries=$((install_retries+1))
  sleep 6
  if [ $install_retries -ge 10 ] ; then
    echo "Installation Failed trying to apply the Verazzano CR YAML"
    exit 1
  fi
done

${GO_REPO_PATH}/verrazzano/tools/scripts/k8s-dump-cluster.sh -d ${WORKSPACE}/debug-new-kind-acceptance-tests-cluster-dump -r ${WORKSPACE}/debug-new-kind-acceptance-tests-cluster-dump/analysis.report

# wait for Verrazzano install to complete
./tests/e2e/config/scripts/wait-for-verrazzano-install.sh
if [ $? -ne 0 ]; then
  exit 1
fi

exit 0
