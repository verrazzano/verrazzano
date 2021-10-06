#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/config.sh

INGRESS_TYPE=$(get_config_value ".ingress.type")

CONFIG_DIR=$SCRIPT_DIR/config
TMP_DIR=$(mktemp -d)

function install_istio()
{
    ISTIO_CHART_DIR=${CHARTS_DIR}/istio

    IMAGE_PULL_SECRETS_ARGUMENT=""
    if [ ${REGISTRY_SECRET_EXISTS} == "TRUE" ]; then
      IMAGE_PULL_SECRETS_ARGUMENT=" --set global.imagePullSecrets[0]=${GLOBAL_IMAGE_PULL_SECRET}"
    fi

    # We just need to build the ISTIO_HUB_OVERRIDE once, using any chart that will reveal it
    ISTIO_HUB_OVERRIDE=""
    if [ -n "${REGISTRY}" ]; then
      local repository=$(build_component_repo_name istio istiod)
      ISTIO_HUB_OVERRIDE="--set global.hub=${REGISTRY}/${repository} "
    fi

    if ! is_chart_deployed istio-base istio-system ${ISTIO_CHART_DIR}/base ; then
      helm_install_retry istio-base ${ISTIO_CHART_DIR}/base istio-system \
        -f $VZ_OVERRIDES_DIR/istio-values.yaml \
        ${IMAGE_PULL_SECRETS_ARGUMENT} \
        || return $?
    fi

    if ! is_chart_deployed istiod istio-system ${ISTIO_CHART_DIR}/istio-control/istio-discovery ; then
      local chart_name=istiod
      build_image_overrides istio ${chart_name}
      helm_install_retry ${chart_name} ${ISTIO_CHART_DIR}/istio-control/istio-discovery istio-system \
        -f $VZ_OVERRIDES_DIR/istio-values.yaml \
        ${HELM_IMAGE_ARGS} \
        ${ISTIO_HUB_OVERRIDE} \
        ${IMAGE_PULL_SECRETS_ARGUMENT} \
        || return $?
    fi

    log "Generate Istio ingress specific configuration"
    local EXTRA_INGRESS_ARGUMENTS=""
    EXTRA_INGRESS_ARGUMENTS=$(get_istio_helm_args_from_config)
    EXTRA_INGRESS_ARGUMENTS="$EXTRA_INGRESS_ARGUMENTS --set gateways.istio-ingressgateway.type=${INGRESS_TYPE}"

    if ! is_chart_deployed istio-ingress istio-system ${ISTIO_CHART_DIR}/gateways/istio-ingress ; then
      local chart_name=istio-ingress
      build_image_overrides istio ${chart_name}
      helm_install_retry ${chart_name} ${ISTIO_CHART_DIR}/gateways/istio-ingress istio-system \
        -f $VZ_OVERRIDES_DIR/istio-values.yaml \
        ${HELM_IMAGE_ARGS} \
        ${ISTIO_HUB_OVERRIDE} \
        ${EXTRA_INGRESS_ARGUMENTS} \
        ${IMAGE_PULL_SECRETS_ARGUMENT} \
        || return $?
    fi

    if ! is_chart_deployed istio-egress istio-system ${ISTIO_CHART_DIR}/gateways/istio-egress ; then
      local chart_name=istio-egress
      build_image_overrides istio ${chart_name}
      helm_install_retry ${chart_name} ${ISTIO_CHART_DIR}/gateways/istio-egress istio-system \
        -f $VZ_OVERRIDES_DIR/istio-values.yaml \
        ${HELM_IMAGE_ARGS} \
        ${ISTIO_HUB_OVERRIDE} \
        ${IMAGE_PULL_SECRETS_ARGUMENT} \
        || return $?
    fi

    if ! is_chart_deployed istiocoredns istio-system ${ISTIO_CHART_DIR}/istiocoredns ; then
      local chart_name=istiocoredns
      build_image_overrides istio ${chart_name}
      helm_install_retry ${chart_name} ${ISTIO_CHART_DIR}/istiocoredns istio-system \
        -f $VZ_OVERRIDES_DIR/istio-values.yaml \
        ${HELM_IMAGE_ARGS} \
        ${IMAGE_PULL_SECRETS_ARGUMENT} \
        || return $?
    fi

    log "Setting Istio global mesh policy to STRICT mode"
    kubectl apply -f <(echo "
apiVersion: "security.istio.io/v1beta1"
kind: "PeerAuthentication"
metadata:
  name: "default"
  namespace: "istio-system"
spec:
  mtls:
    mode: STRICT
")

    log "Adding Istio server header network filter"
    kubectl apply -f <(echo "
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: server-header-filter
  namespace: istio-system
spec:
  configPatches:
    - applyTo: NETWORK_FILTER
      match:
        listener:
          filterChain:
            filter:
              name: envoy.filters.network.http_connection_manager
      patch:
        operation: MERGE
        value:
          typed_config:
            '@type': type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
            server_header_transformation: PASS_THROUGH
")
}


action "Installing Istio" install_istio || exit 1

