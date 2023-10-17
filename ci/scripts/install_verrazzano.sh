#!/usr/bin/env bash
# Copyright (c) 2022, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Required env vars:
# INSTALL_CONFIG_FILE_KIND - source Verrazzano install configuration for KIND
# WORKSPACE - workspace for output files, temp files, etc
# TEST_SCRIPTS_DIR - Location of the E2E tests directory
# KUBECONFIG - kubeconfig path for the target cluster
# GO_REPO_PATH - Local path to the Verrazzano Github repo
#
# Indirect/optional env vars (used to process the installation config):
#
# INSTALL_PROFILE - Verrazzano profile, defaults to "dev"
# VZ_ENVIRONMENT_NAME - environmentName default
# EXTERNAL_ELASTICSEARCH - if "true" && VZ_ENVIRONMENT_NAME=="admin", sets Fluentd configuration to point to EXTERNAL_ES_SECRET and EXTERNAL_ES_URL
# SYSTEM_LOG_ID - configures Verrazzano for OCI logging using the specified OCI logging ID
# ENABLE_API_ENVOY_LOGGING - enables debug in the Istio Envoy containers
# WILDCARD_DNS_DOMAIN - an override for a user-specified wildcard DNS domain to use
# VERRAZZANO_INSTALL_LOGS_DIR - output location for the VZ install logs
#
set -o pipefail

if [ -z "$GO_REPO_PATH" ] ; then
  echo "GO_REPO_PATH must be set"
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
if [ -z "${KUBECONFIG}" ]; then
  echo "KUBECONFIG must be set"
  exit 1
fi
if [ -z "$INSTALL_CONFIG_FILE_KIND" ]; then
  echo "INSTALL_CONFIG_FILE_KIND must be set to valid Verrazzano install file"
  exit 1
fi

scriptHome=$(dirname ${BASH_SOURCE[0]})

set -e
if [ -n "${VZ_TEST_DEBUG}" ]; then
  set -xv
fi

INSTALL_TIMEOUT_VALUE=${INSTALL_TIMEOUT_VALUE:-30m}
WILDCARD_DNS_DOMAIN=${WILDCARD_DNS_DOMAIN:-""}
ENABLE_API_ENVOY_LOGGING=${ENABLE_API_ENVOY_LOGGING:-false}
CREATE_TEST_OVERRIDES=${CREATE_TEST_OVERRIDES:-true}
USE_DB_FOR_GRAFANA=${USE_DB_FOR_GRAFANA:-false}

