#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle Corporation and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh

if [ ${CLUSTER_TYPE} == "OKE" ]; then
  INGRESS_TYPE=LoadBalancer
elif [ ${CLUSTER_TYPE} == "KIND" ]; then
  INGRESS_TYPE=NodePort
fi

CONFIG_DIR=$SCRIPT_DIR/config
INSTALL_DIR=$(mktemp -d)
trap "rm -rf INSTALL_DIR" EXIT

set -u

function create_secret {
  CERTS_OUT=$SCRIPT_DIR/build/istio-certs

  rm -rf $CERTS_OUT
  rm -f ./index.txt* serial serial.old

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
    # Add istio helm repo
    helm repo add istio.io https://storage.googleapis.com/istio-release/releases/${ISTIO_VERSION}/charts || return $?

    # Fetch istio charts for istio and istio-init
    rm -rf "${INSTALL_DIR}"
    mkdir -p $INSTALL_DIR || return $?
    helm fetch istio.io/istio --untar=true --untardir=$INSTALL_DIR || return $?
    helm fetch istio.io/istio-init --untar=true --untardir=$INSTALL_DIR || return $?

    # Create helm template for installing istio CRDs
    helm template istio-init ${INSTALL_DIR}/istio-init \
        --namespace istio-system \
        --set global.hub=$OLCNE_IMAGE_REPO \
        --set global.tag=$ISTIO_VERSION \
        --set global.imagePullSecrets[0]=ocr \
        > ${INSTALL_DIR}/istio-crds.yaml || return $?

    # Create helm template for installing istio proper
    helm template istio ${INSTALL_DIR}/istio \
        --namespace istio-system \
        --set global.hub=$OLCNE_IMAGE_REPO \
        --set global.tag=$ISTIO_VERSION \
        --set global.imagePullSecrets[0]=ocr \
        --set gateways.istio-ingressgateway.type="${INGRESS_TYPE}" \
        --set sidecarInjectorWebhook.rewriteAppHTTPProbe=true \
        --set grafana.enabled=true \
        --set grafana.image.repository=$GRAFANA_REPO \
        --set grafana.image.tag=$GRAFANA_TAG \
        --set prometheus.hub=$OLCNE_IMAGE_REPO \
        --set prometheus.tag=v2.13.1 \
        --set istiocoredns.coreDNSImage=$ISTIO_CORE_DNS_IMAGE \
        --set istiocoredns.coreDNSTag=$ISTIO_CORE_DNS_TAG \
        --set istiocoredns.coreDNSPluginImage=$ISTIO_CORE_DNS_PLUGIN_IMAGE:$ISTIO_CORE_DNS_PLUGIN_TAG \
        --values ${INSTALL_DIR}/istio/example-values/values-istio-multicluster-gateways.yaml \
        > ${INSTALL_DIR}/istio.yaml || return $?

    # Change to use the OLCNE image for kubectl then install the istio CRDs
    sed "s|/kubectl:|/istio_kubectl:|g" ${INSTALL_DIR}/istio-crds.yaml | kubectl apply -f - || return $?

    # Wait for istio CRD creation jobs to complete
    kubectl -n istio-system wait --for=condition=complete job --all --timeout=300s || return $?

    # Change to use the OLCNE image for kubectl then install istio proper
    sed "s|/kubectl:|/istio_kubectl:|g" ${INSTALL_DIR}/istio.yaml | kubectl apply -f - || return $?

}

function update_coredns()
{
    if [ ${CLUSTER_TYPE} == "OKE" ]; then
        local cluster_ip
        cluster_ip=$(kubectl get svc -n istio-system istiocoredns -o jsonpath={.spec.clusterIP})
        if [ $? -ne 0 ] ; then
            return $?
        fi

        # Update coredns configmap to include global section in data.
        # This update requires coredns be greater than 1.4.0
        sed -e "s#@CLUSTER_IP@#${cluster_ip}#g" $CONFIG_DIR/coredns-template.yaml \
           | kubectl apply -f - \
           || return 1
    fi
    return 0
}

function copy_ocr_secret()
{
    kubectl get secret ocr -n default -o yaml \
        | sed 's|namespace: default|namespace: istio-system|' \
        | kubectl apply -n istio-system -f -
}

# Wait for all cluster nodes to be ready
action "Waiting for all Kubernetes nodes to be ready" \
    kubectl wait --for=condition=ready nodes --all || exit 1

# Secret named ocr must exist in the default namespace to pull OLCNE images in a OKE cluster
if [ ${CLUSTER_TYPE} == "OKE" ]; then
  action "Checking for secret named ocr in default namespace" kubectl get secret ocr -n default ||
    fail -e "ERROR: Secret named ocr is required to pull images from ${OLCNE_IMAGE_REPO}.\nCreate the secret in the default namespace and then rerun this script.\ne.g. kubectl create secret docker-registry ocr --docker-username=<username> --docker-password=<password> --docker-server=container-registry.oracle.com"
fi

# Create istio-system namespace if it does not exist
if ! kubectl get namespace istio-system > /dev/null 2>&1 ; then
  action "Creating istio-system namespace" \
    kubectl create namespace istio-system || exit 1
fi

# Copy the secret named ocr to the istio-system namespace for pulling OLCNE images in a OKE cluster
if [ ${CLUSTER_TYPE} == "OKE" ]; then
  if ! kubectl get secret ocr -n istio-system > /dev/null 2>&1 ; then
    action "Copying ocr secret to istio-system namespace" \
        copy_ocr_secret
  fi
fi

# Create certificates and istio secret to hold certificates if we haven't already
if ! kubectl get secret cacerts -n istio-system > /dev/null 2>&1 ; then
  action "Generating Istio CA bundle" create_secret || exit 1
fi

action "Installing Istio" install_istio || exit 1
action "Updating coredns" update_coredns || exit 1

kubectl get pods -n istio-system
