#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

. $SCRIPT_DIR/common.sh


INGRESS_TYPE=LoadBalancer #this is true for both OKE and OLCNE clusters

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
    helm repo add istio.io https://storage.googleapis.com/istio-release/releases/${ISTIO_VERSION}/charts || return $?

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
    EXTRA_HELM_ARGUMENTS=""
    if [ ${REGISTRY_SECRET_EXISTS} == "TRUE" ]; then
      EXTRA_HELM_ARGUMENTS=" --set global.imagePullSecrets[0]=${GLOBAL_IMAGE_PULL_SECRET}"
    fi
    if [ ${CLUSTER_TYPE} == "OLCNE" ] && [ $DNS_TYPE == "manual" ]; then
      ISTIO_INGRESS_IP=$(dig +short ingress-verrazzano.${NAME}.${DNS_SUFFIX})
      if [ -z ${ISTIO_INGRESS_IP} ]; then
        consoleerr
        consoleerr "Unable to identify an Ingress IP address. Check documentation and ensure the ingress-verrazzano DNS record exists"
        exit 1
      fi
      EXTRA_HELM_ARGUMENTS=$EXTRA_HELM_ARGUMENTS" --set gateways.istio-ingressgateway.externalIPs={"${ISTIO_INGRESS_IP}"}"
    fi

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
        --set prometheus.tag=v2.13.1 \
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

    log "Change to use the OLCNE image for kubectl then install the istio CRDs"
    sed "s|/kubectl:|/istio_kubectl:|g" ${TMP_DIR}/istio-crds.yaml | kubectl apply -f - || return $?

    log "Wait for istio CRD creation jobs to complete"
    if ! kubectl -n istio-system wait --for=condition=complete job --all --timeout=300s ; then
      stat=$?
      consoleerr "ERROR: Istio CRD creation failed"
      return $stat
    fi

    log "Change to use the OLCNE image for kubectl then install istio proper"
    sed "s|/kubectl:|/istio_kubectl:|g" ${TMP_DIR}/istio.yaml | kubectl apply -f - || return $?

}

function update_coredns()
{
    if [ "${CLUSTER_TYPE}" == "OKE" ] || [ "${CLUSTER_TYPE}" == "OLCNE" ]; then
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
    VER_ERROR_MSG="Kubernetes serverVersion $servVer must be greater than or equal to v1.16.8 and less than or equal to v1.18"
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
    if [ "$minorVer" -gt 2 ]; then
        log "Helm minor version is $minorVer, expected less than or equal to 2!"
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

function usage {
    consoleerr
    consoleerr "usage: $0 [-n name] [-d dns_type]"
    consoleerr "  -n name        Environment Name. Optional.  Defaults to default."
    consoleerr "  -d dns_type    DNS type [xip.io|manual|oci]. Optional.  Defaults to xip.io."
    consoleerr "  -s dns_suffix  DNS suffix (e.g v8o.example.com). Optional. Not valid for dns_type xip.io. Required for dns-type manual"
    consoleerr "  -h             Help"
    consoleerr
    exit 1
}

NAME="default"
DNS_TYPE="xip.io"

while getopts n:d:s:h flag
do
    case "${flag}" in
        n) NAME=${OPTARG};;
        d) DNS_TYPE=${OPTARG};;
        s) DNS_SUFFIX=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done

if [ $DNS_TYPE == "manual" ] && [ -z $DNS_SUFFIX ]; then
  consoleerr
  consoleerr "-s option is required for ${DNS_TYPE}"
  usage
fi

if [ "$DNS_TYPE" == "manual" ]; then
  command -v dig >/dev/null 2>&1 || {
      fail "dig is required for dns_type $DNS_TYPE but cannot be found on the path. Aborting."
  }
fi

action "Checking Kubernetes version" check_kube_version || exit 1
action "Checking Helm version" check_helm_version || (error "Helm version must be v3.0.x, v.3.1.x or v3.2.x!"; exit 1)

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
