#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname $0); pwd -P)

GHCR_USER="${1:-$GHCR_USER}"
GHCR_PASS="${2:-$GHCR_PASS}"
OCR_USER="${3:-$OCR_USER}"
OCR_PASS="${4:-$OCR_PASS}"

GHCR_SERVER="ghcr.io"
OCR_SERVER="container-registry.oracle.com"
NAMESPACE="bobs-books"
NAMESPACE_LABEL="verrazzano-domain"
GHCR_SECRET="github-packages"
OCR_SECRET="ocr"

if [ -z "${GHCR_USER}" ]; then
  echo "ERROR: GitHub Container Registry username required as first argument or GHCR_USER environment variable."
  exit 1
fi
if [ -z "${GHCR_PASS}" ]; then
  echo "ERROR: GitHub Container Registry password required as second argument or GHCR_PASS environment variable."
  exit 1
fi
if [ -z "${OCR_USER}" ]; then
  echo "ERROR: Oracle Container Registry username required as third argument or OCR_USER environment variable."
  exit 1
fi
if [ -z "${OCR_PASS}" ]; then
  echo "ERROR: Oracle Container Registry password required as fourth argument or OCR_PASS environment variable."
  exit 1
fi
if [ -z "${WEBLOGIC_PASS}" ]; then
  echo "ERROR: WebLogic administration password required as WEBLOGIC_PASS environment variable."
  exit 1
fi
if [ -z "${MYSQL_PASS}" ]; then
  echo "ERROR: Oracle container registry password required as MYSQL_PASS environment variable."
  exit 1
fi

echo "Installing Bob's Books OAM application."

status=$(kubectl get namespace ${NAMESPACE} -o jsonpath="{.status.phase}" 2>/dev/null)
if [ "${status}" == "Active" ]; then
  echo "Found namespace ${NAMESPACE}."
else
  echo "Create namespace ${NAMESPACE}."
  kubectl create namespace "${NAMESPACE}"
  if [ $? -ne 0 ]; then
    echo "ERROR: Failed to create namespace ${NAMESPACE}, exiting."
    exit 1
  fi
fi

label=$(kubectl get namespace ${NAMESPACE} -o jsonpath="{.metadata.labels.${NAMESPACE_LABEL}}" 2>/dev/null)
if [ "${label}" == "true" ]; then
  echo "Namespace ${NAMESPACE} is already labeled."
else
  echo "Create label ${NAMESPACE_LABEL} on namespace ${NAMESPACE}."
  kubectl label namespaces "${NAMESPACE}" "${NAMESPACE_LABEL}=true"
  if [ $? -ne 0 ]; then
    echo "ERROR: Failed to label namespace ${NAMESPACE}, exiting."
    exit 1
  fi
fi

_create_image_pull_secret() {
  local secret_name=$1
  local server=$2
  local username=$3
  local password=$4

  if [ "${skip_secrets:-false}" != "true" ]; then
    echo "Create ${secret_name} image pull secret."
    kubectl delete secret "${secret_name}" -n "${NAMESPACE}" &>/dev/null
    kubectl create secret docker-registry "${secret_name}" -n "${NAMESPACE}" \
      --docker-server="${server}" \
      --docker-username="${username}" \
      --docker-password="${password}"
    if [ $? -ne 0 ]; then
      echo "ERROR: Failed to create image pull secret. Listing secrets."
      kubectl get secret "${secret_name}" -n "${NAMESPACE}"
      exit 1
    fi
  fi
}

_create_image_pull_secret ${GHCR_SECRET} ${GHCR_SERVER} ${GHCR_USER} ${GHCR_PASS}
_create_image_pull_secret ${OCR_SECRET} ${OCR_SERVER} ${OCR_USER} ${OCR_PASS}

_create_generic_secret() {
  local secret_name=$1
  local from_literals=$2
  local label=$3

  if [ "${skip_secrets:-false}" != "true" ]; then
    echo "Create ${secret_name} generic secret."
    kubectl delete secret "${secret_name}" -n "${NAMESPACE}" &>/dev/null
    kubectl create secret generic "${secret_name}" ${from_literals} -n ${NAMESPACE}
    if [ $? -ne 0 ]; then
      echo "ERROR: Failed to create generic secret. Listing secrets."
      kubectl get secret "${secret_name}" -n "${NAMESPACE}"
      exit 1
    fi

    if [ "${label}" != "" ]; then
      kubectl label secret "${secret_name}" "${label}" -n ${NAMESPACE}
    fi
    if [ $? -ne 0 ]; then
      echo "ERROR: Failed to label generic secret. Listing secrets."
      kubectl get secret "${secret_name}" -n "${NAMESPACE}"
      exit 1
    fi
  fi
}

