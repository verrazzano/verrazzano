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

function install_application_operator {
  if is_chart_deployed verrazzano-application-operator ${VERRAZZANO_NS} $VZ_CHARTS_DIR/verrazzano-application-operator ; then
    return 0
  fi

  IMAGE_PULL_SECRETS_ARGUMENT=""
  if [ ${REGISTRY_SECRET_EXISTS} == "TRUE" ]; then
    IMAGE_PULL_SECRETS_ARGUMENT=" --set global.imagePullSecrets[0]=${GLOBAL_IMAGE_PULL_SECRET}"
  fi

  # Used to override the app operator image in development environment
  APP_OPERATOR_IMAGE_ARG=""
  if [ -n "${APP_OPERATOR_IMAGE}" ]; then
    APP_OPERATOR_IMAGE_ARG=" --set image=${APP_OPERATOR_IMAGE}"
  fi

  local chart_name=verrazzano-application-operator
  build_image_overrides verrazzano-application-operator ${chart_name}

  helm_install_retry ${chart_name} ${VZ_CHARTS_DIR}/verrazzano-application-operator ${VERRAZZANO_NS} \
    ${HELM_IMAGE_ARGS} \
    ${IMAGE_PULL_SECRETS_ARGUMENT} \
    ${APP_OPERATOR_IMAGE_ARG} || return $?
  if [ $? -ne 0 ]; then
    error "Failed to install Verrazzano Kubernetes application operator."
    return 1
  fi
}

function install {
  log "Creating ${VERRAZZANO_NS} namespace"
  if ! kubectl get namespace "${VERRAZZANO_NS}" > /dev/null 2>&1 ; then
    kubectl create namespace "${VERRAZZANO_NS}"
  fi

  log "Installing Verrazzano application operator"
  install_application_operator

  log "Installing Verrazzano OAM extensions"
  log $(kubectl apply -f ${PROJ_DIR}/deploy)
  kubectl apply -f ${PROJ_DIR}/deploy
  if [ $? -ne 0 ]; then
    error "Failed to install Verrazzano OAM extensions"
    return 1
  fi

  # Update the image name and ca bundle in the Verrazzano deployment file
  log "Updating Verrazzano application operator image name to ${VERRAZZANO_APP_OP_IMAGE}"
  mkdir -p ${BUILD_DEPLOY}
  cat ${PROJ_DIR}/deploy/verrazzano.yaml_template | sed -e "s|IMAGE_NAME|${VERRAZZANO_APP_OP_IMAGE}|g" > ${BUILD_DEPLOY}/verrazzano.yaml

  log "Installing Verrazzano application operator"
  kubectl apply -f ${BUILD_DEPLOY}/verrazzano.yaml
  if [ $? -ne 0 ]; then
    error "Failed to install Verrazzano application operator"
    return 1
  fi
}

action "Installing Verrazzano application operator" install || fail "Failed to install the Verrazzano OAM operator. \n file: $(cat /home/opc/go/src/github.com/verrazzano/verrazzano/application-operator/installer/build/logs/install.sh.log)"
