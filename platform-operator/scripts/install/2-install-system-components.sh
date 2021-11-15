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

function ensure_rancher_admin_user() {
  log "Ensure default Rancher admin user is present"
  local STDERROR_FILE="${TMP_DIR}/rancher_ensureadminuser.err"
  kubectl --kubeconfig $KUBECONFIG -n cattle-system exec $(kubectl --kubeconfig $KUBECONFIG -n cattle-system get pods -l app=rancher | grep '1/1' | head -1 | awk '{ print $1 }') -- ensure-default-admin > /dev/null 2>$STDERROR_FILE
  local max_retries=5
  local retries=0
  while true ; do
    RANCHER_ADMIN_USERNAME=$(kubectl get users -l authz.management.cattle.io/bootstrapping=admin-user -o jsonpath={'.items[].username'} || true)
    if [ -z "${RANCHER_ADMIN_USERNAME}" ] ; then
      sleep 10
    else
      log "Rancher admin user: ${RANCHER_ADMIN_USERNAME}"
      break
    fi
    ((retries+=1))
    if [ "$retries" -ge "$max_retries" ] ; then
      echo "Could not detect default Rancher admin user"
      local std_error_file=$(cat $STDERROR_FILE)
      log "${std_error_file}"
      rm "$STDERROR_FILE"
      return 1
    fi
    log "Retry Rancher admin user lookup..."
  done
  return 0
}

function reset_rancher_admin_password() {
  if kubectl get secret cattle-system -n rancher-admin-secret 2>&1 > /dev/null ; then
    log "Rancher admin secret exists, skipping"
    return 0
  fi

  ensure_rancher_admin_user || return $?

  log "Reset Rancher admin password and create secrets"
  local STDERROR_FILE="${TMP_DIR}/rancher_resetpwd.err"
  local max_retries=5
  local retries=0
  while true ; do
    RANCHER_DATA=$(kubectl --kubeconfig $KUBECONFIG -n cattle-system exec $(kubectl --kubeconfig $KUBECONFIG -n cattle-system get pods -l app=rancher | grep '1/1' | head -1 | awk '{ print $1 }') -- reset-password 2>$STDERROR_FILE)
    ADMIN_PW=$(echo -n $RANCHER_DATA | awk 'END{ print $NF }')

    if [ -z "$ADMIN_PW" ] ; then
      sleep 10
    else
      break
    fi
    ((retries+=1))
    if [ "$retries" -ge "$max_retries" ] ; then
      error "ERROR: Failed to reset Rancher password"
      local std_error_file=$(cat $STDERROR_FILE)
      log "${std_error_file}"
      rm "$STDERROR_FILE"
      return 1
    fi
    log "Retry Rancher admin password reset..."
  done

  update_secret_from_literal rancher-admin-secret cattle-system "$ADMIN_PW"
}

function create_cattle_system_namespace()
{
    if ! kubectl get namespace cattle-system > /dev/null 2>&1; then
        kubectl create namespace cattle-system
    fi

    log "Adding label needed by Rancher network policies to cattle-system namespace"
    kubectl label namespace cattle-system "verrazzano.io/namespace=cattle-system" --overwrite
}

