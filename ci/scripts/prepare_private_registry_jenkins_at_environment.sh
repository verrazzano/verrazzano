#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# $1 Boolean indicates whether to setup and install Calico or not

set -o pipefail

set -xv

INSTALL_CALICO=${1:-false}
WILDCARD_DNS_DOMAIN=${2:-"nip.io"}
BASE_IMAGE_REPO=${3:-""}   # primarily used for Harbor ephemeral

BOM_FILE=${TARBALL_DIR}/verrazzano-bom.json
CHART_LOCATION=${TARBALL_DIR}/charts

deploy_contour () {
  local namespace="projectcontour"
  kubectl apply -f https://projectcontour.io/quickstart/contour.yaml
  kubectl patch daemonsets -n ${namespace} envoy -p '{"spec":{"template":{"spec":{"nodeSelector":{"ingress-ready":"true"},"tolerations":[{"key":"node-role.kubernetes.io/master","operator":"Equal","effect":"NoSchedule"}]}}}}'
  # wait for contour to complete
#  ${GO_REPO_PATH}/verrazzano/tests/e2e/config/scripts/wait-for-k8s-resource.sh ${namespace} "ready" "pod" false "app.kubernetes.io/component=controller"
#  if [ $? -ne 0 ]; then
#    echo "Deployment of contour failed"
#    exit 1
#  fi
  sleep 50
}

install_new_helm_version() {
  cd ${WORKSPACE}
  # Downloading newer version of helm in order to prevent harbor installation issues
  helm_zip="helm-v3.7.2-linux-amd64.tar.gz"
  echo "Downloading helm 3.7.2 via command: curl -fsSL -o ${helm_zip} https://get.helm.sh/${helm_zip}"
  curl -fsSL -o ${helm_zip} https://get.helm.sh/${helm_zip}
  if [ $? -ne 0 ]; then
    echo "Error downloading helm 3.7.2"
    exit 1
  fi
  tar -zxvf ${helm_zip}
  if [ $? -ne 0 ]; then
    exit 1
  fi
}

deploy_certificates() {
  local namespace="cert-manager"
  cd ${WORKSPACE}/linux-amd64
  ./helm --kubeconfig=${KUBECONFIG} repo add jetstack https://charts.jetstack.io
  ./helm --kubeconfig=${KUBECONFIG} repo update
  kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.6.1/cert-manager.crds.yaml
  ./helm --kubeconfig=${KUBECONFIG} install cert-manager jetstack/cert-manager --namespace ${cert-manager} --create-namespace --version v1.6.1
  kubectl apply -f - <<EOF
    apiVersion: cert-manager.io/v1
    kind: ClusterIssuer
    metadata:
      # This issuer has low thresholds for rate limits,
      # so only use once bugs have been worked out for ingress stanzas
      name: letsencrypt-prod
    spec:
      acme:
        server: https://acme-v02.api.letsencrypt.org/directory
        email: dev@abc.com
        privateKeySecretRef:
          name: letsencrypt-prod
        # Enable the HTTP-01 challenge provider
        #http01: {}
EOF

  # wait for cert-manager to complete
#  ${GO_REPO_PATH}/verrazzano/tests/e2e/config/scripts/wait-for-k8s-resource.sh ${namespace} "ready" "pod" true
#  if [ $? -ne 0 ]; then
#    echo "Deployment of certificates failed"
#    exit 1
#  fi
  sleep 50
}

load_images() {
  # Run the image-helper to load the images into the Harbor registry
  cd ${TARBALL_DIR}
  ${TARBALL_DIR}/vz-registry-image-helper.sh -t ${HARBOR_EPHEMERAL_REGISTRY} -l . -r ${BASE_IMAGE_REPO}
  if [ $? -ne 0 ]; then
    echo "Loading images into Harbor failed"
    exit 1
  fi
}

