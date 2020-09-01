#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install

. $INSTALL_DIR/common.sh

set -o pipefail

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

CONFIG_DIR=$INSTALL_DIR/config

function uninstall_istio() {
  # import istio to the help repository
  log "Add istio helm repository"
  helm repo add istio.io https://storage.googleapis.com/istio-release/releases/${ISTIO_VERSION}/charts || return $?

  log "Fetch istio charts for istio and istio-init"
  helm fetch istio.io/istio --untar=true --untardir=$TMP_DIR || return $?
  helm fetch istio.io/istio-init --untar=true --untardir=$TMP_DIR || return $?

  log "Generate cluster specific configuration"
  EXTRA_HELM_ARGUMENTS=""
  if [ ${CLUSTER_TYPE} == "OLCNE" ] && [ $DNS_TYPE == "manual" ]; then
    ISTIO_INGRESS_IP=$(dig +short ingress-verrazzano.${NAME}.${DNS_SUFFIX})
    if [ -z ${ISTIO_INGRESS_IP} ]; then
      consoleerr
      consoleerr "Unable to identify an Ingress IP address. Check documentation and ensure the ingress-verrazzano DNS record exists"
      exit 1
    fi
    EXTRA_HELM_ARGUMENTS=" --set gateways.istio-ingressgateway.externalIPs={"${ISTIO_INGRESS_IP}"}"
  fi

  # create template to to delete istio by file
  log "Create helm template for uninstalling istio proper"
  helm template istio ${TMP_DIR}/istio \
      --namespace istio-system \
      --set global.hub=$GLOBAL_HUB_REPO \
      --set global.tag=$ISTIO_VERSION \
      --set global.imagePullSecrets[0]=ocr \
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

  # create template to delete istio crds by file
  log "Create helm template for uninstalling istio CRDs"
  helm template istio-init ${TMP_DIR}/istio-init \
      --namespace istio-system \
      --set global.hub=$GLOBAL_HUB_REPO \
      --set global.tag=$ISTIO_VERSION \
      --set global.imagePullSecrets[0]=ocr \
      > ${TMP_DIR}/istio-crds.yaml || return $?

  # delete istio
  log "Change to use the OLCNE image for kubectl then uninstall istio proper"
  sed "s|/kubectl:|/istio_kubectl:|g" ${TMP_DIR}/istio.yaml | kubectl delete --ignore-not-found=true -f -

  # delete istio-crds
  log "Change to use the OLCNE image for kubectl then uninstall the istio CRDs"
  sed "s|/kubectl:|/istio_kubectl:|g" ${TMP_DIR}/istio-crds.yaml | kubectl delete --ignore-not-found=true -f -

  kubectl delete -f ${TMP_DIR}/istio-init/files --ignore-not-found=true
  kubectl delete -f ${TMP_DIR}/istio/files --ignore-not-found=true

  local istio_res=("$(helm repo ls \
    | grep "istio.io" || true)")

  printf "%s\n" "${istio_res[@]}" \
    | awk '{print $1}' \
    | xargs helm repo remove \
    || return $? # return on pipefail
}

function delete_secrets() {
  # Delete istio.default in all namespaces
  log "Collecting istio secrets for deletion"
  kubectl delete secret istio.default --ignore-not-found=true || return $?
  kubectl delete secret istio.default -n kube-public --ignore-not-found=true || return $?
  kubectl delete secret istio.default -n kube-node-lease --ignore-not-found=true || return $?

  # delete secrets left over in kube-system
  local secret_res=("$(kubectl get secrets -n kube-system --no-headers -o custom-columns=":metadata.name,:metadata.annotations" \
  | grep "istio.io" || true)")

  printf "%s\n" "${secret_res[@]}" \
  | awk '{print $1}' \
  | xargs kubectl delete secret -n kube-system \
  || return $? # return on pipefail
}

function delete_istio_namepsace() {
  log "Deleting istio-system namespace"
  kubectl delete namespace istio-system --ignore-not-found=true || return $?
}

action "Deleting Istio Components" uninstall_istio || exit 1
action "Deleting Istio Secrets" delete_secrets || exit 1
action "Deleting Istio Namespace" delete_istio_namepsace || exit 1