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

if [ "$(kubectl get vb -A)" ] || [ "$(kubectl get vm -A)" ] ; then
  error "Please delete all Verrazzano Models and Verrazzano Bindings before continuing the uninstall"
  exit 1
fi

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
    sed "s|/kubectl:|/istio_kubectl:|g" ${TMP_DIR}/istio.yaml | kubectl delete -f -

  # delete istio-crds
  log "Change to use the OLCNE image for kubectl then uninstall the istio CRDs"
  sed "s|/kubectl:|/istio_kubectl:|g" ${TMP_DIR}/istio-crds.yaml | kubectl delete -f -

  kubectl delete -f ${TMP_DIR}/istio-init/files
  kubectl delete -f ${TMP_DIR}/istio/files

  helm repo remove istio.io

}

function delete_secrets() {
  # Delete istio.default in all namespaces
  log "Collecting istio secrets for deletion"
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
  kubectl get secrets -n kube-system -o custom-columns=":metadata.name" --no-headers \
  | grep 'istio' \
  | xargs kubectl delete secret -n kube-system
}

function delete_istio_namepsace() {
  log "Deleting istio-system namespace"
  if [ "$(kubectl get namespace istio-system)" ] ; then
    kubectl delete namespace istio-system
  fi
}

if [ "$(kubectl get vb -A)" ] || [ "$(kubectl get vm -A)" ] ; then
  error "Please delete all Verrazzano Models and Verrazzano Bindings before continuing the uninstall"
fi

function delete_external_dns() {
  log "Deleting external-dns"
  helm delete external-dns -n cert-manager || 2>/dev/null

  # delete clusterrole and clusterrolebinding
  log "Deleting ClusterRoles and ClusterRoleBindings for external-dns"
  if [ "$(kubectl get clusterrole external-dns)" ] ; then
    kubectl delete clusterrole external-dns
  fi

  if [ "$(kubectl get clusterrolebinding external-dns)" ] ; then
    kubectl delete clusterrolebinding external-dns
  fi
}

function delete_nginx() {
  # uninstall ingress-nginx
  log "Deleting ingress-nginx"
  helm delete ingress-controller -n ingress-nginx || 2>/dev/null

  # delete the nginx clusterrole and clusterrolebinding
  log "Deleting ClusterRoles and ClusterRoleBindings for ingress-nginx"
  if [ "$(kubectl get clusterrole ingress-controller-nginx-ingress)" ] ; then
    kubectl delete clusterrole ingress-controller-nginx-ingress
  fi

  if [ "$(kubectl get clusterrolebinding ingress-controller-nginx-ingress)" ] ; then
    kubectl delete clusterrolebinding ingress-controller-nginx-ingress
  fi

  # delete ingress-nginx namespace
  log "Deleting ingress-nginx namespace"
  if [ "$(kubectl get namespace ingress-nginx)" ] ; then
    kubectl delete namespace ingress-nginx
  fi
}

function delete_cert_manager() {
  # uninstall cert manager deployment
  log "Deleting cert-manager"
  helm delete cert-manager -n cert-manager || 2>/dev/null

  # delete the custom resource definition for cert manager
  log "deleting the custom resource definition for cert manager"
  kubectl delete -f https://raw.githubusercontent.com/jetstack/cert-manager/release-0.13/deploy/manifests/00-crds.yaml

  # delete cert manager config map
  log "Deleting config map for cert manager"
  if [ "$(kubectl get configmap cert-manager-controller -n kube-system)" ] ; then
    kubectl delete configmap cert-manager-controller -n kube-system
  fi

  # delete namespace
  log "Deleting cert manager namespace"
  if [ "$(kubectl get namespace cert-manager)" ] ; then
    kubectl delete namespace cert-manager
  fi
}

function delete_rancher() {
  # Deleting rancher components
  log "Deleting rancher"
  helm delete rancher -n cattle-system || 2>/dev/null

  log "Deleting CRDs from rancher"
  while [ "$(kubectl get crds --no-headers -o custom-columns=":metadata.name" | grep -E 'coreos.com|.cattle.io')" ]
  do
    # remove finalizers from crds
    kubectl get crds --no-headers -o custom-columns=":metadata.name" \
      | grep -E 'coreos.com|.cattle.io' \
      | xargs kubectl patch crd -p '{"metadata":{"finalizers":null}}' --type=merge

    # delete crds (include timeout for undiscovered finalizer problem)
    kubectl get crds --no-headers -o custom-columns=":metadata.name" \
      | grep -E 'coreos.com|.cattle.io' \
      | xargs kubectl delete crd &
    sleep 30
    kill $! || 2>/dev/null
  done

  # delete clusterrolebindings deployed by rancher
  log "Deleting ClusterRoleBindings"
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'clusterrolebinding-|cattle-globalrolebinding-|globaladmin-user|grb-u|rancher' \
    | xargs kubectl delete clusterrolebinding

  # delete clusterroles
  log "Deleting ClusterRoles"
  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'p-|project-|user-|cluster-owner|create-ns' \
    | xargs kubectl delete clusterrole

  # delete rolebinding
  log "Deleting RoleBindings"
  local default_names=("default" "kube-node-lease" "kube-public" "kube-system")
  for namespace in "${default_names[@]}"
  do
    kubectl get rolebinding --no-headers -o custom-columns=":metadata.name" -n "${namespace}"\
      | grep 'clusterrolebinding-' \
      | xargs kubectl delete rolebinding -n "${namespace}"
  done

  # delete configmap in kube-system
  if [ "$(kubectl get configmap cattle-controllers -n kube-system)" ] ; then
    kubectl delete configmap cattle-controllers -n kube-system
  fi

  log "Deleting cattle namespaces"
  # delete namespace finalizers
  kubectl get namespaces --no-headers -o custom-columns=":metadata.name" \
    | grep 'cattle' \
    | xargs kubectl patch namespace -p '{"metadata":{"finalizers":null}}' --type=merge

  # delete cattle namespaces
  kubectl get namespaces --no-headers -o custom-columns=":metadata.name" | grep -E 'cattle|local' | xargs kubectl delete namespaces
}