deploy_harbor() {
  cd ${WORKSPACE}/linux-amd64
  # Install harbor via new version of helm
  ./helm --kubeconfig=${KUBECONFIG} repo add harbor https://helm.goharbor.io
  ./helm --kubeconfig=${KUBECONFIG} repo update
  ./helm --kubeconfig=${KUBECONFIG} install ephemeral-harbor harbor/harbor \
    --set expose.ingress.hosts.core=${REGISTRY} \
    --set expose.ingress.annotations.'kubernetes\.io/ingress\.class'=contour \
    --set expose.ingress.annotations.'certmanager\.k8s\.io/cluster-issuer'=letsencrypt-prod \
    --set externalURL=https://${REGISTRY} \
    --set expose.tls.secretName=ephemeral-harbor-ingress-cert \
    --set notary.enabled=false \
    --set notary.trivy=false \
    --set persistence.enabled=false \
    --set harborAdminPassword=${PRIVATE_REGISTRY_PSW}

  # wait for harbor installation to complete
#  ${GO_REPO_PATH}/verrazzano/tests/e2e/config/scripts/wait-for-k8s-resource.sh "default" "ready" "pod" true
#  if [ $? -ne 0 ]; then
#    exit 1
#  fi
  sleep 50

  docker login ${REGISTRY} -u ${PRIVATE_REGISTRY_USR} -p ${PRIVATE_REGISTRY_PSW}
  if [ $? -ne 0 ]; then
    echo "docker login to Harbor ephemeral failed"
  fi

  docker pull nginx:1-alpine
  docker tag nginx:1-alpine ${REGISTRY}/library/nginx:1-test
  docker push ${REGISTRY}/library/nginx:1-test

  cd ${TEST_SCRIPTS_DIR}
  # Create the Harbor project if it does not exist
  ./create_harbor_project.sh -a "https://${REGISTRY}/api/v2.0" -u ${PRIVATE_REGISTRY_USR} -p ${PRIVATE_REGISTRY_PSW} -m ${IMAGE_REPO_SUBPATH_PREFIX} -l false
  if [ $? -ne 0 ]; then
    echo "Harbor installation failed"
    exit 1
  fi
}

start_installation() {
  if [ -z "${GO_REPO_PATH}" ] || [ -z "${WORKSPACE}" ] || [ -z "${TARBALL_DIR}" ] || [ -z "${CLUSTER_NAME}" ] ||
    [ -z "${KIND_KUBERNETES_CLUSTER_VERSION}" ] || [ -z "${KUBECONFIG}" ] ||
    [ -z "${IMAGE_PULL_SECRET}" ] || [ -z "${PRIVATE_REPO}" ] || [ -z "${REGISTRY}" ] || [ -z "${PRIVATE_REGISTRY_USR}" ] ||
    [ -z "${PRIVATE_REGISTRY_PSW}" ] || [ -z "${VZ_ENVIRONMENT_NAME}" ] || [ -z "${INSTALL_PROFILE}" ] ||
    [ -z "${TESTS_EXECUTED_FILE}" ] || [ -z "${INSTALL_CONFIG_FILE_KIND}" ] || [ -z "${TEST_SCRIPTS_DIR}" ] || [ -z "${SETUP_HARBOR}" ]; then
    echo "This script must only be called from Jenkins and requires a number of environment variables are set"
    exit 1
  fi

  cd ${GO_REPO_PATH}/verrazzano
  echo "tests will execute" > ${TESTS_EXECUTED_FILE}
  echo "Create Kind cluster"
  cd ${TEST_SCRIPTS_DIR}
  ./create_kind_cluster.sh "${CLUSTER_NAME}" "${GO_REPO_PATH}/verrazzano/platform-operator" "${KUBECONFIG}" "${KIND_KUBERNETES_CLUSTER_VERSION}" true true true $SETUP_HARBOR $INSTALL_CALICO

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

  echo "Listing pods in kube-system namespace just before harbor ephemeral installation..."
  kubectl get pods -n kube-system

  if [ $SETUP_HARBOR == true ]; then
    install_new_helm_version
    deploy_certificates
    deploy_contour
    deploy_harbor
    load_images
  fi

  cd ${GO_REPO_PATH}/verrazzano
  echo "Install Platform Operator"
  VPO_IMAGE=$(cat ${BOM_FILE} | jq -r '.components[].subcomponents[] | select(.name == "verrazzano-platform-operator") | "\(.repository)/\(.images[].image):\(.images[].tag)"')

  helm upgrade --install myv8o ${CHART_LOCATION}/verrazzano-platform-operator \
      --set global.imagePullSecrets[0]=${IMAGE_PULL_SECRET} \
      --set image=${REGISTRY}/${PRIVATE_REPO}/${VPO_IMAGE} --set global.registry=${REGISTRY} \
      --set global.repository=${PRIVATE_REPO}

  # make sure ns exists
  ./tests/e2e/config/scripts/check_verrazzano_ns_exists.sh verrazzano-install

  # Create docker secret for platform operator image
  ./tests/e2e/config/scripts/create-image-pull-secret.sh "${IMAGE_PULL_SECRET}" "${REGISTRY}" "${PRIVATE_REGISTRY_USR}" "${PRIVATE_REGISTRY_PSW}" verrazzano-install

  # Configure the custom resource to install Verrazzano on Kind
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
      echo "Installation Failed trying to apply the Verrazzano CR YAML"
      exit 1
    fi
  done

  ${GO_REPO_PATH}/verrazzano/tools/scripts/k8s-dump-cluster.sh -d ${WORKSPACE}/post-vz-install-cluster-dump -r ${WORKSPACE}/post-vz-install-cluster-dump/analysis.report

  # wait for Verrazzano install to complete
  ./tests/e2e/config/scripts/wait-for-verrazzano-install.sh
  if [ $? -ne 0 ]; then
    exit 1
  fi
}

start_installation