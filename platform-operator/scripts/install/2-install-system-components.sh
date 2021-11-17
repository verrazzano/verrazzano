#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/config.sh

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

VERRAZZANO_DEFAULT_SECRET_NAMESPACE="cert-manager"
VERRAZZANO_DEFAULT_SECRET_NAME="verrazzano-ca-certificate-secret"
VERRAZZANO_INSTALL_NS=verrazzano-install

# Scaffolding while we move things into the VPO; we need to wait for NGINX to become ready before continuing
function wait_for_nginx() {
  wait_for_deployment ingress-nginx ingress-controller-ingress-nginx-controller
  return $?
}

function wait_for_rancher() {
  wait_for_deployment cattle-system rancher
  return $?
}

function install_nginx_ingress_controller()
{
    local ingress_nginx_ns=ingress-nginx
    local chartName=ingress-controller
    local NGINX_INGRESS_CHART_DIR=${CHARTS_DIR}/ingress-nginx

    if ! is_chart_deployed ${chartName} ${ingress_nginx_ns} ${NGINX_INGRESS_CHART_DIR} ; then
      # Create the namespace for nginx
      if ! kubectl get namespace ${ingress_nginx_ns} ; then
          kubectl create namespace ${ingress_nginx_ns}
          kubectl label namespace ${ingress_nginx_ns} istio-injection=enabled
      fi

      # Handle any additional NGINX install args - since NGINX is for Verrazzano system Ingress,
      # these should be in .ingress.verrazzano.nginxInstallArgs[]
      local EXTRA_NGINX_ARGUMENTS=$(get_nginx_helm_args_from_config)

      if [ "$DNS_TYPE" == "oci" ]; then
        EXTRA_NGINX_ARGUMENTS="$EXTRA_NGINX_ARGUMENTS --set-string controller.service.annotations.external-dns\.alpha\.kubernetes\.io/ttl=60"
        local dns_zone=$(get_config_value ".dns.oci.dnsZoneName")
        EXTRA_NGINX_ARGUMENTS="$EXTRA_NGINX_ARGUMENTS --set controller.service.annotations.external-dns\.alpha\.kubernetes\.io/hostname=verrazzano-ingress.${NAME}.${dns_zone}"
      fi

      local ingress_type=$(get_config_value ".ingress.type")
      EXTRA_NGINX_ARGUMENTS="$EXTRA_NGINX_ARGUMENTS --set controller.service.type=${ingress_type}"

      if [ "${REGISTRY_SECRET_EXISTS}" == "TRUE" ]; then
        if ! kubectl get secret ${GLOBAL_IMAGE_PULL_SECRET} -n ingress-nginx > /dev/null 2>&1 ; then
            action "Copying ${GLOBAL_IMAGE_PULL_SECRET} secret to ${ingress_nginx_ns} namespace" \
              copy_registry_secret ${ingress_nginx_ns}
        fi
        EXTRA_NGINX_ARGUMENTS="$EXTRA_NGINX_ARGUMENTS --set imagePullSecrets[0].name=${GLOBAL_IMAGE_PULL_SECRET}"
      fi

      build_image_overrides ingress-nginx ${chartName}

      helm_install_retry ${chartName} ${NGINX_INGRESS_CHART_DIR} ${ingress_nginx_ns} \
        -f $VZ_OVERRIDES_DIR/ingress-nginx-values.yaml \
        ${EXTRA_NGINX_ARGUMENTS} \
        ${HELM_IMAGE_ARGS} \
        || return $?
    fi

    # Label the ingress-nginx namespace so that we can apply network policies
    log "Adding label needed by network policies to ingress-nginx namespace"
    kubectl label namespace ingress-nginx "verrazzano.io/namespace=ingress-nginx" --overwrite

    # Handle any ports specified for Verrazzano Ingress - these must be patched after install
    local nginx_svc_patch_spec=$(get_verrazzano_ports_spec)
    if [ ! -z "${nginx_svc_patch_spec}" ]; then
      log "Patching NGINX service with: ${nginx_svc_patch_spec}"
      kubectl patch service -n ingress-nginx ingress-controller-ingress-nginx-controller -p "${nginx_svc_patch_spec}"
    fi
    log "Waiting for all the pods in ingress-nginx namespace to reach ready state"
    kubectl wait --for=condition=ready pods --all -n ${ingress_nginx_ns} --timeout=10m
}

function setup_cert_manager_crd() {
  local CERT_MANAGER_MANIFEST_DIR=${MANIFESTS_DIR}/cert-manager
  cp "$CERT_MANAGER_MANIFEST_DIR/cert-manager.crds.yaml" "$TMP_DIR/cert-manager.crds.yaml"
  if [ "$DNS_TYPE" == "oci" ]; then
    command -v patch >/dev/null 2>&1 || {
      fail "patch is required but cannot be found on the path. Aborting.";
    }
    log "Patching cert-manager.crds.yaml to add OCI DNS"
    patch "$TMP_DIR/cert-manager.crds.yaml" "$SCRIPT_DIR/config/cert-manager.crds.patch"
  fi
}