INSTALL_PROFILE=${INSTALL_PROFILE:-"dev"}
VERRAZZANO_INSTALL_LOGS_DIR=${VERRAZZANO_INSTALL_LOGS_DIR:-${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs}

VERRAZZANO_INSTALL_LOGS_DIR=${VERRAZZANO_INSTALL_LOGS_DIR:-${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs}

is_macos () {
	if [[ "${OSTYPE}" == "darwin"** ]]
	then
		return 0
	fi
	return 1
}

get_arch_type() {
  local os=linux
  if is_macos ; then
    os=darwin
  fi
  echo $os
}

setup_overrides() {
  # We should make this optional
  TEST_OVERRIDE_CONFIGMAP_FILE="${TEST_SCRIPTS_DIR}/pre-install-overrides/test-overrides-configmap.yaml"
  TEST_OVERRIDE_SECRET_FILE="${TEST_SCRIPTS_DIR}/pre-install-overrides/test-overrides-secret.yaml"

  echo "Creating Override ConfigMap"
  if ! kubectl get cm test-overrides 2>&1 > /dev/null; then
    if ! kubectl create cm test-overrides --from-file="${TEST_OVERRIDE_CONFIGMAP_FILE}" ; then
      echo "Could not create Override ConfigMap"
      exit 1
    fi
  fi

  echo "Creating Override Secret"
  if ! kubectl get secret test-overrides 2>&1 > /dev/null; then
    if ! kubectl create secret generic test-overrides --from-file="${TEST_OVERRIDE_SECRET_FILE}" ; then
      echo "Could not create Override Secret"
      exit 1
    fi
  fi
}

setup_pull_secrets() {
  echo "Create Image Pull Secrets"
  ${TEST_SCRIPTS_DIR}/create-image-pull-secret.sh "${IMAGE_PULL_SECRET}" "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}"
  # REVIEW: Do we need github-packages still?
  ${TEST_SCRIPTS_DIR}/create-image-pull-secret.sh github-packages "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}"
  if [ -n "${OCR_REPO}" ] && [ -n "${OCR_CREDS_USR}" ] && [ -n "${OCR_CREDS_PSW}" ]; then
    echo "Creating Oracle Container Registry pull secret"
    ${TEST_SCRIPTS_DIR}/create-image-pull-secret.sh ocr "${OCR_REPO}" "${OCR_CREDS_USR}" "${OCR_CREDS_PSW}"
  fi

  # Create the verrazzano-install namespace
  kubectl create namespace verrazzano-install || true

  # create secret in verrazzano-install ns
  ${TEST_SCRIPTS_DIR}/create-image-pull-secret.sh "${IMAGE_PULL_SECRET}" "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}" "verrazzano-install"
}

setup_operator_yaml() {
  echo "Determine which yaml file to use to install the Verrazzano Platform Operator"
  TARGET_OPERATOR_FILE=${TARGET_OPERATOR_FILE:-"${WORKSPACE}/acceptance-test-operator.yaml"}
  if [ -z "$OPERATOR_YAML" ]; then
    # Derive the name of the operator.yaml file, copy or generate the file, then install
    if  [ -z "${VERRAZZANO_OPERATOR_IMAGE}" ] || [ "NONE" == "${VERRAZZANO_OPERATOR_IMAGE}" ]; then
        echo "Using operator.yaml from object storage"
        oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${OCI_OS_LOCATION}/operator.yaml --file ${WORKSPACE}/downloaded-operator.yaml
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
}

setup_grafana_mysql() {
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
}

create_install_file() {
  # Configure the custom resource to install Verrazzano on Kind
  VZ_INSTALL_FILE=${VZ_INSTALL_FILE:-"${WORKSPACE}/acceptance-test-config.yaml"}
  ./tests/e2e/config/scripts/process_kind_install_yaml.sh ${INSTALL_CONFIG_FILE_KIND} ${WILDCARD_DNS_DOMAIN}
  # If grafana is using a DB add the database configuration to the VZ file
  if [ $USE_DB_FOR_GRAFANA == true ]; then
    ./tests/e2e/config/scripts/process_grafana_db_install_yaml.sh ${INSTALL_CONFIG_FILE_KIND}
  fi
  cp -v ${INSTALL_CONFIG_FILE_KIND} ${VZ_INSTALL_FILE}

  setup_overrides
}

install_verrazzano() {
  if [[ -n "${OCI_OS_LOCATION}" ]]; then
    ARCHTYPE=$(get_arch_type)
    VZ_CLI_TARGZ="vz-${ARCHTYPE}-amd64.tar.gz"
    echo "Downloading VZ CLI from object storage"
    oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${OCI_OS_LOCATION}/$VZ_CLI_TARGZ --file ${WORKSPACE}/$VZ_CLI_TARGZ
    tar xzf "$WORKSPACE"/$VZ_CLI_TARGZ -C "$WORKSPACE"
  fi

  echo "Installing Verrazzano on Kind"
  if [ -f "$WORKSPACE/vz" ]; then
    cd $WORKSPACE
    ./vz install --filename ${WORKSPACE}/acceptance-test-config.yaml --manifests ${TARGET_OPERATOR_FILE} --timeout ${INSTALL_TIMEOUT_VALUE}
  else
    cd ${GO_REPO_PATH}/verrazzano/tools/vz
    GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go install --filename ${VZ_INSTALL_FILE} --manifests ${TARGET_OPERATOR_FILE} --timeout ${INSTALL_TIMEOUT_VALUE}
  fi

  result=$?
  if [[ $result -ne 0 ]]; then
    exit 1
  fi
}

cd ${GO_REPO_PATH}/verrazzano

setup_pull_secrets
setup_operator_yaml
setup_grafana_mysql

# optionally create a cluster dump snapshot for verifying uninstalls
if [ -n "${CLUSTER_SNAPSHOT_DIR}" ]; then
  ./tests/e2e/config/scripts/looping-test/dump_cluster.sh ${CLUSTER_SNAPSHOT_DIR}
fi

create_install_file
install_verrazzano

exit 0

