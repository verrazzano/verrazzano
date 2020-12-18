#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
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

  echo "Generating CA bundle for Istio"

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

  rm -rf $CERTS_OUT
  rm -f ./index.txt* serial serial.old

  return 0
}

function install_istio()
{
    log "Add istio helm repository"
    helm repo add istio.io https://storage.googleapis.com/istio-release/releases/${ISTIO_HELM_CHART_VERSION}/charts || return $?

    log "Fetch istio charts for istio and istio-init"
    helm fetch istio.io/istio --untar=true --untardir=$TMP_DIR || return $?
    helm fetch istio.io/istio-init --untar=true --untardir=$TMP_DIR || return $?

    EXTRA_ISTIO_ARGUMENTS=""
    if [ ${REGISTRY_SECRET_EXISTS} == "TRUE" ]; then
      EXTRA_ISTIO_ARGUMENTS=" --set global.imagePullSecrets[0]=${GLOBAL_IMAGE_PULL_SECRET}"
    fi

    log "Create helm template for installing istio CRDs"
    helm template istio-init ${TMP_DIR}/istio-init \
        --namespace istio-system \
        --set global.hub=$GLOBAL_HUB_REPO \
        --set global.tag=$ISTIO_VERSION \
        ${EXTRA_ISTIO_ARGUMENTS} \
        > ${TMP_DIR}/istio-crds.yaml || return $?

    log "Generate cluster specific configuration"
    local EXTRA_HELM_ARGUMENTS=""
    if [ ${REGISTRY_SECRET_EXISTS} == "TRUE" ]; then
      EXTRA_HELM_ARGUMENTS=" --set global.imagePullSecrets[0]=${GLOBAL_IMAGE_PULL_SECRET}"
    fi

    EXTRA_HELM_ARGUMENTS="$EXTRA_HELM_ARGUMENTS $(get_istio_helm_args_from_config)"

    log "Create helm template for installing istio proper"
    helm template istio ${TMP_DIR}/istio \
        --namespace istio-system \
        --set global.hub=$GLOBAL_HUB_REPO \
        --set global.tag=$ISTIO_VERSION \
        --set gateways.istio-ingressgateway.type="${INGRESS_TYPE}" \
        --set sidecarInjectorWebhook.rewriteAppHTTPProbe=true \
        --set grafana.enabled=true \
        --set grafana.image.repository=$GRAFANA_REPO \
        --set grafana.image.tag=$GRAFANA_TAG \
        --set prometheus.hub=$GLOBAL_HUB_REPO \
        --set prometheus.tag=$PROMETHEUS_TAG \
        --set istiocoredns.coreDNSImage=$ISTIO_CORE_DNS_IMAGE \
        --set istiocoredns.coreDNSTag=$ISTIO_CORE_DNS_TAG \
        --set istiocoredns.coreDNSPluginImage=$ISTIO_CORE_DNS_PLUGIN_IMAGE:$ISTIO_CORE_DNS_PLUGIN_TAG \
        --set gateways.istio-ingressgateway.ports[0].port=80 \
        --set gateways.istio-ingressgateway.ports[0].targetPort=80 \
        --set gateways.istio-ingressgateway.ports[0].name=http2 \
        --set gateways.istio-ingressgateway.ports[0].nodePort=31380 \
        --set gateways.istio-ingressgateway.ports[1].port=443 \
        --set gateways.istio-ingressgateway.ports[1].name=https \
        --set gateways.istio-ingressgateway.ports[1].nodePort=31390 \
        --values ${TMP_DIR}/istio/example-values/values-istio-multicluster-gateways.yaml \
        ${EXTRA_HELM_ARGUMENTS} \
        > ${TMP_DIR}/istio.yaml || return $?

    log "Change to use the Istio image for kubectl then install the istio CRDs"
    sed "s|/kubectl:|/istio_kubectl:|g" ${TMP_DIR}/istio-crds.yaml | kubectl apply -f - || return $?

    log "Wait for istio CRD creation jobs to complete"
    if ! kubectl -n istio-system wait --for=condition=complete job --all --timeout=300s ; then
      consoleerr "ERROR: Istio CRD creation failed"
      return 1
    fi

    log "Change to use the Istio image for kubectl then install istio proper"
    sed "s|/kubectl:|/istio_kubectl:|g" ${TMP_DIR}/istio.yaml | kubectl apply -f - || return $?

}

function update_coredns()
{
    local cluster_ip
    cluster_ip=$(kubectl get svc -n istio-system istiocoredns -o jsonpath={.spec.clusterIP})
    if [ $? -ne 0 ] ; then
      return 1
    fi

    # Update coredns configmap to include global section in data.
    # This update requires coredns be greater than 1.4.0
    sed -e "s#@CLUSTER_IP@#${cluster_ip}#g" $CONFIG_DIR/coredns-template.yaml \
       | kubectl apply -f - \
       || return 1

    return 0
}

function check_kube_version {
    local kubeVer=$(kubectl version -o json)
    log "------Begin Kubernetes Version Info----"
    log "$kubeVer"
    log "------End Kubernetes Version Info----"
    local servVer=$(echo $kubeVer | jq -r '.serverVersion.gitVersion')
    if [ "$servVer" == "null" ] || [ -z "$servVer" ]; then
        log "Could not retrieve Kubernetes server version"
        return 1
    fi

    local major=$(echo $kubeVer | jq -r '.serverVersion.major')
    local minor=$(echo $kubeVer | jq -r '.serverVersion.minor')
    local patch=$(echo $servVer | cut -d'.' -f 3)
    VER_ERROR_MSG="Kubernetes serverVersion $servVer must be greater than or equal to v1.16.8 and less than or equal to v1.18.*"
    if [ "$major" -ne 1 ] ; then
      log $VER_ERROR_MSG
      return 1
    fi
    if [ "$minor" -lt 16 ] || [ "$minor" -gt 18  ]; then
      log $VER_ERROR_MSG
      return 1
    fi
    if [ "$minor" -eq 16 ] && [ "$patch" -lt 8  ]; then
      log $VER_ERROR_MSG
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

action "Checking Kubernetes version" check_kube_version || exit 1
action "Checking Helm version" check_helm_version || (error "Helm version must be v3.x! Your Helm version is: $(helm version --short)"; exit 1)

# Wait for all cluster nodes to exist, and then to be ready
action "Waiting for all Kubernetes nodes to exist in cluster" wait_for_nodes_to_exist || exit 1

log "Kubernetes nodes exist"
action "Waiting for all Kubernetes nodes to be ready" \
    kubectl wait --for=condition=ready nodes --all || exit 1

# Create istio-system namespace if it does not exist
if ! kubectl get namespace istio-system > /dev/null 2>&1 ; then
  action "Creating istio-system namespace" \
    kubectl create namespace istio-system || exit 1
fi

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
action "Updating CoreDNS configuration" update_coredns || exit 1

kubectl get pods -n istio-system