function install_rancher()
{
    local RANCHER_CHART_DIR=${CHARTS_DIR}/rancher

    # Create the rancher-operator-system namespace so we can create network policies
    if ! kubectl get namespace rancher-operator-system > /dev/null 2>&1; then
        kubectl create namespace rancher-operator-system
    fi

    local INGRESS_TLS_SOURCE=""
    local EXTRA_RANCHER_ARGUMENTS=""
    local RANCHER_PATCH_DATA=""
    local useAdditionalCAs=false
    # DONE
    if ! is_chart_deployed rancher cattle-system ${RANCHER_CHART_DIR} ; then
         # CERT_ISSUER_TYPE=$(get_config_value ".certificates.issuerType")
      if [ "$CERT_ISSUER_TYPE" == "acme" ]; then
        INGRESS_TLS_SOURCE="letsEncrypt"

#          if [ -z "$(get_config_value '.certificates.acme.environment')" ]; then
#            echo "production"
#          else
#            get_config_value ".certificates.acme.environment"
#          fi
        if [ "$(get_acme_environment)" != "production" ]; then
          log "Using ACME staging, enable use of additional trusted CAs for Rancher"
          useAdditionalCAs=true
        fi
         # Defer to append overrides
        EXTRA_RANCHER_ARGUMENTS="--set letsEncrypt.ingress.class=rancher --set letsEncrypt.email=$(get_config_value ".certificates.acme.emailAddress") --set letsEncrypt.environment=$(get_acme_environment) --set additionalTrustedCAs=${useAdditionalCAs}"
        # Defer to Patch Ingress in post-install
        RANCHER_PATCH_DATA="{\"metadata\":{\"annotations\":{\"kubernetes.io/tls-acme\":\"true\",\"nginx.ingress.kubernetes.io/auth-realm\":\"${DNS_SUFFIX} auth\",\"external-dns.alpha.kubernetes.io/target\":\"verrazzano-ingress.${NAME}.${DNS_SUFFIX}\",\"cert-manager.io/issuer\":null,\"cert-manager.io/issuer-kind\":null,\"external-dns.alpha.kubernetes.io/ttl\":\"60\"}}}"
      elif [ "$CERT_ISSUER_TYPE" == "ca" ]; then
        INGRESS_TLS_SOURCE="secret"
        if [ $(get_config_value ".certificates.ca.secretName") == "$VERRAZZANO_DEFAULT_SECRET_NAME" ] &&
           [ $(get_config_value ".certificates.ca.clusterResourceNamespace") == "$VERRAZZANO_DEFAULT_SECRET_NAMESPACE" ]; then
          EXTRA_RANCHER_ARGUMENTS="--set privateCA=true"
          local retries=0
          local max_retries=60
          until [ "$retries" -ge "$max_retries" ]
          do
            log "Waiting for secret $VERRAZZANO_DEFAULT_SECRET_NAME in namespace $VERRAZZANO_DEFAULT_SECRET_NAMESPACE to be created."
            if kubectl -n $VERRAZZANO_DEFAULT_SECRET_NAMESPACE get secret $VERRAZZANO_DEFAULT_SECRET_NAME >/dev/null 2>&1 ; then
              kubectl -n $VERRAZZANO_DEFAULT_SECRET_NAMESPACE get secret $VERRAZZANO_DEFAULT_SECRET_NAME -o jsonpath='{.data.ca\.crt}' | base64 --decode > ${TMP_DIR}/cacerts.pem || return $?
              break
            fi
            retries=$(($retries+1))
            sleep 1
          done
          if [ "$retries" -ge "$max_retries" ]; then
            fail "Failed to get secret $VERRAZZANO_DEFAULT_SECRET_NAME in namespace $VERRAZZANO_DEFAULT_SECRET_NAMESPACE.";
          fi
          log "Copy CA certificate which is used by the Rancher Agent to validate the connection to the server."
          kubectl -n cattle-system create secret generic tls-ca --from-file=cacerts.pem=${TMP_DIR}/cacerts.pem || return $?
        fi
        # Defer to Patch ingress in post install

        RANCHER_PATCH_DATA="{\"metadata\":{\"annotations\":{\"kubernetes.io/tls-acme\":\"true\",\"nginx.ingress.kubernetes.io/auth-realm\":\"${NAME}.${DNS_SUFFIX} auth\",\"cert-manager.io/cluster-issuer\":\"verrazzano-cluster-issuer\"}}}"
      else
        fail "certificates issuerType $CERT_ISSUER_TYPE is not supported.";
      fi

      # Settings required to point Rancher at a registry for background helm install
      # DONE
      if [ -n "${REGISTRY}" ]; then
        local sys_default_reg=${REGISTRY}

        if [ -n "${IMAGE_REPO}" ]; then
          sys_default_reg=${REGISTRY}/${IMAGE_REPO}
        fi

        EXTRA_RANCHER_ARGUMENTS="${EXTRA_RANCHER_ARGUMENTS} --set systemDefaultRegistry=${sys_default_reg} --set useBundledSystemChart=true"
      fi

      if [ "$useAdditionalCAs" = "true" ] && ! kubectl -n cattle-system get secret tls-ca-additional 2>&1 > /dev/null ; then
        log "Using ACME staging, create staging certs secret for Rancher"
        local acme_staging_certs=${TMP_DIR}/ca-additional.pem
        echo -n "" > ${acme_staging_certs}
        curl_args=(--output ${TMP_DIR}/int-r3.pem "https://letsencrypt.org/certs/staging/letsencrypt-stg-int-r3.pem")
        call_curl 200 http_response http_status curl_args || true
        if [ ${http_status:--1} -ne 200 ]; then
          log "Error downloading LetsEncrypt Staging intermediate R3 cert"
        else
          cat ${TMP_DIR}/int-r3.pem >> ${acme_staging_certs}
        fi
        curl_args=(--output ${TMP_DIR}/int-e1.pem "https://letsencrypt.org/certs/staging/letsencrypt-stg-int-e1.pem")
        call_curl 200 http_response http_status curl_args || true
        if [ ${http_status:--1} -ne 200 ]; then
          log "Error downloading LetsEncrypt Staging intermediate E1 cert"
        else
          cat ${TMP_DIR}/int-e1.pem >> ${acme_staging_certs}
        fi
        curl_args=(--output ${TMP_DIR}/root-x1.pem "https://letsencrypt.org/certs/staging/letsencrypt-stg-root-x1.pem")
        call_curl 200 http_response http_status curl_args || true
        if [ ${http_status:--1} -ne 200 ]; then
          log "Error downloading LetsEncrypt Staging X1 Root cert"
        else
          cat ${TMP_DIR}/root-x1.pem >> ${acme_staging_certs}
        fi
        kubectl -n cattle-system create secret generic tls-ca-additional --from-file=ca-additional.pem=${acme_staging_certs}
      fi

      local chart_name=rancher
      build_image_overrides rancher ${chart_name}


#  local dns_type=$(get_config_value '.dns.type')
#  if [ "$dns_type" == "external" ]; then
#    echo "true"
#  else
#    echo "false"
#  fi
      # Check if this install is using a dns type "external".
      if [ $(is_external_dns) == "true" ]; then # We do not need this check
        log "Installing cattle-system/${chart_name}"
        # Do not add --wait since helm install will not fully work in OLCNE until MKNOD is added in the next command
        helm upgrade ${chart_name} ${RANCHER_CHART_DIR} \
          --install --namespace cattle-system \
          --set hostname=rancher.${NAME}.${DNS_SUFFIX} \
          --set ingress.tls.source=${INGRESS_TLS_SOURCE} \
          ${HELM_IMAGE_ARGS} \
          ${IMAGE_PULL_SECRETS_ARGUMENT} \
          ${EXTRA_RANCHER_ARGUMENTS} \
          || return $?
      else
        helm_install_retry ${chart_name} ${RANCHER_CHART_DIR} cattle-system \
          --set hostname=rancher.${NAME}.${DNS_SUFFIX} \
          --set ingress.tls.source=${INGRESS_TLS_SOURCE} \
          ${HELM_IMAGE_ARGS} \
          ${IMAGE_PULL_SECRETS_ARGUMENT} \
          ${EXTRA_RANCHER_ARGUMENTS} \
          || return $?
      fi
    fi

    # CRI-O does not deliver MKNOD by default, until https://github.com/rancher/rancher/pull/27582 is merged we must add the capability
    # OLCNE uses CRI-O and needs this change, and it doesn't hurt other cases
    kubectl patch deployments -n cattle-system rancher -p '{"spec":{"template":{"spec":{"containers":[{"name":"rancher","securityContext":{"capabilities":{"add":["MKNOD"]}}}]}}}}'

    log "Patch Rancher ingress"
    kubectl patch ingress rancher -n cattle-system -p "$RANCHER_PATCH_DATA" --type=merge

    log "Rollout Rancher"
    kubectl -n cattle-system rollout status -w deploy/rancher || return $?

    log "Waiting for Rancher TLS cert to reach ready state"
    kubectl wait --for=condition=ready cert tls-rancher-ingress -n cattle-system

    # Make sure rancher ingress has an IP
    wait_for_ingress_ip rancher cattle-system || exit 1

    # TODO
    reset_rancher_admin_password || return $?
}

