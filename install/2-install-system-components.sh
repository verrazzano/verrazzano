#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/config.sh

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

set -eu

function install_nginx_ingress_controller()
{
    # Create the namespace for nginx
    if ! kubectl get namespace ingress-nginx ; then
        kubectl create namespace ingress-nginx
    fi

    helm repo add stable https://kubernetes-charts.storage.googleapis.com
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
      ${EXTRA_NGINX_ARGUMENTS} \
      --timeout 15m0s \
      --wait

    if [ ! -z "${patch_service_spec}" ]; then
      kubectl patch service -n ingress-nginx ingress-controller-nginx-ingress-controller -p "${patch_service_spec}"
    fi
}

function setup_cert_manager_crd() {
  curl -L -o "$TMP_DIR/00-crds.yaml" \
    "https://raw.githubusercontent.com/jetstack/cert-manager/release-${CERT_MANAGER_RELEASE}/deploy/manifests/00-crds.yaml"
  if [ "$DNS_TYPE" == "oci" ]; then
    command -v patch >/dev/null 2>&1 || {
      fail "patch is required but cannot be found on the path. Aborting.";
    }
    patch "$TMP_DIR/00-crds.yaml" "$SCRIPT_DIR/config/00-crds.patch"
  fi
}

function setup_cluster_issuer() {
  if [ "$CERT_ISSUER_TYPE" == "acme" ]; then
    local OCI_PRIVATE_KEY_PASSPHRASE=$(get_config_value ".dns.oci.privateKeyPassphrase")
    local OCI_REGION=$(get_config_value ".dns.oci.region")
    local OCI_TENANCY_OCID=$(get_config_value ".dns.oci.tenancyOcid")
    local OCI_USER_OCID=$(get_config_value ".dns.oci.userOcid")
    local OCI_COMPARTMENT_OCID=$(get_config_value ".dns.oci.dnsZoneCompartmentOcid")
    local OCI_FINGERPRINT=$(get_config_value ".dns.oci.fingerprint")
    local OCI_PRIVATE_KEY_FILE=$(get_config_value ".dns.oci.privateKeyFile")
    local EMAIL_ADDRESS=$(get_config_value ".dns.oci.emailAddress")
    local OCI_DNS_ZONE_OCID=$(get_config_value ".dns.oci.dnsZoneOcid")
    local OCI_DNS_ZONE_NAME=$(get_config_value ".dns.oci.dnsZoneName")

    [ ! -f $OCI_PRIVATE_KEY_FILE ] && { fail $OCI_PRIVATE_KEY_FILE does not exist; }

    source /dev/stdin <<<"$(echo 'cat <<EOF >$TMP_DIR/oci.yaml'; cat $SCRIPT_DIR/config/oci.yaml; echo EOF;)"

    kubectl create secret generic -n cert-manager $OCI_DNS_CONFIG_SECRET --from-file=$TMP_DIR/oci.yaml

    kubectl apply -f <(echo "
apiVersion: cert-manager.io/v1alpha2
kind: ClusterIssuer
metadata:
  name: verrazzano-cluster-issuer
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
              name: $OCI_DNS_CONFIG_SECRET
              key: "oci.yaml"
            ocizonename: $DNS_SUFFIX
")
  elif [ "$CERT_ISSUER_TYPE" == "ca" ]; then
    kubectl apply -f <(echo "
apiVersion: cert-manager.io/v1alpha2
kind: ClusterIssuer
metadata:
  name: verrazzano-cluster-issuer
spec:
  ca:
    secretName: $(get_config_value ".certificates.ca.secretName")
")
  else
    fail "certificates issuerType $CERT_ISSUER_TYPE is not supported.";
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

    setup_cert_manager_crd
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
        --set ingressShim.defaultIssuerName=verrazzano-cluster-issuer \
        --set ingressShim.defaultIssuerKind=ClusterIssuer \
        --set clusterResourceNamespace=$(get_config_value ".certificates.clusterResourceNamespace") \
        --wait

    setup_cluster_issuer

    kubectl -n cert-manager rollout status -w deploy/cert-manager
}

function install_external_dns()
{
  if [ "$DNS_TYPE" == "oci" ]; then
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
        --set domainFilters[0]=${DNS_SUFFIX} \
        --set zoneIdFilters[0]=$(get_config_value ".dns.oci.dnsZoneOcid") \
        --set txtOwnerId=v8o-local-${NAME} \
        --set txtPrefix=_v8o-local-${NAME}_ \
        --set policy=sync \
        --set interval=24h \
        --set triggerLoopOnEvent=true \
        --set extraVolumes[0].name=config \
        --set extraVolumes[0].secret.secretName=$OCI_DNS_CONFIG_SECRET \
        --set extraVolumeMounts[0].name=config \
        --set extraVolumeMounts[0].mountPath=/etc/kubernetes/ \
        --wait
  fi
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

    local INGRESS_TLS_SOURCE=""
    local EXTRA_RANCHER_ARGUMENTS=""
    local RANCHER_PATCH_DATA=""
    if [ "$CERT_ISSUER_TYPE" == "acme" ]; then
      INGRESS_TLS_SOURCE="letsEncrypt"
      EXTRA_RANCHER_ARGUMENTS="--set letsEncrypt.ingress.class=rancher --set letsEncrypt.email=$EMAIL_ADDRESS --set letsEncrypt.environment=production"
      RANCHER_PATCH_DATA="{\"metadata\":{\"annotations\":{\"kubernetes.io/tls-acme\":\"true\",\"nginx.ingress.kubernetes.io/auth-realm\":\"${DNS_SUFFIX} auth\",\"external-dns.alpha.kubernetes.io/target\":\"verrazzano-ingress.${NAME}.${DNS_SUFFIX}\",\"cert-manager.io/issuer\":null,\"external-dns.alpha.kubernetes.io/ttl\":\"60\"}}}"
    elif [ "$CERT_ISSUER_TYPE" == "ca" ]; then
      INGRESS_TLS_SOURCE="rancher"
      RANCHER_PATCH_DATA="{\"metadata\":{\"annotations\":{\"kubernetes.io/tls-acme\":\"true\",\"nginx.ingress.kubernetes.io/auth-realm\":\"${NAME}.${DNS_SUFFIX} auth\",\"cert-manager.io/issuer\":\"rancher\",\"cert-manager.io/issuer-kind\":\"Issuer\"}}}"
    else
      fail "certificates issuerType $CERT_ISSUER_TYPE is not supported.";
    fi

    log "Install Rancher"
    helm upgrade rancher rancher-stable/rancher \
      --install --namespace cattle-system \
      --version $RANCHER_VERSION  \
      --set systemDefaultRegistry=ghcr.io/verrazzano \
      --set rancherImage=$RANCHER_IMAGE \
      --set rancherImageTag=$RANCHER_TAG \
      --set hostname=rancher.${NAME}.${DNS_SUFFIX} \
      --set ingress.tls.source=${INGRESS_TLS_SOURCE} \
      ${EXTRA_RANCHER_ARGUMENTS} \
      --wait

    # CRI-O does not deliver MKNOD by default, until https://github.com/rancher/rancher/pull/27582 is merged we must add the capability
    # OLCNE uses CRI-O and needs this change, and it doesn't hurt other cases
    kubectl patch deployments -n cattle-system rancher -p '{"spec":{"template":{"spec":{"containers":[{"name":"rancher","securityContext":{"capabilities":{"add":["MKNOD"]}}}]}}}}'

    log "Patch Rancher ingress"
    kubectl patch ingress rancher -n cattle-system -p "$RANCHER_PATCH_DATA" --type=merge

    log "Rollout Rancher"
    kubectl -n cattle-system rollout status -w deploy/rancher || return $?

    log "Create Rancher secrets"
    RANCHER_DATA=$(kubectl --kubeconfig $KUBECONFIG -n cattle-system exec $(kubectl --kubeconfig $KUBECONFIG -n cattle-system get pods -l app=rancher | grep '1/1' | head -1 | awk '{ print $1 }') -- reset-password 2>/dev/null)
    ADMIN_PW=`echo $RANCHER_DATA | awk '{ print $NF }'`
    kubectl -n cattle-system create secret generic rancher-admin-secret --from-literal=password="$ADMIN_PW"
}

OCI_DNS_CONFIG_SECRET="verrazzano-oci-dns-config"
NAME=$(get_config_value ".environmentName")
DNS_TYPE=$(get_config_value ".dns.type")
CERT_ISSUER_TYPE=$(get_config_value ".certificates.issuerType")

action "Installing Nginx Ingress Controller" install_nginx_ingress_controller || exit 1

# We can only know the ingress IP after installing nginx ingress controller
INGRESS_IP=$(get_verrazzano_ingress_ip)

# DNS_SUFFIX is only used by install_rancher
DNS_SUFFIX=$(get_dns_suffix ${INGRESS_IP})

action "Installing cert manager" install_cert_manager || exit 1
action "Installing external DNS" install_external_dns || exit 1
action "Installing Rancher" install_rancher || exit 1
