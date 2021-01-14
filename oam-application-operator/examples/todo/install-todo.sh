#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname $0); pwd -P)

DOCKER_SVR="${1:-$OCIR_PHX_REPO}"
DOCKER_USR="${2:-$OCIR_CREDS_USR}"
DOCKER_PWD="${3:-$OCIR_CREDS_PSW}"

NAMESPACE="todo"
SECRET="tododomain-repo-credentials"
TODO_COMPONENT_FILE="${SCRIPT_DIR}/todo-comp.yaml"

# WLS_DOMAIN can get out of sync with value specified for domain in todo-comp.yaml
WLS_DOMAIN="tododomain"

if [ -z "${DOCKER_SVR}" ]; then
  echo "ERROR: Container registry required as first argument or OCIR_PHX_REPO environment variable."
  exit 1
fi
if [ -z "${DOCKER_USR}" ]; then
  echo "ERROR: Container registry username required as second argument or OCIR_CREDS_USR environment variable."
  exit 1
fi
if [ -z "${DOCKER_PWD}" ]; then
  echo "ERROR: Container registry username required as second argument or OCIR_CREDS_PSW environment variable."
  exit 1
fi
if [ -z "${WEBLOGIC_PSW}" ]; then
  #
  # WEBLOGIC_PSW also used as password for database/jdbc credentials
  #
  echo "ERROR: WebLogic administration password required as WEBLOGIC_PSW environment variable."
  exit 1
fi
if [ -z "${TODO_APP_IMAGE}" ]; then
  echo "ERROR: The image for the To-Do List application is required as TODO_APP_IMAGE environment variable."
  exit 1
fi

set -u

echo "Installing Todo OAM application."

status=$(kubectl get namespace ${NAMESPACE} -o jsonpath="{.status.phase}" 2> /dev/null)
if [ "${status}" == "Active" ]; then
  echo "Found namespace ${NAMESPACE}."
  echo "Ensuring namespace label exists."
  kubectl label --overwrite namespace ${NAMESPACE} verrazzano-domain=true
  if [ $? -ne 0 ]; then
      echo "ERROR: Failed to label namespace ${NAMESPACE}, exiting."
      exit 1
  fi
else
  echo "Create namespace ${NAMESPACE}."
  kubectl create namespace "${NAMESPACE}"
  if [ $? -ne 0 ]; then
      echo "ERROR: Failed to create namespace ${NAMESPACE}, exiting."
      exit 1
  fi
  kubectl label namespace ${NAMESPACE} verrazzano-domain=true
  if [ $? -ne 0 ]; then
      echo "ERROR: Failed to label namespace ${NAMESPACE}, exiting."
      exit 1
  fi
fi

echo "Create image pull secret."
if [ "${skip_secrets:-false}" != "true" ]; then
  kubectl get secret "${SECRET}" -n "${NAMESPACE}" &> /dev/null
  if [ $? -eq 0 ]; then
    echo "Delete existing secret."
    kubectl delete secret "${SECRET}" -n "${NAMESPACE}"
  fi
  kubectl create secret docker-registry "${SECRET}" -n "${NAMESPACE}"\
    --docker-server="${DOCKER_SVR}" \
    --docker-username="${DOCKER_USR}" \
    --docker-password="${DOCKER_PWD}"
  if [ $? -ne 0 ]; then
      echo "ERROR: Failed to create image pull secret. Listing secrets."
      kubectl get secret "${SECRET}" -n "${NAMESPACE}"
      exit 1
  fi
fi

function create_and_label_generic_secret() {
  typeset _secret="$1"
  typeset _username="$2"
  typeset _password="$3"
  typeset _label="$4"

  kubectl get secret "${_secret}" -n "${NAMESPACE}" &> /dev/null
  if [ $? -eq 0 ]; then
    echo "Delete existing secret '${_secret}."
    kubectl delete secret "${_secret}" -n "${NAMESPACE}"
  fi

  if [[ -z ${_username} ]] ; then
    kubectl create secret generic "${_secret}" -n "${NAMESPACE}" \
        --from-literal=password="$_password"
  else
    kubectl create secret generic "${_secret}" -n "${NAMESPACE}" \
        --from-literal=password="$_password" --from-literal=username="${_username}"
  fi

  if [ $? -ne 0 ]; then
      echo "ERROR: Failed to create secret. Listing secrets."
      kubectl get secret "${_secret}" -n "${NAMESPACE}"
      return 1
  fi

  if [[ -n ${_label} ]] ; then
    kubectl -n ${NAMESPACE} label secret ${_secret} ${_label}
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to create secret. Listing secrets."
        kubectl get secret "${_secret}" -n "${NAMESPACE}"
        return 1
    fi
  fi
}

echo "Create WebLogic secrets."
if [ "${skip_secrets:-false}" != "true" ]; then
  create_and_label_generic_secret tododomain-weblogic-credentials weblogic "${WEBLOGIC_PSW}" weblogic.domainUID=tododomain
  create_and_label_generic_secret tododomain-jdbc-tododb derek "${WEBLOGIC_PSW}" weblogic.domainUID=tododomain
  create_and_label_generic_secret tododomain-runtime-encrypt-secret "" "${WEBLOGIC_PSW}" weblogic.domainUID=tododomain
fi

echo "Substitute image name template in ${TODO_COMPONENT_FILE} as ${TODO_APP_IMAGE}"
sed -i '' -e "s|%TODO_APP_IMAGE%|${TODO_APP_IMAGE}|" ${TODO_COMPONENT_FILE}

echo "Apply application configuration."
kubectl apply -f ${SCRIPT_DIR}/
code=$?
if [ ${code} -ne 0 ]; then
  echo "ERROR: Applying application configuration failed: ${code}. Exiting."
  exit ${code}
fi

echo "Wait for at least one running workload pod."
attempt=1
while true; do
  kubectl -n "${NAMESPACE}" wait --for=condition=ready pods --selector="weblogic.domainName=${WLS_DOMAIN}" --timeout 15s
  if [ $? -eq 0 ]; then
    echo "Application pods found ready on attempt ${attempt}."
    break
  elif [ ${attempt} -eq 1 ]; then
    echo "No application pods found ready on initial attempt. Retrying after delay."
  elif [ ${attempt} -ge 60 ]; then
    echo "ERROR: No application pod found ready after ${attempt} attempts. Listing pods."
    kubectl get pods -n "${NAMESPACE}"
    echo "ERROR: Exiting."
    exit 1
  fi
  attempt=$(($attempt+1))
  sleep 5
done

echo "Installation of Todo OAM application successful."
