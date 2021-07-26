#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/config.sh

CONFIG_DIR=$SCRIPT_DIR/config

VERRAZZANO_NS=verrazzano-system
VERRAZZANO_MC=verrazzano-mc
MONITORING_NS=monitoring

ENV_NAME=$(get_config_value ".environmentName")

INGRESS_TYPE=$(get_config_value ".ingress.type")
INGRESS_IP=$(get_verrazzano_ingress_ip)
if [ -n "${INGRESS_IP:-}" ]; then
  log "Found ingress address ${INGRESS_IP}"
else
  fail "Failed to find ingress address."
fi

DNS_TYPE=$(get_config_value ".dns.type")
DNS_SUFFIX=$(get_dns_suffix ${INGRESS_IP})

# Check if the nginx ingress ports are accessible
function check_ingress_ports() {
  exitvalue=0
  if [ ${INGRESS_TYPE} == "LoadBalancer" ] && [ $DNS_TYPE != "external" ]; then
    # Get the ports from the ingress
    PORTS=$(kubectl get services -n ingress-nginx ingress-controller-ingress-nginx-controller -o=custom-columns=PORT:.spec.ports[*].name --no-headers)
    IFS=',' read -r -a port_array <<< "$PORTS"

    index=0
    for element in "${port_array[@]}"
    do
      # For each get the port, nodePort and targetPort
      RESP=$(kubectl get services -n ingress-nginx ingress-controller-ingress-nginx-controller -o=custom-columns=PORT:.spec.ports[$index].port,NODEPORT:.spec.ports[$index].nodePort,TARGETPORT:.spec.ports[$index].targetPort --no-headers)
      ((index++))

      IFS=' ' read -r -a vals <<< "$RESP"
      PORT="${vals[0]}"
      NODEPORT="${vals[1]}"
      TARGETPORT="${vals[2]}"

      # Attempt to access the port on the $INGRESS_IP
      if [ $TARGETPORT == "https" ]; then
        ARGS=(-k https://$INGRESS_IP:$PORT)
        call_curl 0 response http_code ARGS
      else
        ARGS=(http://$INGRESS_IP:$PORT)
        call_curl 0 response http_code ARGS
      fi

      # Check the result of the curl call
      if [ $? -eq 0 ]; then
        log "Port $PORT is accessible on ingress address $INGRESS_IP.  Note that '404 page not found' is an expected response."
      else
        log "ERROR: Port $PORT is NOT accessible on ingress address $INGRESS_IP!  Check that security lists include an ingress rule for the node port $NODEPORT."
        log "See install README for details(https://github.com/verrazzano/verrazzano/operator/blob/master/install/README.md#1-oke-missing-security-list-ingress-rules)."
        exitvalue=1
      fi
    done
  fi
  return $exitvalue
}

action "Checking ingress ports" check_ingress_ports || fail "ERROR: Failed ingress port check."

set -eu

function install_verrazzano()
{
  EXTRA_V8O_ARGUMENTS=$(get_verrazzano_helm_args_from_config)
  if [ ${REGISTRY_SECRET_EXISTS} == "TRUE" ]; then
    EXTRA_V8O_ARGUMENTS="${EXTRA_V8O_ARGUMENTS} --set global.imagePullSecrets[0]=${GLOBAL_IMAGE_PULL_SECRET}"
  fi

  local profile=$(get_install_profile)
  if [ ! -f "${VZ_CHARTS_DIR}/verrazzano/values.${profile}.yaml" ]; then
    error "The file ${VZ_CHARTS_DIR}/verrazzano/values.${profile}.yaml does not exist"
    exit 1
  fi
  local PROFILE_VALUES_OVERRIDE=" -f ${VZ_CHARTS_DIR}/verrazzano/values.${profile}.yaml"

  # Get the endpoint for the Kubernetes API server.  The endpoint returned has the format of IP:PORT
  local ENDPOINT=$(kubectl get endpoints --namespace default kubernetes --no-headers | awk '{ print $2}')
  local ENDPOINT_ARRAY=(${ENDPOINT//:/ })

  local DNS_TYPE=$(get_config_value ".dns.type")
  local EXTERNAL_DNS_ENABLED=false
  if [ "$DNS_TYPE" == "oci" ]; then
    EXTERNAL_DNS_ENABLED=true
  fi

  if [ "$DNS_TYPE" == "wildcard" ]; then
    EXTRA_V8O_ARGUMENTS="${EXTRA_V8O_ARGUMENTS} --set dns.wildcard.domain=$(get_config_value ".dns.wildcard.domain")"
  fi

  if ! is_chart_deployed verrazzano ${VERRAZZANO_NS} ${VZ_CHARTS_DIR}/verrazzano ; then
    local chart_name=verrazzano
    build_image_overrides verrazzano ${chart_name}
    local image_args=${HELM_IMAGE_ARGS}
    build_image_overrides monitoring-init-images monitoring-init-images
    HELM_IMAGE_ARGS="${HELM_IMAGE_ARGS} ${image_args}"

    helm \
        upgrade --install ${chart_name} \
        ${VZ_CHARTS_DIR}/verrazzano \
        --namespace ${VERRAZZANO_NS} \
        --set image.pullPolicy=IfNotPresent \
        --set config.envName=${ENV_NAME} \
        --set config.dnsSuffix=${DNS_SUFFIX} \
        --set config.enableMonitoringStorage=true \
        --set kubernetes.service.endpoint.ip=${ENDPOINT_ARRAY[0]} \
        --set kubernetes.service.endpoint.port=${ENDPOINT_ARRAY[1]} \
        --set externaldns.enabled=${EXTERNAL_DNS_ENABLED} \
        --set keycloak.enabled=$(is_keycloak_enabled) \
        --set rancher.enabled=$(is_rancher_enabled) \
        --set api.proxy.OidcProviderHost=keycloak.${ENV_NAME}.${DNS_SUFFIX} \
        --set api.proxy.OidcProviderHostInCluster=keycloak-http.keycloak.svc.cluster.local \
        $(get_fluentd_extra_volume_mounts) \
        ${HELM_IMAGE_ARGS} \
        ${PROFILE_VALUES_OVERRIDE} \
        ${EXTRA_V8O_ARGUMENTS} || return $?
  fi
  log "Waiting for the verrazzano-operator pod in ${VERRAZZANO_NS} to reach Ready state"
  kubectl  wait -l app=verrazzano-operator --for=condition=Ready pod -n verrazzano-system

  log "Verifying that needed secrets are created"
  retries=0
  until [ "$retries" -ge 60 ]
  do
      kubectl get secret -n ${VERRAZZANO_NS} verrazzano | grep verrazzano && break
      retries=$(($retries+1))
      sleep 5
  done
  if ! kubectl get secret --namespace ${VERRAZZANO_NS} verrazzano ; then
      error "ERROR: failed creating verrazzano secret"
      exit 1
  fi
  log "Verrazzano install completed"
}

function install_oam_operator {
  if is_chart_deployed oam-kubernetes-runtime ${VERRAZZANO_NS} ${CHARTS_DIR}/oam-kubernetes-runtime ; then
    return 0
  fi

  IMAGE_PULL_SECRETS_ARGUMENT=""
  if [ ${REGISTRY_SECRET_EXISTS} == "TRUE" ]; then
    IMAGE_PULL_SECRETS_ARGUMENT=" --set imagePullSecrets[0].name=${GLOBAL_IMAGE_PULL_SECRET}"
  fi

  log "Install OAM Kubernetes operator"
  local chart_name=oam-kubernetes-runtime
  build_image_overrides oam-kubernetes-runtime ${chart_name}

  helm upgrade --install --wait ${chart_name} \
    ${CHARTS_DIR}/oam-kubernetes-runtime \
    --namespace "${VERRAZZANO_NS}" \
    ${HELM_IMAGE_ARGS} \
    ${IMAGE_PULL_SECRETS_ARGUMENT} \
    || return $?
  if [ $? -ne 0 ]; then
    error "Failed to install OAM Kubernetes operator."
    return 1
  fi
}

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

  log "Install Verrazzano Kubernetes application operator"
  local chart_name=verrazzano-application-operator
  build_image_overrides verrazzano-application-operator ${chart_name}

  helm upgrade --install --wait ${chart_name} \
    $VZ_CHARTS_DIR/verrazzano-application-operator \
    --namespace "${VERRAZZANO_NS}" \
    ${HELM_IMAGE_ARGS} \
    ${IMAGE_PULL_SECRETS_ARGUMENT} \
    ${APP_OPERATOR_IMAGE_ARG} || return $?
  if [ $? -ne 0 ]; then
    error "Failed to install Verrazzano Kubernetes application operator."
    return 1
  fi
}

function install_weblogic_operator {

  if ! kubectl get serviceaccount weblogic-operator-sa -n ${VERRAZZANO_NS} > /dev/null 2>&1; then
    log "Create WebLogic Kubernetes operator service account"
    kubectl create serviceaccount -n "${VERRAZZANO_NS}" weblogic-operator-sa
    if [ $? -ne 0 ]; then
      error "Failed to create WebLogic Kubernetes operator service account."
      return 1
    fi
  fi

  if is_chart_deployed weblogic-operator ${VERRAZZANO_NS} ${CHARTS_DIR}/weblogic-operator ; then
    return 0
  fi

  IMAGE_PULL_SECRETS_ARGUMENT=""
  if [ ${REGISTRY_SECRET_EXISTS} == "TRUE" ]; then
    IMAGE_PULL_SECRETS_ARGUMENT=" --set imagePullSecrets[0].name=${GLOBAL_IMAGE_PULL_SECRET}"
  fi

  log "Install WebLogic Kubernetes operator"
  local chart_name=weblogic-operator
  build_image_overrides weblogic-operator ${chart_name}

  helm upgrade --install --wait ${chart_name} \
    ${CHARTS_DIR}/weblogic-operator \
    --namespace "${VERRAZZANO_NS}" \
    -f $VZ_OVERRIDES_DIR/weblogic-values.yaml \
    --set serviceAccount=weblogic-operator-sa \
    --set domainNamespaceSelectionStrategy=LabelSelector \
    --set domainNamespaceLabelSelector=verrazzano-managed \
    --set enableClusterRoleBinding=true \
    ${HELM_IMAGE_ARGS} \
    ${IMAGE_PULL_SECRETS_ARGUMENT} \
    || return $?
  if [ $? -ne 0 ]; then
    error "Failed to install WebLogic Kubernetes operator."
    return 1
  fi
}

function install_coherence_operator {
  if is_chart_deployed coherence-operator ${VERRAZZANO_NS} ${CHARTS_DIR}/coherence-operator ; then
    return 0
  fi

  IMAGE_PULL_SECRETS_ARGUMENT=""
  if [ ${REGISTRY_SECRET_EXISTS} == "TRUE" ]; then
    IMAGE_PULL_SECRETS_ARGUMENT=" --set imagePullSecrets[0].name=${GLOBAL_IMAGE_PULL_SECRET}"
  fi

  log "Install the Coherence Kubernetes operator"
  local chart_name=coherence-operator
  build_image_overrides coherence-operator ${chart_name}

  helm upgrade --install --wait ${chart_name} \
    ${CHARTS_DIR}/coherence-operator \
    --namespace "${VERRAZZANO_NS}" \
    -f $VZ_OVERRIDES_DIR/coherence-values.yaml \
    ${HELM_IMAGE_ARGS} \
    ${IMAGE_PULL_SECRETS_ARGUMENT} \
    || return $?
  if [ $? -ne 0 ]; then
    error "Failed to install the Coherence Kubernetes operator."
    return 1
  fi
}

# Set environment variable for checking if optional imagePullSecret was provided
REGISTRY_SECRET_EXISTS=$(check_registry_secret_exists)

if ! kubectl get namespace ${VERRAZZANO_NS} ; then
  action "Creating ${VERRAZZANO_NS} namespace" kubectl create namespace ${VERRAZZANO_NS} || exit 1
fi

log "Adding label needed by network policies to ${VERRAZZANO_NS} namespace"
kubectl label namespace ${VERRAZZANO_NS} "verrazzano.io/namespace=${VERRAZZANO_NS}" --overwrite

log "Adding label for enabling istio sidecar injection by default to ${VERRAZZANO_NS} namespace"
kubectl label namespace ${VERRAZZANO_NS} "istio-injection=enabled" --overwrite

if ! kubectl get namespace ${VERRAZZANO_MC} ; then
  action "Creating ${VERRAZZANO_MC} namespace" kubectl create namespace ${VERRAZZANO_MC} || exit 1
fi

if [ $(is_verrazzano_operator_enabled) == "true" ]; then
  if ! kubectl get namespace ${MONITORING_NS} ; then
    action "Creating ${MONITORING_NS} namespace" kubectl create namespace ${MONITORING_NS} || exit 1
  fi

  log "Adding label needed by network policies to ${MONITORING_NS} namespace"
  kubectl label namespace ${MONITORING_NS} "verrazzano.io/namespace=${MONITORING_NS}" --overwrite
fi

# If Keycloak is being installed, create the Keycloak namespace if it doesn't exist so we can apply network policies
if [ $(is_keycloak_enabled) == "true" ] && ! kubectl get namespace keycloak ; then
  action "Creating keycloak namespace" kubectl create namespace keycloak || exit 1
  # Label the keycloak namespace so that we istio injection is enabled
  log "Adding label needed for istio sidecar injection to keycloak namespace"
  kubectl label namespace keycloak "istio-injection=enabled" --overwrite
  # Label the keycloak namespace so that we can apply network policies
  log "Adding label needed by network policies to keycloak namespace"
  kubectl label namespace keycloak "verrazzano.io/namespace=keycloak" --overwrite
fi

if [ "${REGISTRY_SECRET_EXISTS}" == "TRUE" ]; then
  if ! kubectl get secret ${GLOBAL_IMAGE_PULL_SECRET} -n ${VERRAZZANO_NS} > /dev/null 2>&1 ; then
    action "Copying ${GLOBAL_IMAGE_PULL_SECRET} secret to ${VERRAZZANO_NS} namespace" \
        copy_registry_secret "${VERRAZZANO_NS}"
  fi
  if ! kubectl get secret ${GLOBAL_IMAGE_PULL_SECRET} -n ${MONITORING_NS} > /dev/null 2>&1 ; then
    action "Copying ${GLOBAL_IMAGE_PULL_SECRET} secret to ${MONITORING_NS} namespace" \
        copy_registry_secret "${MONITORING_NS}"
  fi
fi

if [ $(is_verrazzano_operator_enabled) == "true" ]; then
  action "Installing Verrazzano system components" install_verrazzano || exit 1
fi
action "Installing Coherence Kubernetes operator" install_coherence_operator || exit 1
action "Installing WebLogic Kubernetes operator" install_weblogic_operator || exit 1
action "Installing OAM Kubernetes operator" install_oam_operator || exit 1
action "Installing Verrazzano Application Kubernetes operator" install_application_operator || exit 1
