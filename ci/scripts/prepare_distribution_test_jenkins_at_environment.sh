#!/usr/bin/env bash
#
# Copyright (c) 2022, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# $1 Boolean indicates whether to setup and install Calico or not

set -o pipefail

set -xv

if [ -z "$GO_REPO_PATH" ] || [ -z "$WORKSPACE" ] || [ -z "$TARBALL_DIR" ] || [ -z "$CLUSTER_NAME" ] ||
  [ -z "$KIND_KUBERNETES_CLUSTER_VERSION" ] || [ -z "$KUBECONFIG" ] ||
  [ -z "$IMAGE_PULL_SECRET" ] || [ -z "$PRIVATE_REPO" ] || [ -z "$REGISTRY" ] || [ -z "$PRIVATE_REGISTRY_USR" ] ||
  [ -z "$PRIVATE_REGISTRY_PSW" ] || [ -z "$VZ_ENVIRONMENT_NAME" ] || [ -z "$INSTALL_PROFILE" ] ||
  [ -z "$TESTS_EXECUTED_FILE" ] || [ -z "$TEST_SCRIPTS_DIR" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

INSTALL_CALICO=${1:-false}
WILDCARD_DNS_DOMAIN=${2:-"nip.io"}

if [ -z "$3" ]; then
  echo "Location of verrazzano install file must be specified"
  exit 1
fi

INSTALL_CONFIG_FILE_KIND="$3"
KIND_NODE_COUNT=${KIND_NODE_COUNT:-1}

BOM_FILE=${TARBALL_DIR}/manifests/verrazzano-bom.json
CHART_LOCATION=${TARBALL_DIR}/manifests/charts

cd ${GO_REPO_PATH}/verrazzano
echo "tests will execute" > ${TESTS_EXECUTED_FILE}
echo "Create Kind cluster"
cd ${TEST_SCRIPTS_DIR}
SETUP_PRIVATE_REGISTRY=true ./create_kind_cluster.sh "${CLUSTER_NAME}" "${GO_REPO_PATH}/verrazzano/platform-operator" "${KUBECONFIG}" "${KIND_KUBERNETES_CLUSTER_VERSION}" true true true $INSTALL_CALICO "NONE" ${KIND_NODE_COUNT}

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
./tests/e2e/config/scripts/create-image-pull-secret.sh "${IMAGE_PULL_SECRET}" "${REGISTRY}" "${PRIVATE_REGISTRY_USR}" "${PRIVATE_REGISTRY_PSW}"
./tests/e2e/config/scripts/create-image-pull-secret.sh ocr "${OCR_REPO}" "${OCR_CREDS_USR}" "${OCR_CREDS_PSW}"

# Create docker secret for platform operator image
kubectl create ns verrazzano-install
./tests/e2e/config/scripts/create-image-pull-secret.sh "${IMAGE_PULL_SECRET}" "${REGISTRY}" "${PRIVATE_REGISTRY_USR}" "${PRIVATE_REGISTRY_PSW}" verrazzano-install

# optionally create a cluster dump snapshot for verifying uninstalls
if [ -n "${CLUSTER_SNAPSHOT_DIR}" ]; then
  ./tests/e2e/config/scripts/looping-test/dump_cluster.sh ${CLUSTER_SNAPSHOT_DIR}
fi

yq eval -i '.metadata.name = "verrazzano"' ${INSTALL_CONFIG_FILE_KIND}
yq eval -i '.spec.components.prometheusAdapter.enabled = true' ${INSTALL_CONFIG_FILE_KIND}
yq eval -i '.spec.components.kubeStateMetrics.enabled = true' ${INSTALL_CONFIG_FILE_KIND}
yq eval -i '.spec.components.prometheusPushgateway.enabled = true' ${INSTALL_CONFIG_FILE_KIND}
yq eval -i '.spec.components.velero.enabled = true' ${INSTALL_CONFIG_FILE_KIND}
yq eval -i '.spec.components.rancherBackup.enabled = true' ${INSTALL_CONFIG_FILE_KIND}
yq eval -i '.spec.components.jaegerOperator.enabled = true' ${INSTALL_CONFIG_FILE_KIND}
yq eval -i '.spec.components.argoCD.enabled = true' ${INSTALL_CONFIG_FILE_KIND}
yq eval -i '.spec.components.clusterAPI.enabled = true' ${INSTALL_CONFIG_FILE_KIND}

# Configure the custom resource to install Verrazzano on Kind
./tests/e2e/config/scripts/process_kind_install_yaml.sh ${INSTALL_CONFIG_FILE_KIND} ${WILDCARD_DNS_DOMAIN}

echo "Installing Verrazzano on Kind"
${GO_REPO_PATH}/vz install -f "${INSTALL_CONFIG_FILE_KIND}" --manifests "${TARBALL_DIR}/manifests/k8s/verrazzano-platform-operator.yaml" --image-registry ${REGISTRY} --image-prefix ${PRIVATE_REPO}