_create_generic_secret bobbys-front-end-weblogic-credentials "--from-literal=password=${WEBLOGIC_PASS} --from-literal=username=weblogic"
_create_generic_secret bobbys-front-end-runtime-encrypt-secret "--from-literal=password=${WEBLOGIC_PASS}" "weblogic.domainUID=bobbys-front-end"
_create_generic_secret bobs-bookstore-weblogic-credentials "--from-literal=password=${WEBLOGIC_PASS} --from-literal=username=weblogic"
_create_generic_secret bobs-bookstore-runtime-encrypt-secret "--from-literal=password=${WEBLOGIC_PASS}" "weblogic.domainUID=bobs-bookstore"
_create_generic_secret mysql-credentials "--from-literal=username=books --from-literal=password=${MYSQL_PASS} --from-literal=url=jdbc:mysql://mysql.${NAMESPACE}.svc.cluster.local:3306/books"

helm repo add coherence https://oracle.github.io/coherence-operator/charts
helm repo update
helm upgrade --install coherence-operator coherence/coherence-operator \
  --namespace ${NAMESPACE} \
  --version 2.1.1 \
  --set serviceAccount=coherence-operator

echo "Wait for Coherence pod to be ready."
attempt=1
while true; do
  kubectl wait --for=condition=ready pod --selector='app=coherence-operator' -n ${NAMESPACE} --timeout 15s
  if [ $? -eq 0 ]; then
    echo "Coherence pod found ready on attempt ${attempt}."
    break
  elif [ ${attempt} -eq 1 ]; then
    echo "No Coherence pods found ready on initial attempt. Retrying after delay."
  elif [ ${attempt} -ge 30 ]; then
    echo "ERROR: No Coherence pod found ready after ${attempt} attempts. Listing pods."
    kubectl get pods -n "${NAMESPACE}"
    echo "ERROR: Exiting."
    exit 1
  fi
  attempt=$(($attempt + 1))
  sleep 1
done

kubectl apply -f ${SCRIPT_DIR}/
if [ $? -ne 0 ]; then
  echo "ERROR: Failed to apply YAML files, exiting."
  exit 1
fi

_wait_for_success() {
  local cmd=$1

  attempt=1
  while true; do
    eval ${cmd}
    if [ $? -eq 0 ]; then
      echo "Success on attempt ${attempt}."
      break
    elif [ ${attempt} -eq 1 ]; then
      echo "No success on initial attempt. Retrying after delay."
    elif [ ${attempt} -ge 30 ]; then
      echo "ERROR: No success after ${attempt} attempts. Listing all."
      kubectl get all -n "${NAMESPACE}"
      echo "ERROR: Exiting."
      exit 1
    fi
    attempt=$(($attempt + 1))
    sleep 10
  done
}

BOBBY_POD="bobbys-front-end-managed-server1"
echo "Wait for ${BOBBY_POD} pod to start."
_wait_for_success "kubectl get pod ${BOBBY_POD} -n ${NAMESPACE} &> /dev/null"
echo "Wait for ${BOBBY_POD} pod to be ready."
_wait_for_success "kubectl wait --for=condition=ready pod ${BOBBY_POD} -n ${NAMESPACE} --timeout 15s"

BOB_POD="bobs-bookstore-managed-server1"
echo "Wait for ${BOB_POD} pod to start."
_wait_for_success "kubectl get pod ${BOB_POD} -n ${NAMESPACE} &> /dev/null"
echo "Wait for ${BOB_POD} pod to be ready."
_wait_for_success "kubectl wait --for=condition=ready pod ${BOB_POD} -n ${NAMESPACE} --timeout 15s"

ROBERT_SERVICE="robert-helidon"
echo "Wait for service ${ROBERT_SERVICE} to be available."
_wait_for_success "kubectl get service ${ROBERT_SERVICE} -n ${NAMESPACE} &> /dev/null"

_determine_endpoint() {
  echo "Determine service endpoint."
  attempt=1
  while true; do
    EXTERNAL_IP=$(kubectl get service -n "istio-system" "istio-ingressgateway" -o jsonpath={.status.loadBalancer.ingress[0].ip})
    if [[ ! -z "${EXTERNAL_IP}" ]]; then
      echo "Application endpoint found on attempt ${attempt}, external IP \"${EXTERNAL_IP}\"."
      break
    elif [ ${attempt} -eq 1 ]; then
      echo "No application endpoints found on initial attempt. Retrying after delay."
    elif [ ${attempt} -ge 60 ]; then
      echo "ERROR: No application endpoints found ater ${attempt} attempts. Listing services and exiting."
      kubectl get services -n "${NAMESPACE}"
      exit 1
    fi
    attempt=$(($attempt + 1))
    sleep 1
  done
}

_determine_endpoint
ROBERT_URL="http://${EXTERNAL_IP}/"
BOBBY_URL="http://${EXTERNAL_IP}/bobbys-front-end"
BOBS_ORDERS_URL="http://${EXTERNAL_IP}/bobs-bookstore-order-manager/orders"

echo "To access Robert's Books, navigate to ${ROBERT_URL}"
echo "To access Bobby's Books, navigate to ${BOBBY_URL}"
echo "To access the backend order manager, navigate to ${BOBS_ORDERS_URL}"

echo "Installation of Bob's Books OAM example is complete."
