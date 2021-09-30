#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
PROJ_DIR=$(cd $(dirname "$0"); cd ../..; pwd -P)
BUILD_DEPLOY=${PROJ_DIR}/build/deploy
CERTS_OUT=${PROJ_DIR}/build/webhook-certs

VERRAZZANO_NS=verrazzano-system

if [ -z "${VERRAZZANO_APP_OP_IMAGE:-}" ] ; then
  export VERRAZZANO_APP_OP_IMAGE=$(cat $PROJ_DIR/deploy/application-operator.txt)
  if [ -z "${VERRAZZANO_APP_OP_IMAGE:-}" ] ; then
    error "Failed to determine Verrazzano application operator image."
    return 1
  fi
fi

. $SCRIPT_DIR/common.sh

function install {
  log "Creating ${VERRAZZANO_NS} namespace"
  if ! kubectl get namespace "${VERRAZZANO_NS}" > /dev/null 2>&1 ; then
    kubectl create namespace "${VERRAZZANO_NS}"
  fi

  log "Installing Verrazzano application operator"
  local chart_name=verrazzano-application-operator
  HELM_IMAGE_ARGS="${HELM_IMAGE_ARGS} --set image=${VERRAZZANO_APP_OP_IMAGE}"

  helm_install_retry ${chart_name} ${VZ_CHARTS_DIR}/verrazzano-application-operator ${VERRAZZANO_NS} \
    ${HELM_IMAGE_ARGS} \
    ${IMAGE_PULL_SECRETS_ARGUMENT} \
    ${APP_OPERATOR_IMAGE_ARG} || return $?
  if [ $? -ne 0 ]; then
    error "Failed to install Verrazzano Kubernetes application operator."
    return 1
  fi

    log "Installing Verrazzano OAM extensions"
  kubectl apply -f ${PROJ_DIR}/deploy
  if [ $? -ne 0 ]; then
    error "Failed to install Verrazzano OAM extensions"
    return 1
  fi

}

action "Installing Verrazzano application operator" install || fail "Failed to install the Verrazzano OAM operator."
