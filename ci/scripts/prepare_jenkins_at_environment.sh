#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

if [ -z "$JENKINS_URL" ] || [ -z "$GO_REPO_PATH" ] || [ -z "$TESTS_EXECUTED_FILE" ] || [ -z "$WORKSPACE" ] || [ -z "$VERRAZZANO_OPERATOR_IMAGE" ] || [ -z "$INSTALL_CONFIG_FILE_KIND" ] || [ -z "$OCI_OS_LOCATION" ] || [ -z "$TEST_SCRIPTS_DIR" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

cd ${GO_REPO_PATH}/verrazzano
echo "tests will execute" > ${TESTS_EXECUTED_FILE}
echo "Create Kind cluster"
cd ${TEST_SCRIPTS_DIR}
./create_kind_cluster.sh "${CLUSTER_NAME}" "${GO_REPO_PATH}/verrazzano/platform-operator" "${KUBECONFIG}" "${KIND_KUBERNETES_CLUSTER_VERSION}" true true true

#echo "Install Calico"
#kubectl apply -f https://docs.projectcalico.org/manifests/canal.yaml

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
if [ "NONE" = "${VERRAZZANO_OPERATOR_IMAGE}" ]; then
    echo "Using operator.yaml from object storage"
    oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${OCI_OS_LOCATION}/operator.yaml --file ${WORKSPACE}/downloaded-operator.yaml
    cp ${WORKSPACE}/downloaded-operator.yaml ${WORKSPACE}/acceptance-test-operator.yaml
else
    echo "Generating operator.yaml based on image name provided: ${VERRAZZANO_OPERATOR_IMAGE}"
    env IMAGE_PULL_SECRETS=verrazzano-container-registry DOCKER_IMAGE=${VERRAZZANO_OPERATOR_IMAGE} ./tools/scripts/generate_operator_yaml.sh > ${WORKSPACE}/acceptance-test-operator.yaml
fi
kubectl apply -f ${WORKSPACE}/acceptance-test-operator.yaml

# make sure ns exists
./tests/e2e/config/scripts/check_verrazzano_ns_exists.sh verrazzano-install

# create secret in verrazzano-install ns
./tests/e2e/config/scripts/create-image-pull-secret.sh "${IMAGE_PULL_SECRET}" "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}" "verrazzano-install"

# Configure the custom resource to install verrazzano on Kind
echo "Installing yq"
GO111MODULE=on go get github.com/mikefarah/yq/v4
export PATH=${HOME}/go/bin:${PATH}
./tests/e2e/config/scripts/process_kind_install_yaml.sh ${INSTALL_CONFIG_FILE_KIND}

echo "Wait for Operator to be ready"
cd ${GO_REPO_PATH}/verrazzano
kubectl -n verrazzano-install rollout status deployment/verrazzano-platform-operator

echo "Installing Verrazzano on Kind"
kubectl apply -f ${INSTALL_CONFIG_FILE_KIND}

${GO_REPO_PATH}/verrazzano/tools/scripts/k8s-dump-cluster.sh -d debug-new-kind-acceptance-tests-cluster-dump -r debug-new-kind-acceptance-tests-cluster-dump/analysis.report

# wait for Verrazzano install to complete
./tests/e2e/config/scripts/wait-for-verrazzano-install.sh

exit 0