if [ "$(kubectl get vb -A)" ] || [ "$(kubectl get vm -A)" ] ; then
  error "Please delete all Verrazzano Models and Verrazzano Bindings before continuing the uninstall"
fi

function delete_verrazzano() {
  # delete helm installation of Verrazzano
  log "Deleting Verrazzano"
  helm delete verrazzano -n verrazzano-system || 2>/dev/null

  # delete verrazzano-managed-cluster-local secret
  log "Deleting Verrazzano secrets"
  if [ "$(kubectl get secret verrazzano-managed-cluster-local)" ] ; then
    kubectl delete secret verrazzano-managed-cluster-local
  fi

  # delete crds
  log "Deleting Verrazzano crds"
  kubectl get crds --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'verrazzano.io' \
    | xargs kubectl delete crd

  # deleting certificatesigningrequests
  log "Deleting CertificateSigningRequests"
  kubectl get csr --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'csr-' \
    | xargs kubectl delete csr

  log "Deleting ClusterRoles and ClusterRoleBindings"
  # deleting clusterrolebindings
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'filebeat|journalbeat|node-exporter' \
    | xargs kubectl delete clusterrolebinding

  # deleting clusterroles
  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'filebeat|journalbeat|node-exporter' \
    | xargs kubectl delete clusterrole

  # deleting namespaces
  log "Deleting Verrazzano namespaces"
  kubectl get namespace --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'verrazzano-system|monitoring|logging' \
    | xargs kubectl delete namespace
}

if [ "$(kubectl get vb -A)" ] || [ "$(kubectl get vm -A)" ] ; then
  error "Please delete all Verrazzano Models and Verrazzano Bindings before continuing the uninstall"
fi

function delete_mysql() {
  # delete helm installation of MySQL
  log "Deleting MySQL"
  helm delete mysql -n keycloak || 2>/dev/null
}

function delete_keycloak() {
  # delete helm installation of Keycloak
  log "Deleting Keycloak"
  helm delete keycloak -n keycloak || 2>/dev/null

  # delete keycloak namespace
  log "Deleting Keycloak namespace"
  kubectl get namespace --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'keycloak' \
    | xargs kubectl delete namespace
}

function delete_resources() {
  log "Deleting ClusterRoles and ClusterRoleBindings"
  # deleting clusterrolebindings
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'cattle-admin|proxy-role-binding-kubernetes-master' \
    | xargs kubectl delete clusterrolebinding

  # deleting clusterroles
  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'cattle-admin|local-cluster|proxy-clusterrole-kubeapiserver' \
    | xargs kubectl delete clusterrole
}

function finalize() {
  log "Deleting ocr Secret"
  if [ "$(kubectl get secret ocr)" ] ; then
    kubectl delete secret ocr
  fi

  # Grab all leftover Helm repos and delete resources
  log "Deleting Helm repos"
  helm repo ls | awk 'NR>1 {print $1}' | xargs -I name helm repo remove name

  # Removing possible reference to verrazzano in clusterroles and clusterrolebindings
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'verrazzano' \
    | xargs kubectl delete clusterrolebinding

  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'verrazzano' \
    | xargs kubectl delete clusterrole
}

function 4-uninstall_keycloak() {
  action "Deleting MySQL Components" delete_mysql || exit 1
  action "Deleting Keycloak Components" delete_keycloak || exit 1
  action "Deleting Leftover Resources" delete_resources || exit 1
}

function 3-uninstall_verrazzano() {
  action "Deleting Verrazzano Components" delete_verrazzano || exit 1
}

function 2-uninstall_system_components() {
  action "Deleting External DNS Components" delete_external_dns || exit 1
  action "Deleting Nginx Components" delete_nginx || exit 1
  action "Deleting Cert Manager Components" delete_cert_manager || exit 1
  action "Deleting Rancher Components" delete_rancher || exit 1
}

function 1-uninstall_istio() {
  action "Deleting Istio Components" uninstall_istio || exit 1
  action "Deleting Istio Secrets" delete_secrets || exit 1
  action "Deleting Istio Namespace" delete_istio_namepsace || exit 1
  action "Finalizing Uninstall" finalize || exit 1
}

function usage {
    error
    error "usage: $0 [-i ignore_num] [-n script_num] [-h]"
    error " -n script_num   Number of script to be executed"
    error " -h              Help"
    error
    exit 1
}


while getopts n:h flag
do
    case "${flag}" in
        n) NUM_SCRIPT=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done

if [[ "$NUM_SCRIPT" =~ ^[1-4]$ ]] ; then
  script=$(echo '4-uninstall_keycloak
                 3-uninstall_verrazzano
                 2-uninstall_system_components
                 1-uninstall_istio' \
            | grep "$NUM_SCRIPT")
  $script
else
  4-uninstall_keycloak
  3-uninstall_verrazzano
  2-uninstall_system_components
  1-uninstall_istio
fi

