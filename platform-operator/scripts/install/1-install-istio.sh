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
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

set -ueo pipefail

function create_secret {
  CERTS_OUT=$SCRIPT_DIR/build/istio-certs

  rm -rf $CERTS_OUT || true
  rm -f ./index.txt* serial serial.old || true

  mkdir -p $CERTS_OUT
  touch ./index.txt
  echo 1000 > ./serial

  if ! kubectl get secret cacerts -n istio-system > /dev/null 2>&1; then
    log "Generating CA bundle for Istio"

    # Create the private key for the root CA
    openssl genrsa -out $CERTS_OUT/root-key.pem 4096 || return $?

    # Generate a root CA with the private key
    openssl req -config $CONFIG_DIR/istio_root_ca_config.txt -key $CERTS_OUT/root-key.pem -new -x509 -days 7300 -sha256 -extensions v3_ca -out $CERTS_OUT/root-cert.pem || return $?

    # Create the private key for the intermediate CA
    openssl genrsa -out $CERTS_OUT/ca-key.pem 4096 || return $?

    # Generate certificate signing request (CSR)
    openssl req -config $CONFIG_DIR/istio_intermediate_ca_config.txt -new -sha256 -key $CERTS_OUT/ca-key.pem -out $CERTS_OUT/intermediate-csr.pem || return $?

    # create intermediate cert using the root CA
    openssl ca -batch -config $CONFIG_DIR/istio_root_ca_config.txt -extensions v3_intermediate_ca -days 3650 -notext -md sha256 \
        -keyfile $CERTS_OUT/root-key.pem \
        -cert $CERTS_OUT/root-cert.pem \
        -in $CERTS_OUT/intermediate-csr.pem \
        -out $CERTS_OUT/ca-cert.pem \
        -outdir $CERTS_OUT || return $?

    # Create certificate chain file
    cat $CERTS_OUT/ca-cert.pem $CERTS_OUT/root-cert.pem > $CERTS_OUT/cert-chain.pem || return $?

    kubectl create secret generic cacerts -n istio-system \
        --from-file=$CERTS_OUT/ca-cert.pem \
        --from-file=$CERTS_OUT/ca-key.pem  \
        --from-file=$CERTS_OUT/root-cert.pem \
        --from-file=$CERTS_OUT/cert-chain.pem || return $?
  else
    log "Istio CA Certs bundle and secret already created"
  fi

  rm -rf $CERTS_OUT
  rm -f ./index.txt* serial serial.old

  return 0
}

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
      log "Installing istio-system/istio-base"
      helm upgrade istio-base ${ISTIO_CHART_DIR}/base \
        --install \
        --namespace istio-system \
        --wait \
        -f $VZ_OVERRIDES_DIR/istio-values.yaml \
        ${IMAGE_PULL_SECRETS_ARGUMENT} \
        || return $?
    fi

    if ! is_chart_deployed istiod istio-system ${ISTIO_CHART_DIR}/istio-control/istio-discovery ; then
      local chart_name=istiod
      build_image_overrides istio ${chart_name}
      helm_install-retry ${chart_name} ${ISTIO_CHART_DIR}/istio-control/istio-discovery istio-system \
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
      helm_install-retry ${chart_name} ${ISTIO_CHART_DIR}/gateways/istio-ingress istio-system \
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
      helm_install-retry ${chart_name} ${ISTIO_CHART_DIR}/gateways/istio-egress istio-system \
        -f $VZ_OVERRIDES_DIR/istio-values.yaml \
        ${HELM_IMAGE_ARGS} \
        ${ISTIO_HUB_OVERRIDE} \
        ${IMAGE_PULL_SECRETS_ARGUMENT} \
        || return $?
    fi

    if ! is_chart_deployed istiocoredns istio-system ${ISTIO_CHART_DIR}/istiocoredns ; then
      local chart_name=istiocoredns
      build_image_overrides istio ${chart_name}
      helm_install-retry ${chart_name} ${ISTIO_CHART_DIR}/istiocoredns istio-system \
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

function log_kube_version {
    local kubeVer=$(kubectl version -o json)
    log "------Begin Kubernetes Version Info----"
    log "$kubeVer"
    log "------End Kubernetes Version Info----"
    local servVer=$(echo $kubeVer | jq -r '.serverVersion.gitVersion')
    if [ "$servVer" == "null" ] || [ -z "$servVer" ]; then
        log "Could not retrieve Kubernetes server version"
        return 1
    fi
}

function check_helm_version {
    local helmVer=$(helm version --short | cut -d':' -f2 | tr -d " ")
    log "Helm version is $helmVer"
    local majorVer=$(echo $helmVer | cut -d'.' -f1)
    local minorVer=$(echo $helmVer | cut -d'.' -f2)
    if [ "$majorVer" != "v3" ]; then
        log "Helm major version is $majorVer, expected v3!"
        return 1
    fi
    return 0
}

function wait_for_nodes_to_exist {
    retries=0
    until kubectl get nodes | grep NAME; do
      retries=$(($retries+1))
      sleep 10
      if [ "$retries" -ge 30 ] ; then
        break
      fi
    done
    if [ "$retries" -ge 30 ] ; then
      log "Kubernetes nodes don't exist in cluster"
      return 1
    fi
}

action "Checking Kubernetes version" log_kube_version || exit 1
action "Checking Helm version" check_helm_version || (error "Helm version must be v3.x! Your Helm version is: $(helm version --short)"; exit 1)

# Wait for all cluster nodes to exist, and then to be ready
action "Waiting for all Kubernetes nodes to exist in cluster" wait_for_nodes_to_exist || exit 1

log "Kubernetes nodes exist"
action "Waiting for all Kubernetes nodes to be ready" \
    kubectl wait --for=condition=ready nodes --all || exit 1

# Label the kube-system namespace so that we can apply network policies
log "Adding label needed by network policies to kube-system namespace"
kubectl label namespace kube-system "verrazzano.io/namespace=kube-system" --overwrite

# Create istio-system namespace if it does not exist
if ! kubectl get namespace istio-system > /dev/null 2>&1 ; then
  action "Creating istio-system namespace" \
    kubectl create namespace istio-system || exit 1
fi

log "Adding label needed by network policies to istio-system namespace"
kubectl label namespace istio-system "verrazzano.io/namespace=istio-system" --overwrite

# Copy the optional global registry secret to the istio-system namespace for pulling OLCNE images in a OKE cluster
REGISTRY_SECRET_EXISTS=$(check_registry_secret_exists)
if [ "${REGISTRY_SECRET_EXISTS}" == "TRUE" ]; then
  if ! kubectl get secret ${GLOBAL_IMAGE_PULL_SECRET} -n istio-system > /dev/null 2>&1 ; then
    action "Copying ${GLOBAL_IMAGE_PULL_SECRET} secret to istio-system namespace" \
        copy_registry_secret "istio-system"
  fi
fi

# Create certificates and istio secret to hold certificates if we haven't already
if ! kubectl get secret cacerts -n istio-system > /dev/null 2>&1 ; then
  action "Generating Istio CA bundle" create_secret || exit 1
fi

action "Installing Istio" install_istio || exit 1

