#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/config.sh

set -eu

function install_nginx_ingress_controller()
{
    # Create the namespace for nginx
    if ! kubectl get namespace ingress-nginx ; then
        kubectl create namespace ingress-nginx
    fi

    helm repo add stable https://charts.helm.sh/stable
    helm repo update

    local ingress_type=""
    ingress_type=$(get_config_value ".ingress.type")

    local EXTRA_NGINX_ARGUMENTS=""
    local patch_service_spec=""
    local extra_install_args=""
    local extra_install_args_len=0
    local param_name=""
    local param_value=""
    if [ "$ingress_type" == "LoadBalancer" ]; then
      # Get any patch for the service and deployment specs
      patch_service_spec="$(get_config_value '.ingress.verrazzano.patchServiceSpec')"
      # Handle any additional NGINX install args - since NGINX is for Verrazzano system Ingress,
      # these should be in .ingress.verrazzano.extraInstallArgs[]
      extra_install_args=($(get_config_array ".ingress.verrazzano.extraInstallArgs[]"))
      if [ ${#extra_install_args[@]} -ne 0 ]; then
        for arg in "${extra_install_args[@]}"; do
          param_name=$(echo "$arg" | jq -r '.name')
          param_value=$(echo "$arg" | jq -r '.value')
          if [ ! -z "$param_name" ] && [ ! -z "$param_value" ]; then
            EXTRA_NGINX_ARGUMENTS="$EXTRA_NGINX_ARGUMENTS --set $param_name=$param_value"
          fi
        done
      fi

      # Handle any external IPs specified for Verrazzano Ingress - this may exist when an
      # external LB is used
      local additional_external_ips=($(get_config_array ".ingress.verrazzano.additionalExternalIPs[]"))
      local additional_external_ips_len=${#additional_external_ips[@]}
      if [ $additional_external_ips_len -ne 0 ]; then
        printf -v joined '%s,' "${additional_external_ips[@]}"
        EXTRA_NGINX_ARGUMENTS=$EXTRA_NGINX_ARGUMENTS" --set controller.service.externalIPs={"${joined%,}"}"
      fi
    fi #end if ingress_type is LoadBalancer

    helm upgrade ingress-controller stable/nginx-ingress --install \
      --set controller.image.repository=$NGINX_INGRESS_CONTROLLER_IMAGE \
      --set controller.image.tag=$NGINX_INGRESS_CONTROLLER_TAG \
      --set controller.config.client-body-buffer-size=64k \
      --set defaultBackend.image.repository=$NGINX_DEFAULT_BACKEND_IMAGE \
      --set defaultBackend.image.tag=$NGINX_DEFAULT_BACKEND_TAG \
      --namespace ingress-nginx \
      --set controller.metrics.enabled=true \
      --set controller.podAnnotations.'prometheus\.io/port'=10254 \
      --set controller.podAnnotations.'prometheus\.io/scrape'=true \
      --set controller.podAnnotations.'system\.io/scrape'=true \
      --version $NGINX_INGRESS_CONTROLLER_VERSION \
      --set controller.service.type="${ingress_type}" \
      --set controller.publishService.enabled=true \
      --timeout 15m0s \
      ${EXTRA_NGINX_ARGUMENTS} \
      --wait

    if [ ! -z "${patch_service_spec}" ]; then
      kubectl patch service -n ingress-nginx ingress-controller-nginx-ingress-controller -p "${patch_service_spec}"
    fi
}

function install_cert_manager()
{
    # Create the namespace for cert-manager
    if ! kubectl get namespace cert-manager ; then
        kubectl create namespace cert-manager
    fi

    helm repo add jetstack https://charts.jetstack.io
    helm repo update

    kubectl apply \
        -f "https://raw.githubusercontent.com/jetstack/cert-manager/release-${CERT_MANAGER_RELEASE}/deploy/manifests/00-crds.yaml" \
        --validate=false

    helm upgrade cert-manager jetstack/cert-manager \
        --install \
        --namespace cert-manager \
        --version $CERT_MANAGER_HELM_CHART_VERSION \
        --set image.repository=$CERT_MANAGER_IMAGE \
        --set image.tag=$CERT_MANAGER_TAG \
        --set extraArgs[0]=--acme-http01-solver-image=$CERT_MANAGER_SOLVER_IMAGE:$CERT_MANAGER_SOLVER_TAG \
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
    log "Create Rancher namespace (if required)"
    if ! kubectl get namespace cattle-system > /dev/null 2>&1; then
        kubectl create namespace cattle-system
    fi

    log "Add Rancher helm repository location"
    helm repo add rancher-stable https://releases.rancher.com/server-charts/stable

    log "Update helm repositoriess"
    helm repo update

    log "Install Rancher"
    helm upgrade rancher rancher-stable/rancher \
      --install --namespace cattle-system \
      --version $RANCHER_VERSION  \
      --set systemDefaultRegistry=ghcr.io/verrazzano \
      --set rancherImageTag=$RANCHER_TAG \
      --set rancherImage=$RANCHER_IMAGE \
      --set hostname=rancher.${NAME}.${DNS_SUFFIX} \
      --set ingress.tls.source=rancher

    # CRI-O does not deliver MKNOD by default, until https://github.com/rancher/rancher/pull/27582 is merged we must add the capability
    # OLCNE uses CRI-O and needs this change, and it doesn't hurt other cases
    kubectl patch deployments -n cattle-system rancher -p '{"spec":{"template":{"spec":{"containers":[{"name":"rancher","securityContext":{"capabilities":{"add":["MKNOD"]}}}]}}}}'

    # Patch for rancher needed in both cases this script supports (xip.io "magic" DNS and external DNS)
    RANCHER_PATCH_DATA="{\"metadata\":{\"annotations\":{\"kubernetes.io/tls-acme\":\"true\",\"nginx.ingress.kubernetes.io/auth-realm\":\"${NAME}.${DNS_SUFFIX} auth\",\"cert-manager.io/issuer\":\"rancher\",\"cert-manager.io/issuer-kind\":\"Issuer\"}}}"

    log "Patch Rancher ingress"
    kubectl patch ingress rancher -n cattle-system -p "$RANCHER_PATCH_DATA" --type=merge

    log "Rollout Rancher"
    kubectl -n cattle-system rollout status -w deploy/rancher || return $?

    log "Create Rancher secrets"
    RANCHER_DATA=$(kubectl --kubeconfig $KUBECONFIG -n cattle-system exec $(kubectl --kubeconfig $KUBECONFIG -n cattle-system get pods -l app=rancher | grep '1/1' | head -1 | awk '{ print $1 }') -- reset-password 2>/dev/null)
    ADMIN_PW=`echo $RANCHER_DATA | awk '{ print $NF }'`

    if [ -z "$ADMIN_PW" ] ; then
      error "ERROR: Failed to reset Rancher password"
      return 1
    fi

    kubectl -n cattle-system create secret generic rancher-admin-secret --from-literal=password="$ADMIN_PW"
}

function usage {
    consoleerr
    consoleerr "usage: $0"
    consoleerr "-h             Help"
    consoleerr "INSTALL_CONFIG_FILE environment variable must be set to a valid installation config file"
    consoleerr
    exit 1
}

NAME=$(get_config_value ".environmentName")
DNS_TYPE=$(get_config_value ".dns.type")
DNS_SUFFIX=""

if [ "$DNS_TYPE" == "oci" ]; then
  fail "This script is not intended for use with OCI DNS. Please run the appropriate script for OCI DNS."
fi

log "2a got name $NAME and DNS_TYPE $DNS_TYPE"

while getopts h flag
do
    case "${flag}" in
        h) usage;;
        *) usage;;
    esac
done

# check environment name length
validate_environment_name $NAME
if [ $? -ne 0 ]; then
  exit 1
fi

action "Installing NGINX ingress controller" install_nginx_ingress_controller || exit 1

# We can only know the ingress IP after installing nginx ingress controller
INGRESS_IP=$(get_verrazzano_ingress_ip)

# DNS_SUFFIX is only used by install_rancher
DNS_SUFFIX=$(get_dns_suffix ${INGRESS_IP})
log "2a dns suffix is ${DNS_SUFFIX}"

action "Installing certificate manager" install_cert_manager || exit 1
action "Installing Rancher" install_rancher || exit 1
