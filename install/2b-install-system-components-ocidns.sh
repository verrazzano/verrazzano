#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
INGRESS_VERSION=1.27.0
DNS_PREFIX="verrazzano-ingress"
OCI_PRIVATE_KEY_PASSPHRASE=${OCI_PRIVATE_KEY_PASSPHRASE:-""}

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh

CONFIG_DIR=$SCRIPT_DIR/config

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

CHECK_VALUES=false
set +u
if [ -z "$OCI_REGION" ]; then
    echo "OCI_REGION environment variable must set to OCI Region"
    CHECK_VALUES=true
fi
if [ -z "$OCI_TENANCY_OCID" ]; then
    echo "OCI_TENANCY_OCID environment variable must set to OCI Tenancy OCID"
    CHECK_VALUES=true
fi
if [ -z "$OCI_USER_OCID" ]; then
    echo "OCI_USER_OCID environment variable must set to OCI User OCID"
    CHECK_VALUES=true
fi
if [ -z "$OCI_COMPARTMENT_OCID" ]; then
    echo "OCI_COMPARTMENT_OCID environment variable must set to OCI Compartment OCID"
    CHECK_VALUES=true
fi
if [ -z "$OCI_FINGERPRINT" ]; then
    echo "OCI_FINGERPRINT environment variable must set to OCI Fingerprint"
    CHECK_VALUES=true
fi
if [ -z "$OCI_PRIVATE_KEY_FILE" ]; then
    echo "OCI_PRIVATE_KEY_FILE environment variable must set to OCI Private Key File"
    CHECK_VALUES=true
fi
if [ -z "$EMAIL_ADDRESS" ]; then
    echo "EMAIL_ADDRESS environment variable must set to your email address"
    CHECK_VALUES=true
fi
if [ -z "$OCI_DNS_ZONE_OCID" ]; then
    echo "OCI_DNS_ZONE_OCID environment variable must set to OCI DNS Zone OCID"
    CHECK_VALUES=true
fi
if [ -z "$OCI_DNS_ZONE_NAME" ]; then
    echo "OCI_DNS_ZONE_NAME environment variable must set to OCI DNS Zone Name"
    CHECK_VALUES=true
fi
if [ $CHECK_VALUES = true ]; then
    exit 1
fi

[ ! -f $OCI_PRIVATE_KEY_FILE ] && { echo $OCI_PRIVATE_KEY_FILE does not exist; exit 1; }

set -eu

function install_nginx_ingress_controller()
{
    # Create the namespace for nginx
    if ! kubectl get namespace ingress-nginx ; then
        kubectl create namespace ingress-nginx
    fi

    helm repo add stable https://charts.helm.sh/stable
    helm repo update

    helm upgrade ingress-controller stable/nginx-ingress --install \
      --set controller.image.repository=$NGINX_INGRESS_CONTROLLER_IMAGE \
      --set controller.image.tag=$NGINX_INGRESS_CONTROLLER_TAG \
      --set controller.config.client-body-buffer-size=64k \
      --set defaultBackend.image.repository=$NGINX_DEFAULT_BACKEND_IMAGE \
      --set defaultBackend.image.tag=$NGINX_DEFAULT_BACKEND_TAG \
      --set controller.publishService.enabled=true \
      --namespace ingress-nginx \
      --set controller.metrics.enabled=true \
      --set controller.podAnnotations.'prometheus\.io/port'=10254 \
      --set controller.podAnnotations.'prometheus\.io/scrape'=true \
      --set controller.podAnnotations.'system\.io/scrape'=true \
      --version $INGRESS_VERSION \
      --set controller.service.enableHttp=false \
      --set controller.service.type=LoadBalancer \
      --set controller.service.annotations.'external-dns\.alpha\.kubernetes\.io/ttl'=60 \
      --set controller.service.annotations.'external-dns\.alpha\.kubernetes\.io/hostname'=${DNS_PREFIX}.${NAME}.${OCI_DNS_ZONE_NAME} \
      --timeout 15m0s \
      --wait
}

