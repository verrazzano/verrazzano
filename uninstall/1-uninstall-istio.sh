#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../install

. $INSTALL_DIR/common.sh

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

CONFIG_DIR=$INSTALL_DIR/config

function delete_coredns() {
  # delete coredns if cluster type is OKE
  if [ ${CLUSTER_TYPE} == "OKE" ]; then
        local cluster_ip
        cluster_ip=$(kubectl get svc -n istio-system istiocoredns -o jsonpath={.spec.clusterIP})
        if [ $? -ne 0 ] ; then
            return $?
        fi

        # Update coredns configmap to include global section in data.
        # This update requires coredns be greater than 1.4.0
        sed -e "s#@CLUSTER_IP@#${cluster_ip}#g" $CONFIG_DIR/coredns-template.yaml \
           | kubectl delete -f -
    fi
    return 0
}

function uninstall_istio() {
  # check if istio namespace has been created
  if [ -z $(kubectl get namespace istio-system) ] ; then
    return 0
  fi

  # import istio to the help repository
  log "Add istio helm repository"
    helm repo add istio.io https://storage.googleapis.com/istio-release/releases/${ISTIO_VERSION}/charts || return $?

  # grab the istio charts
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
  log "Create helm template for installing istio proper"
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
    sed "s|/kubectl:|/istio_kubectl:|g" ${TMP_DIR}/istio.yaml | kubectl delete -f -

  # delete istio-crds
  log "Change to use the OLCNE image for kubectl then uninstall the istio CRDs"
  sed "s|/kubectl:|/istio_kubectl:|g" ${TMP_DIR}/istio-crds.yaml | kubectl delete -f -

  kubectl delete -f ${TMP_DIR}/istio-init/files
  kubectl delete -f ${TMP_DIR}/istio/files

  helm repo remove istio.io

}

function delete_secrets() {
  if [ "$(kubectl get secret istio.default)" ] ; then
    kubectl delete secret istio.default
  fi

  if [ "$(kubectl get secret istio.default -n kube-public)" ] ; then
    kubectl delete secret istio.default -n kube-public
  fi

  if [ "$(kubectl get secret istio.default -n kube-node-lease)" ] ; then
    kubectl delete secret istio.default -n kube-node-lease
  fi

  # delete secrets left over in kube-system
  local istiosec=($(kubectl get secrets -n kube-system -o custom-columns=":metadata.name" --no-headers |  grep istio))

  for sec in "${istiosec[@]}"
  do
    kubectl delete secret "${sec}" -n kube-system
  done
}

function delete_istio_namepsace() {
  if [ "$(kubectl get namespace istio-system)" ] ; then
    kubectl delete namespace istio-system
  fi
}

action "Deleting Core DNS" delete_coredns || exit 1
action "Uninstalling Istio Components" uninstall_istio || exit 1
action "Cleaning Up Istio Secrets" delete_secrets || exit 1
action "Deleting Istio Namespace" delete_istio_namepsace || exit 1