function setup_cluster_issuer() {
  log "In setup_cluster_issuer. Cert Issuer Type = ${CERT_ISSUER_TYPE}"
  if [ "$CERT_ISSUER_TYPE" == "acme" ]; then
    local OCI_DNS_CONFIG_SECRET=$(get_config_value ".dns.oci.ociConfigSecret")
    local EMAIL_ADDRESS=$(get_config_value ".certificates.acme.emailAddress")
    local OCI_DNS_ZONE_OCID=$(get_config_value ".dns.oci.dnsZoneOcid")
    local OCI_DNS_ZONE_NAME=$(get_config_value ".dns.oci.dnsZoneName")

    if ! kubectl get secret $OCI_DNS_CONFIG_SECRET -n $VERRAZZANO_INSTALL_NS ; then
        fail "The OCI Configuration Secret $OCI_DNS_CONFIG_SECRET does not exist in the namespace $VERRAZZANO_INSTALL_NS"
    fi

    acmeURL="https://acme-v02.api.letsencrypt.org/directory"
    if [ "$(get_acme_environment)" != "production" ]; then
      log "Non-production case, using the ACME staging environment"
      acmeURL="https://acme-staging-v02.api.letsencrypt.org/directory"
    fi

    # attempt first kubectl command with retry to ensure that cert-manager webhook is fully initialized
    kubectl_apply_with_retry "
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: verrazzano-cluster-issuer
spec:
  acme:
    email: $EMAIL_ADDRESS
    server: "${acmeURL}"
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
"
  elif [ "$CERT_ISSUER_TYPE" == "ca" ]; then
    if [ $(get_config_value ".certificates.ca.secretName") == "$VERRAZZANO_DEFAULT_SECRET_NAME" ] &&
       [ $(get_config_value ".certificates.ca.clusterResourceNamespace") == "$VERRAZZANO_DEFAULT_SECRET_NAMESPACE" ]; then
    log "Certificate not specified. Creating default Verrazzano Issuer and Certificate in verrazzano-install namespace"

    # attempt first kubectl command with retry to ensure that cert-manager webhook is fully initialized
    kubectl_apply_with_retry "
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: verrazzano-selfsigned-issuer
  namespace: $(get_config_value ".certificates.ca.clusterResourceNamespace")
spec:
  selfSigned: {}
"

    kubectl apply -f <(echo "
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: verrazzano-ca-certificate
  namespace: $(get_config_value ".certificates.ca.clusterResourceNamespace")
spec:
  secretName: $(get_config_value ".certificates.ca.secretName")
  commonName: verrazzano-root-ca
  isCA: true
  issuerRef:
    name: verrazzano-selfsigned-issuer
    kind: Issuer
")
    fi
    kubectl apply -f <(echo "
apiVersion: cert-manager.io/v1
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
    local CERT_MANAGER_CHART_DIR=${CHARTS_DIR}/cert-manager
    local chartName=cert-manager
    local cert_manager_ns=cert-manager

    # Create the namespace for cert-manager
    if ! kubectl get namespace ${cert_manager_ns} ; then
        kubectl create namespace ${cert_manager_ns}
    fi

    setup_cert_manager_crd
    local yaml=$(<"$TMP_DIR/cert-manager.crds.yaml")
    kubectl_apply_with_retry "$yaml" --validate=false

    if ! is_chart_deployed ${chartName} ${cert_manager_ns} ${CERT_MANAGER_CHART_DIR} ; then
      log "cert-manager hasn't been previously installed"
    else
      log "cert-manager has been previously installed"
    fi

    local EXTRA_CERT_MANAGER_ARGUMENTS=""
    if [ "$CERT_ISSUER_TYPE" == "ca" ]; then
      EXTRA_CERT_MANAGER_ARGUMENTS="--set clusterResourceNamespace=$(get_config_value ".certificates.ca.clusterResourceNamespace")"
    fi

    if [ "${REGISTRY_SECRET_EXISTS}" == "TRUE" ]; then
      if ! kubectl get secret ${GLOBAL_IMAGE_PULL_SECRET} -n ${cert_manager_ns} > /dev/null 2>&1 ; then
          action "Copying ${GLOBAL_IMAGE_PULL_SECRET} secret to ${cert_manager_ns} namespace" \
            copy_registry_secret ${cert_manager_ns}
      fi
      EXTRA_CERT_MANAGER_ARGUMENTS="${EXTRA_CERT_MANAGER_ARGUMENTS} --set global.imagePullSecrets[0].name=${GLOBAL_IMAGE_PULL_SECRET}"
    fi

    build_image_overrides cert-manager ${chartName}

    helm_install_retry ${chartName} ${CERT_MANAGER_CHART_DIR} ${cert_manager_ns} \
        --version v1.2.0 \
        -f $VZ_OVERRIDES_DIR/cert-manager-values.yaml \
        ${HELM_IMAGE_ARGS} \
        ${EXTRA_CERT_MANAGER_ARGUMENTS} \
        || return $?

    kubectl -n cert-manager rollout status -w deploy/cert-manager

    log "Waiting for all the pods in cert-manager namespace to reach ready state"
    kubectl wait --for=condition=ready pods --all -n ${cert_manager_ns} --timeout=10m

    log "Waiting for cert-manager-webhook to reach ready state"
    kubectl rollout status deploy/cert-manager-webhook -n ${cert_manager_ns}  --timeout=10m

    setup_cluster_issuer
}

function install_external_dns()
{
  local EXTERNAL_DNS_CHART_DIR=${CHARTS_DIR}/external-dns
  local chartName=external-dns
  local externalDNSNamespace=cert-manager

  if ! kubectl get secret $OCI_DNS_CONFIG_SECRET -n ${externalDNSNamespace} ; then
    # secret does not exist, so copy the configured oci config secret from verrazzano-install namespace.
    # Operator has already checked for existence of secret in verrazzano-install namespace
    # The DNS zone compartment will get appended to secret generated for cert external dns
    local dns_compartment=$(get_config_value ".dns.oci.dnsZoneCompartmentOcid")
    kubectl get secret ${OCI_DNS_CONFIG_SECRET} -n ${VERRAZZANO_INSTALL_NS} -o go-template='{{range $k,$v := .data}}{{if not $v}}{{$v}}{{else}}{{$v | base64decode}}{{end}}{{"\n"}}{{end}}' \
        | sed '/^$/d' > $TMP_DIR/oci.yaml
    echo "compartment: $dns_compartment" >> $TMP_DIR/oci.yaml
    kubectl create secret generic $OCI_DNS_CONFIG_SECRET --from-file=$TMP_DIR/oci.yaml -n ${externalDNSNamespace}
  fi

  if ! is_chart_deployed ${chartName} ${externalDNSNamespace} ${EXTERNAL_DNS_CHART_DIR} ; then
    local extraExternalDNSArgs=""
    if [ "${REGISTRY_SECRET_EXISTS}" == "TRUE" ]; then
      if ! kubectl get secret ${GLOBAL_IMAGE_PULL_SECRET} -n ${externalDNSNamespace} > /dev/null 2>&1 ; then
          action "Copying ${GLOBAL_IMAGE_PULL_SECRET} secret to ${externalDNSNamespace} namespace" \
            copy_registry_secret ${externalDNSNamespace}
      fi
      extraExternalDNSArgs="${extraExternalDNSArgs} --set global.imagePullSecrets[0]=${GLOBAL_IMAGE_PULL_SECRET}"
    fi

    build_image_overrides external-dns ${chartName}

    helm_install_retry ${chartName} ${EXTERNAL_DNS_CHART_DIR} ${externalDNSNamespace} \
        -f $VZ_OVERRIDES_DIR/external-dns-values.yaml \
        ${HELM_IMAGE_ARGS} \
        --set domainFilters[0]=${DNS_SUFFIX} \
        --set zoneIdFilters[0]=$(get_config_value ".dns.oci.dnsZoneOcid") \
        --set txtOwnerId=v8o-local-${NAME}-${TIMESTAMP} \
        --set txtPrefix=_v8o-local-${NAME}-${TIMESTAMP}_ \
        --set extraVolumes[0].name=config \
        --set extraVolumes[0].secret.secretName=$OCI_DNS_CONFIG_SECRET \
        --set extraVolumeMounts[0].name=config \
        --set extraVolumeMounts[0].mountPath=/etc/kubernetes/ \
        ${extraExternalDNSArgs} \
        || return $?
    fi
}

function kubectl_apply_with_retry() {
  local count=0
  local ret=0
  until kubectl apply -f <(echo "$1") "${@:2}"; do
    ret=$?
    count=$((count+1))
    if [[ "$count" -lt 60 ]]; then
      echo "kubectl apply failed, waiting for 5 seconds and trying again"
      sleep 5
    else
      echo "kubectl apply attempt timed out."
      break
    fi
  done

  if [ $ret -ne 0 ]; then
    echo "kubectl apply failed with non-zero return code."
  else
    echo "kubectl apply succeeded."
  fi
  return $ret
}

REGISTRY_SECRET_EXISTS=$(check_registry_secret_exists)

OCI_DNS_CONFIG_SECRET=$(get_config_value ".dns.oci.ociConfigSecret")
NAME=$(get_config_value ".environmentName")
TIMESTAMP=$(date +%s)
DNS_TYPE=$(get_config_value ".dns.type")
CERT_ISSUER_TYPE=$(get_config_value ".certificates.issuerType")

platform_operator_install_message "NGINX Ingress Controller"
action "Wait for NGINX availability" wait_for_nginx || exit 1

# Turn on fail on error/unset variables
set -eu

# We can only know the ingress IP after installing nginx ingress controller
INGRESS_IP=$(get_verrazzano_ingress_ip)

DNS_SUFFIX=$(get_dns_suffix ${INGRESS_IP})

action "Installing cert manager" install_cert_manager || exit 1
if [ "$DNS_TYPE" == "oci" ]; then
  action "Installing external DNS" install_external_dns || exit 1
fi

platform_operator_install_message "Rancher"
action "Wait for Rancher availability" wait_for_rancher || exit 1