function set_rancher_server_url
{
    echo "Get Rancher admin password."
    rancher_admin_password=$(kubectl get secret --namespace cattle-system rancher-admin-secret -o jsonpath={.data.password})
    if [ $? -ne 0 ]; then
      echo "Failed to get Rancher admin password. Continuing without setting Rancher server URL."
      return 0
    fi
    rancher_admin_password=$(echo ${rancher_admin_password} | base64 --decode)
    if [ $? -ne 0 ]; then
      echo "Failed to decode Rancher admin password. Continuing without setting Rancher server URL."
      return 0
    fi
    echo "Get Rancher access token."
    get_rancher_access_token "${RANCHER_HOSTNAME}" "${rancher_admin_password}"
    if [ $? -ne 0 ] ; then
      echo "Failed to get Rancher access token. Continuing without setting Rancher server URL."
      return 0
    fi

    if [ -z "${RANCHER_ACCESS_TOKEN}" ]; then
      echo "Failed to get valid Rancher access token. Continuing without setting Rancher server URL."
      return 0
    fi
    echo "Set Rancher server URL to https://${RANCHER_HOSTNAME}"
    curl_args=("https://${RANCHER_HOSTNAME}:$(get_nginx_nodeport)/v3/settings/server-url" $(get_rancher_resolve ${RANCHER_HOSTNAME}) \
          -H 'content-type: application/json' \
          -H "Authorization: Bearer ${RANCHER_ACCESS_TOKEN}" \
          -X PUT \
          --data-binary '{"name":"server-url","value":"'https://${RANCHER_HOSTNAME}'"}' \
          --insecure)
    call_curl 200 http_response http_status curl_args || true
    if [ ${http_status:--1} -ne 200 ]; then
      echo "Failed to set Rancher server URL. Continuing without setting Rancher server URL."
      return 0
    else
      echo "Successfully set Rancher server URL."
    fi
}

