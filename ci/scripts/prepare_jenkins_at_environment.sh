#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# $1 Boolean indicates whether to setup and install Calico or not

set -o pipefail

set -xv

if [ -z "$JENKINS_URL" ] || [ -z "$GO_REPO_PATH" ] || [ -z "$TESTS_EXECUTED_FILE" ] || [ -z "$WORKSPACE" ] || [ -z "$VERRAZZANO_OPERATOR_IMAGE" ] || [ -z "$INSTALL_CONFIG_FILE_KIND" ] || [ -z "$OCI_OS_LOCATION" ] || [ -z "$OCI_OS_COMMIT_BUCKET" ] || [ -z "$TEST_SCRIPTS_DIR" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

if ! [ -x "$(command -v go)" ]; then
  echo "vz command-line tool requires go which does not appear to be installed"
  exit 1
fi

INSTALL_CALICO=${1:-false}
WILDCARD_DNS_DOMAIN=${2:-"x=nip.io"}
USE_DB_FOR_GRAFANA=${3:-false}
KIND_NODE_COUNT=${KIND_NODE_COUNT:-1}
TEST_OVERRIDE_CONFIGMAP_FILE="./tests/e2e/config/scripts/pre-install-overrides/test-overrides-configmap.yaml"
TEST_OVERRIDE_SECRET_FILE="./tests/e2e/config/scripts/pre-install-overrides/test-overrides-secret.yaml"
INSTALL_TIMEOUT_VALUE=${INSTALL_TIMEOUT:-30m}

cd ${GO_REPO_PATH}/verrazzano
echo "tests will execute" > ${TESTS_EXECUTED_FILE}
echo "Create Kind cluster"
cd ${TEST_SCRIPTS_DIR}
./create_kind_cluster.sh "${CLUSTER_NAME}" "${GO_REPO_PATH}/verrazzano/platform-operator" "${KUBECONFIG}" "${KIND_KUBERNETES_CLUSTER_VERSION}" true true true $INSTALL_CALICO "NONE" ${KIND_NODE_COUNT}
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

echo "Determine which yaml file to use to install the Verrazzano Platform Operator"
cd ${GO_REPO_PATH}/verrazzano

TARGET_OPERATOR_FILE=${TARGET_OPERATOR_FILE:-"${WORKSPACE}/acceptance-test-operator.yaml"}
if [ -z "$OPERATOR_YAML" ] && [ "" = "${OPERATOR_YAML}" ]; then
  # Derive the name of the operator.yaml file, copy or generate the file, then install
  if [ "NONE" = "${VERRAZZANO_OPERATOR_IMAGE}" ]; then
      echo "Using operator.yaml from object storage"
      oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ${OCI_OS_LOCATION}/operator.yaml --file ${WORKSPACE}/downloaded-operator.yaml
      cp -v ${WORKSPACE}/downloaded-operator.yaml ${TARGET_OPERATOR_FILE}
  else
      echo "Generating operator.yaml based on image name provided: ${VERRAZZANO_OPERATOR_IMAGE}"
      env IMAGE_PULL_SECRETS=verrazzano-container-registry DOCKER_IMAGE=${VERRAZZANO_OPERATOR_IMAGE} ./tools/scripts/generate_operator_yaml.sh > ${TARGET_OPERATOR_FILE}
  fi
else
  # The operator.yaml filename was provided, install using that file.
  echo "Using provided operator.yaml file: " ${OPERATOR_YAML}
  TARGET_OPERATOR_FILE=${OPERATOR_YAML}
fi

VZ_CLI_TARGZ="vz-linux-amd64.tar.gz"
echo "Downloading VZ CLI from object storage"
if [[ -z "$OCI_OS_LOCATION" ]]; then
  OCI_OS_LOCATION="$BRANCH_NAME/$(git rev-parse --short=8 HEAD)"
fi
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ${OCI_OS_LOCATION}/$VZ_CLI_TARGZ --file ${WORKSPACE}/$VZ_CLI_TARGZ
tar xzf "$WORKSPACE"/$VZ_CLI_TARGZ -C "$WORKSPACE"

# Create the verrazzano-install namespace
kubectl create namespace verrazzano-install

# if enabled, deploy the Grafana MySQL instance and create the Grafana DB secret
if [ $USE_DB_FOR_GRAFANA == true ]; then
  # create the necessary secrets
  MYSQL_ROOT_PASSWORD=$(openssl rand -base64 12)
  MYSQL_PASSWORD=$(openssl rand -base64 12)
  ROOT_SECRET=$(echo -n $MYSQL_ROOT_PASSWORD | base64)
  USER_SECRET=$(echo -n $MYSQL_PASSWORD | base64)
  USER=$(echo -n "grafana" | base64)

  kubectl apply -f - <<-EOF
apiVersion: v1
kind: Secret
metadata:
  name: grafana-db
  namespace: verrazzano-install
type: Opaque
data:
  root-password: $ROOT_SECRET
  username: $USER
  password: $USER_SECRET
EOF
  # deploy MySQL instance
  kubectl apply -f $WORKSPACE/tests/testdata/grafana/grafana-mysql.yaml
fi

# create secret in verrazzano-install ns
./tests/e2e/config/scripts/create-image-pull-secret.sh "${IMAGE_PULL_SECRET}" "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}" "verrazzano-install"

# optionally create a cluster dump snapshot for verifying uninstalls
if [ -n "${CLUSTER_SNAPSHOT_DIR}" ]; then
  ./tests/e2e/config/scripts/looping-test/dump_cluster.sh ${CLUSTER_SNAPSHOT_DIR}
fi

# Configure the custom resource to install Verrazzano on Kind
VZ_INSTALL_FILE=${VZ_INSTALL_FILE:-"${WORKSPACE}/acceptance-test-config.yaml"}
./tests/e2e/config/scripts/process_kind_install_yaml.sh ${INSTALL_CONFIG_FILE_KIND} ${WILDCARD_DNS_DOMAIN}
# If grafana is using a DB add the database configuration to the VZ file
if [ $USE_DB_FOR_GRAFANA == true ]; then
  ./tests/e2e/config/scripts/process_grafana_db_install_yaml.sh ${INSTALL_CONFIG_FILE_KIND}
fi
cp -v ${INSTALL_CONFIG_FILE_KIND} ${VZ_INSTALL_FILE}

echo "Creating Override ConfigMap"
kubectl create cm test-overrides --from-file=${TEST_OVERRIDE_CONFIGMAP_FILE}
if [ $? -ne 0 ]; then
  echo "Could not create Override ConfigMap"
  exit 1
fi

echo "Creating Override Secret"
kubectl create secret generic test-overrides --from-file=${TEST_OVERRIDE_SECRET_FILE}
if [ $? -ne 0 ]; then
  echo "Could not create Override Secret"
  exit 1
fi

echo "Installing Verrazzano on Kind"
if [ -f "$WORKSPACE/vz" ]; then
  cd $WORKSPACE
  ./vz install --filename ${WORKSPACE}/acceptance-test-config.yaml --operator-file ${TARGET_OPERATOR_FILE} --timeout ${INSTALL_TIMEOUT_VALUE}
else
  cd ${GO_REPO_PATH}/verrazzano/tools/vz
  GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go install --filename ${VZ_INSTALL_FILE} --operator-file ${TARGET_OPERATOR_FILE} --timeout ${INSTALL_TIMEOUT_VALUE}
fi
result=$?
if [[ $result -ne 0 ]]; then
  exit 1
fi

exit 0