function install_cert_manager()
{
    # Create the namespace for cert-manager
    if ! kubectl get namespace cert-manager ; then
        kubectl create namespace cert-manager
    fi

    helm repo add jetstack https://charts.jetstack.io

    curl -L -o "$TMP_DIR/00-crds.yaml" \
        "https://raw.githubusercontent.com/jetstack/cert-manager/release-${CERT_MANAGER_RELEASE}/deploy/manifests/00-crds.yaml"
    patch "$TMP_DIR/00-crds.yaml" "$CONFIG_DIR/00-crds.patch"
    kubectl apply -f "$TMP_DIR/00-crds.yaml" --validate=false

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
        --set ingressShim.defaultIssuerName=verrazzano-dns-issuer \
        --set ingressShim.defaultIssuerKind=ClusterIssuer \
        --wait

    source /dev/stdin <<<"$(echo 'cat <<EOF >$TMP_DIR/oci.yaml'; cat $CONFIG_DIR/oci.yaml; echo EOF;)"
    kubectl create secret generic -n cert-manager verrazzano-oci-dns-config --from-file=$TMP_DIR/oci.yaml
    kubectl apply -f <(echo "
apiVersion: cert-manager.io/v1alpha2
kind: ClusterIssuer
metadata:
  name: verrazzano-dns-issuer
spec:
  acme:
    email: $EMAIL_ADDRESS
    server: "https://acme-v02.api.letsencrypt.org/directory"
    privateKeySecretRef:
      name: verrazzano-cert-acme-secret
    solvers:
      - dns01:
          ocidns:
            useInstancePrincipals: false
            serviceAccountSecretRef:
              name: verrazzano-oci-dns-config
              key: "oci.yaml"
            ocizonename: $OCI_DNS_ZONE_NAME
")

    kubectl -n cert-manager rollout status -w deploy/cert-manager
}

function install_external_dns()
{
    helm upgrade external-dns stable/external-dns \
        --install \
        --namespace cert-manager \
        --version $EXTERNAL_DNS_VERSION \
        --set image.registry=$EXTERNAL_DNS_REGISTRY \
        --set image.repository=$EXTERNAL_DNS_REPO \
        --set image.tag=$EXTERNAL_DNS_TAG \
        --set provider=oci \
        --set logLevel=debug \
        --set registry=txt \
        --set sources[0]=ingress \
        --set sources[1]=service \
        --set domainFilters[0]=${OCI_DNS_ZONE_NAME} \
        --set zoneIdFilters[0]=$OCI_DNS_ZONE_OCID \
        --set txtOwnerId=v8o-local-${NAME} \
        --set txtPrefix=_v8o-local-${NAME}_ \
        --set policy=sync \
        --set interval=24h \
        --set triggerLoopOnEvent=true \
        --set extraVolumes[0].name=config \
        --set extraVolumes[0].secret.secretName=verrazzano-oci-dns-config \
        --set extraVolumeMounts[0].name=config \
        --set extraVolumeMounts[0].mountPath=/etc/kubernetes/ \
        --wait
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
      --set rancherImage=$RANCHER_IMAGE \
      --set rancherImageTag=$RANCHER_TAG \
      --set hostname=rancher.${NAME}.${OCI_DNS_ZONE_NAME} \
      --set ingress.tls.source=letsEncrypt \
      --set letsEncrypt.ingress.class=rancher \
      --set letsEncrypt.environment=production \
      --set letsEncrypt.email=$EMAIL_ADDRESS \
      --wait

    K8S_IO_HOSTNAME=${DNS_PREFIX}.${NAME}.${OCI_DNS_ZONE_NAME}

    RANCHER_PATCH_DATA="{\"metadata\":{\"annotations\":{\"kubernetes.io/tls-acme\":\"true\",\"nginx.ingress.kubernetes.io/auth-realm\":\"${OCI_DNS_ZONE_NAME} auth\",\"external-dns.alpha.kubernetes.io/target\":\"${K8S_IO_HOSTNAME}\",\"cert-manager.io/issuer\":null,\"external-dns.alpha.kubernetes.io/ttl\":\"60\"}}}"

    log "Patch Rancher ingress"
    kubectl patch ingress rancher -n cattle-system -p "$RANCHER_PATCH_DATA"  --type=merge

    log "Rollout Rancher"
    kubectl -n cattle-system rollout status -w deploy/rancher || return $?

    log "Create Rancher secrets"
    RANCHER_DATA=$(kubectl --kubeconfig $KUBECONFIG -n cattle-system exec $(kubectl --kubeconfig $KUBECONFIG -n cattle-system get pods -l app=rancher | grep '1/1' | head -1 | awk '{ print $1 }') -- reset-password 2>/dev/null)
    ADMIN_PW=`echo $RANCHER_DATA | awk '{ print $NF }'`
    kubectl -n cattle-system create secret generic rancher-admin-secret --from-literal=password="$ADMIN_PW"
}

function usage {
    consoleerr
    consoleerr "usage: $0 [-n name]"
    consoleerr "  -n name        Environment Name. Required."
    consoleerr "  -d dns_type    DNS type [oci]. Optional.  Defaults to oci."
    consoleerr "  -h             Help"
    consoleerr
    exit 1
}

NAME=""
DNS_TYPE="oci"

while getopts n:d:h flag
do
    case "${flag}" in
        n) NAME=${OPTARG};;
        d) DNS_TYPE=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done

# check environment name length
validate_environment_name $NAME
if [ $? -ne 0 ]; then
  exit 1
fi

if [ $DNS_TYPE != "oci" ]; then
  consoleerr
  consoleerr "Unknown DNS type ${DNS_TYPE}!"
  usage
fi

if [ -z "$NAME" ]; then
    consoleerr
    consoleerr "-n option is required"
    usage
fi

command -v patch >/dev/null 2>&1 || {
    fail "patch is required for dns_type $DNS_TYPE but cannot be found on the path. Aborting.";
}

action "Installing Nginx Ingress Controller" install_nginx_ingress_controller || exit 1
action "Installing cert manager" install_cert_manager || exit 1
action "Installing external DNS" install_external_dns || exit 1
action "Installing Rancher" install_rancher || exit 1