function wait_for_rancher_agent_to_exist {
    retries=0
    until kubectl -n cattle-system get deploy | grep cattle-cluster-agent; do
      retries=$(($retries+1))
      sleep 2
      if [ "$retries" -ge 30 ] ; then
        break
      fi
    done
}

function patch_rancher_agents() {
    local rancher_in_cluster_host=$(get_rancher_in_cluster_host ${RANCHER_HOSTNAME})

    if [ ${RANCHER_HOSTNAME} != ${rancher_in_cluster_host} ]; then
        local patch_data='{"spec":{"template":{"spec":{"hostAliases":[{"hostnames":["'"${RANCHER_HOSTNAME}"'"],"ip":"'"${rancher_in_cluster_host}"'"}]}}}}'

        wait_for_rancher_agent_to_exist

        # only when cattle-cluster-agent is deployed
        kubectl -n cattle-system get deploy/cattle-cluster-agent
        if [ $? -eq 0 ]; then
            echo "cattle-cluster-agent is deployed.  Continue with patching cattle-cluster-agent."
            kubectl -n cattle-system patch deployments cattle-cluster-agent --patch ${patch_data}
        else
            echo "cattle-cluster-agent is not deployed.  Skip patching."
        fi

        # only when cattle-node-agent is deployed
        kubectl -n cattle-system get daemonset/cattle-node-agent
        if [ $? -eq 0 ]; then
            echo "cattle-node-agent is deployed.  Continue with patching cattle-node-agent."
            kubectl -n cattle-system patch daemonsets cattle-node-agent --patch ${patch_data}
        else
            echo "cattle-node-agent is not deployed.  Skip patching."
        fi
    else
        echo "Rancher host is the same from inside and outside the cluster.  No need to patch agents."
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

#  local ingress_ip=$1
#  local dns_type=$(get_config_value ".dns.type")
#  if [ $dns_type == "wildcard" ]; then
#    dns_suffix="${ingress_ip}".$(get_config_value ".dns.wildcard.domain")
#  elif [ $dns_type == "oci" ]; then
#    dns_suffix=$(get_config_value ".dns.oci.dnsZoneName")
#  elif [ $dns_type == "external" ]; then
#    dns_suffix=$(get_config_value ".dns.external.suffix")
#  fi
#  echo ${dns_suffix}
# DNS_SUFFIX is only used by install_rancher
DNS_SUFFIX=$(get_dns_suffix ${INGRESS_IP})

RANCHER_HOSTNAME=rancher.${NAME}.${DNS_SUFFIX}

action "Installing cert manager" install_cert_manager || exit 1
if [ "$DNS_TYPE" == "oci" ]; then
  action "Installing external DNS" install_external_dns || exit 1
fi

if [ $(is_rancher_enabled) == "true" ]; then
  action "Installing Rancher" install_rancher || exit 1
  action "Setting Rancher Server URL" set_rancher_server_url || true
  action "Patching Rancher Agents" patch_rancher_agents || true
fi
