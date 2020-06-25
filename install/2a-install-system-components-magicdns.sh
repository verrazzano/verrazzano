#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle Corporation and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh

if [ ${CLUSTER_TYPE} == "OKE" ]; then
  INGRESS_TYPE=LoadBalancer
elif [ ${CLUSTER_TYPE} == "KIND" ] || [ "${CLUSTER_TYPE}" == "OLCNE" ]; then
  INGRESS_TYPE=NodePort
fi

VERRAZZANO_NS=verrazzano-system

set -eu

function set_INGRESS_IP() {
  if [ ${CLUSTER_TYPE} == "OKE" ]; then
    INGRESS_IP=$(kubectl get svc ingress-controller-nginx-ingress-controller -n ingress-nginx -o json | jq -r '.status.loadBalancer.ingress[0].ip')
  elif [ ${CLUSTER_TYPE} == "KIND" ]; then
    INGRESS_IP=$(kubectl get node ${KIND_CLUSTER_NAME}-control-plane -o json | jq -r '.status.addresses[] | select (.type == "InternalIP") | .address')
  fi
}

function install_nginx_ingress_controller()
{
    set +e
    helm uninstall ingress-controller --namespace ingress-nginx
    set -e

    # Create the namespace for nginx
    if ! kubectl get namespace ingress-nginx ; then
        kubectl create namespace ingress-nginx
    fi

    helm repo add stable https://kubernetes-charts.storage.googleapis.com
    helm repo update

    helm upgrade ingress-controller stable/nginx-ingress --install \
      --set controller.image.repository=$NGINX_INGRESS_CONTROLLER_IMAGE \
      --set controller.image.tag=$NGINX_INGRESS_CONTROLLER_TAG \
      --set defaultBackend.image.repository=$NGINX_DEFAULT_BACKEND_IMAGE \
      --set defaultBackend.image.tag=$NGINX_DEFAULT_BACKEND_TAG \
      --namespace ingress-nginx \
      --set controller.metrics.enabled=true \
      --set controller.podAnnotations.'prometheus\.io/port'=10254 \
      --set controller.podAnnotations.'prometheus\.io/scrape'=true \
      --set controller.podAnnotations.'system\.io/scrape'=true \
      --version $NGINX_INGRESS_CONTROLLER_VERSION \
      --set controller.service.type="${INGRESS_TYPE}" \
      --timeout 15m0s \
      --wait

    if [ $CLUSTER_TYPE = "KIND" ]; then
        kubectl patch deployments -n ingress-nginx ingress-controller-nginx-ingress-controller -p '{"spec":{"template":{"spec":{"containers":[{"name":"nginx-ingress-controller","ports":[{"containerPort":80,"hostPort":80},{"containerPort":443,"hostPort":443}]}],"tolerations":[{"key":"node-role.kubernetes.io/master","operator":"Equal","effect":"NoSchedule"}],"nodeSelector":{"ingress-ready":"true"}}}}}'
    fi

    set_INGRESS_IP

    if [ $DNS_TYPE = "xip.io" ]; then
      DNS_SUFFIX="${INGRESS_IP}".xip.io
    fi
}

function install_cert_manager()
{
    set +e
    helm uninstall cert-manager --namespace cert-manager
    set -e

    # Create the namespace for cert-manager
    if ! kubectl get namespace cert-manager ; then
        kubectl create namespace cert-manager
    fi

    helm repo add jetstack https://charts.jetstack.io
    helm repo update

    kubectl apply \
        -f https://raw.githubusercontent.com/jetstack/cert-manager/release-0.13/deploy/manifests/00-crds.yaml \
        --validate=false

    helm upgrade cert-manager jetstack/cert-manager \
        --install \
        --namespace cert-manager \
        --version $CERT_MANAGER_VERSION \
        --set image.repository=$CERT_MANAGER_IMAGE \
        --set image.tag=$CERT_MANAGER_TAG \
        --set extraArgs[0]=--acme-http01-solver-image=$CERT_MANAGER_SOLVER_IMAGE:$CERT_MANAGER_TAG \
        --set cainjector.enabled=false \
        --set webhook.enabled=false \
        --set webhook.injectAPIServerCA=false \
        --set ingressShim.defaultIssuerName=verrazzano-issuer \
        --set ingressShim.defaultIssuerKind=ClusterIssuer \
        --set clusterResourceNamespace=cattle-system \
        --wait

    kubectl apply -f <(echo "
apiVersion: cert-manager.io/v1alpha2
kind: ClusterIssuer
metadata:
  name: verrazzano-issuer
spec:
  ca:
    secretName: tls-rancher
")

    kubectl -n cert-manager rollout status -w deploy/cert-manager
}

function install_rancher()
{
    set +e
    helm uninstall rancher --namespace cattle-system
    set -e

    if ! kubectl get namespace cattle-system ; then
        kubectl create namespace cattle-system
    fi

    helm repo add rancher-stable https://releases.rancher.com/server-charts/stable
    helm repo update

    helm upgrade rancher rancher-stable/rancher \
      --install --namespace cattle-system \
      --version $RANCHER_VERSION  \
      --set rancherImageTag=$RANCHER_TAG \
      --set rancherImage=$RANCHER_IMAGE \
      --set hostname=rancher.${NAME}.${DNS_SUFFIX} \
      --set ingress.tls.source=rancher

    if [ $CLUSTER_TYPE == "OLCNE" ]; then
      # CRI-O does not deliver MKNOD by default, until https://github.com/rancher/rancher/pull/27582 is merged we must add the capability
      kubectl patch deployments -n cattle-system rancher -p '{"spec":{"template":{"spec":{"containers":[{"name":"rancher","securityContext":{"capabilities":{"add":["MKNOD"]}}}]}}}}'
    fi

    if [ $DNS_TYPE == "xip.io" ] || [ $DNS_TYPE == "manual" ]; then
      RANCHER_PATCH_DATA="{\"metadata\":{\"annotations\":{\"kubernetes.io/tls-acme\":\"true\",\"nginx.ingress.kubernetes.io/auth-realm\":\"${NAME}.${DNS_SUFFIX} auth\",\"cert-manager.io/issuer\":\"rancher\",\"cert-manager.io/issuer-kind\":\"Issuer\"}}}"
    fi

    kubectl patch ingress rancher -n cattle-system -p "$RANCHER_PATCH_DATA"  --type=merge

    kubectl -n cattle-system rollout status -w deploy/rancher
}

function usage {
    consoleerr
    consoleerr "usage: $0 [-n name] [-d dns_type]"
    consoleerr "  -n name        Environment Name. Optional.  Defaults to default."
    consoleerr "  -d dns_type    DNS type [xip.io|manual]. Optional.  Defaults to xip.io."
    consoleerr "  -s dns_suffix  DNS suffix (e.g v8o.example.com). Not valid for dns_type xip.io. Required for dns-type oci or manual"
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
# check for valid DNS type
if [ $DNS_TYPE != "xip.io" ] && [ $DNS_TYPE != "manual" ]; then
  consoleerr
  consoleerr "Unknown DNS type ${DNS_TYPE}!"
  usage
fi

if [ $DNS_TYPE == "manual" ] && [ -z $DNS_SUFFIX ]; then
  consoleerr
  consoleerr "-s option is required for ${DNS_TYPE}"
  usage
fi

action "Installing Nginx Ingress Controller" install_nginx_ingress_controller || exit 1
action "Installing cert manager" install_cert_manager || exit 1
action "Installing Rancher" install_rancher || exit 1
