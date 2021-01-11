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

set -eu

function install_nginx_ingress_controller()
{
    local NGINX_INGRESS_CHART_DIR=${CHARTS_DIR}/ingress-nginx

    # Create the namespace for nginx
    if ! kubectl get namespace ingress-nginx ; then
        kubectl create namespace ingress-nginx
    fi

    # Handle any additional NGINX install args - since NGINX is for Verrazzano system Ingress,
    # these should be in .ingress.verrazzano.nginxInstallArgs[]
    local EXTRA_NGINX_ARGUMENTS=$(get_nginx_helm_args_from_config)

    if [ "$DNS_TYPE" == "oci" ]; then
      EXTRA_NGINX_ARGUMENTS="$EXTRA_NGINX_ARGUMENTS --set controller.service.annotations.external-dns\.alpha\.kubernetes\.io/ttl=\"60\""
      local dns_zone=$(get_config_value ".dns.oci.dnsZoneName")
      EXTRA_NGINX_ARGUMENTS="$EXTRA_NGINX_ARGUMENTS --set controller.service.annotations.external-dns\.alpha\.kubernetes\.io/hostname=verrazzano-ingress.${NAME}.${dns_zone}"
    fi

    local ingress_type=$(get_config_value ".ingress.type")
    EXTRA_NGINX_ARGUMENTS="$EXTRA_NGINX_ARGUMENTS --set controller.service.type=${ingress_type}"

    helm upgrade ingress-controller ${NGINX_INGRESS_CHART_DIR} --install \
      --namespace ingress-nginx \
      -f $SCRIPT_DIR/components/ingress-nginx-values.yaml \
      ${EXTRA_NGINX_ARGUMENTS} \
      --timeout 15m0s \
      --wait \
      || return $?

    # Handle any ports specified for Verrazzano Ingress - these must be patched after install
    local nginx_svc_patch_spec=$(get_verrazzano_ports_spec)
    if [ ! -z "${nginx_svc_patch_spec}" ]; then
      log "Patching NGINX service with: ${nginx_svc_patch_spec}"
      kubectl patch service -n ingress-nginx ingress-controller-ingress-nginx-controller -p "${nginx_svc_patch_spec}"
    fi
}

function setup_cert_manager_crd() {
  local CERT_MANAGER_MANIFEST_DIR=${MANIFESTS_DIR}/cert-manager
  cp "$CERT_MANAGER_MANIFEST_DIR/00-crds.yaml" "$TMP_DIR/00-crds.yaml"
  if [ "$DNS_TYPE" == "oci" ]; then
    command -v patch >/dev/null 2>&1 || {
      fail "patch is required but cannot be found on the path. Aborting.";
    }
    patch "$TMP_DIR/00-crds.yaml" "$SCRIPT_DIR/config/00-crds.patch"
  fi
}

function setup_cluster_issuer() {
  if [ "$CERT_ISSUER_TYPE" == "acme" ]; then
    local OCI_DNS_CONFIG_SECRET=$(get_config_value ".dns.oci.ociConfigSecret")
    local EMAIL_ADDRESS=$(get_config_value ".certificates.acme.emailAddress")
    local OCI_DNS_ZONE_OCID=$(get_config_value ".dns.oci.dnsZoneOcid")
    local OCI_DNS_ZONE_NAME=$(get_config_value ".dns.oci.dnsZoneName")

    if ! kubectl get secret $OCI_DNS_CONFIG_SECRET ; then
        fail "The OCI Configuration Secret $OCI_DNS_CONFIG_SECRET does not exist"
    fi

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
    local CERT_MANAGER_CHART_DIR=${CHARTS_DIR}/cert-manager

    # Create the namespace for cert-manager
    if ! kubectl get namespace cert-manager ; then
        kubectl create namespace cert-manager
    fi

    setup_cert_manager_crd
    kubectl apply -f "$TMP_DIR/00-crds.yaml" --validate=false

    local EXTRA_CERT_MANAGER_ARGUMENTS=""
    if [ "$CERT_ISSUER_TYPE" == "ca" ]; then
      EXTRA_CERT_MANAGER_ARGUMENTS="--set clusterResourceNamespace=$(get_config_value ".certificates.ca.clusterResourceNamespace")"
    fi

    helm upgrade cert-manager ${CERT_MANAGER_CHART_DIR} \
        --install \
        --namespace cert-manager \
        --set image.repository=$CERT_MANAGER_IMAGE \
        --set image.tag=$CERT_MANAGER_TAG \
        --set extraArgs[0]=--acme-http01-solver-image=$CERT_MANAGER_SOLVER_IMAGE:$CERT_MANAGER_SOLVER_TAG \
        --set cainjector.enabled=false \
        --set webhook.enabled=false \
        --set webhook.injectAPIServerCA=false \
        --set ingressShim.defaultIssuerName=verrazzano-cluster-issuer \
        --set ingressShim.defaultIssuerKind=ClusterIssuer \
        ${EXTRA_CERT_MANAGER_ARGUMENTS} \
        --wait \
        || return $?

    setup_cluster_issuer

    kubectl -n cert-manager rollout status -w deploy/cert-manager
}

function install_external_dns()
{
  local EXTERNAL_DNS_CHART_DIR=${CHARTS_DIR}/external-dns

  if [ "$DNS_TYPE" == "oci" ]; then
    if ! kubectl get secret $OCI_DNS_CONFIG_SECRET -n cert-manager ; then
      # secret does not exist, so copy the configured oci config secret from default namespace.
      # Operator has already checked for existence of secret in default namespace
      # The DNS zone compartment will get appended to secret generated for cert external dns
      local dns_compartment=$(get_config_value ".dns.oci.dnsZoneCompartmentOcid")
      kubectl get secret ${OCI_DNS_CONFIG_SECRET} -o go-template='{{range $k,$v := .data}}{{if not $v}}{{$v}}{{else}}{{$v | base64decode}}{{end}}{{"\n"}}{{end}}' \
          | sed '/^$/d' > $TMP_DIR/oci.yaml
      echo "compartment: $dns_compartment" >> $TMP_DIR/oci.yaml
      kubectl create secret generic $OCI_DNS_CONFIG_SECRET --from-file=$TMP_DIR/oci.yaml -n cert-manager
    fi

    helm upgrade external-dns ${EXTERNAL_DNS_CHART_DIR} \
        --install \
        --namespace cert-manager \
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
        --wait \
        || return $?
  fi
}

function install_rancher()
{
    local RANCHER_CHART_DIR=${CHARTS_DIR}/rancher

    log "Create Rancher namespace (if required)"
    if ! kubectl get namespace cattle-system > /dev/null 2>&1; then
        kubectl create namespace cattle-system
    fi

    local INGRESS_TLS_SOURCE=""
    local EXTRA_RANCHER_ARGUMENTS=""
    local RANCHER_PATCH_DATA=""
    if [ "$CERT_ISSUER_TYPE" == "acme" ]; then
      INGRESS_TLS_SOURCE="letsEncrypt"
      EXTRA_RANCHER_ARGUMENTS="--set letsEncrypt.ingress.class=rancher --set letsEncrypt.email=$(get_config_value ".certificates.acme.emailAddress") --set letsEncrypt.environment=$(get_acme_environment)"
      RANCHER_PATCH_DATA="{\"metadata\":{\"annotations\":{\"kubernetes.io/tls-acme\":\"true\",\"nginx.ingress.kubernetes.io/auth-realm\":\"${DNS_SUFFIX} auth\",\"external-dns.alpha.kubernetes.io/target\":\"verrazzano-ingress.${NAME}.${DNS_SUFFIX}\",\"cert-manager.io/issuer\":null,\"external-dns.alpha.kubernetes.io/ttl\":\"60\"}}}"
    elif [ "$CERT_ISSUER_TYPE" == "ca" ]; then
      INGRESS_TLS_SOURCE="rancher"
      RANCHER_PATCH_DATA="{\"metadata\":{\"annotations\":{\"kubernetes.io/tls-acme\":\"true\",\"nginx.ingress.kubernetes.io/auth-realm\":\"${NAME}.${DNS_SUFFIX} auth\",\"cert-manager.io/issuer\":\"rancher\",\"cert-manager.io/issuer-kind\":\"Issuer\"}}}"
    else
      fail "certificates issuerType $CERT_ISSUER_TYPE is not supported.";
    fi

    log "Install Rancher"
    # Do not add --wait since helm install will not fully work in OLCNE until MKNOD is added in the next command
    helm upgrade rancher ${RANCHER_CHART_DIR} \
      --install --namespace cattle-system \
      --set systemDefaultRegistry=ghcr.io/verrazzano \
      --set rancherImage=$RANCHER_IMAGE \
      --set rancherImageTag=$RANCHER_TAG \
      --set hostname=rancher.${NAME}.${DNS_SUFFIX} \
      --set ingress.tls.source=${INGRESS_TLS_SOURCE} \
      ${EXTRA_RANCHER_ARGUMENTS}

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

    if [ -z "$ADMIN_PW" ] ; then
      error "ERROR: Failed to reset Rancher password"
      return 1
    fi

    kubectl -n cattle-system create secret generic rancher-admin-secret --from-literal=password="$ADMIN_PW"
}

function set_rancher_server_url
{
    local rancher_server_url="https://${RANCHER_HOSTNAME}"
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
    echo "Set Rancher server URL to ${rancher_server_url}"
    curl_args=("${rancher_server_url}/v3/settings/server-url" $(get_rancher_resolve ${RANCHER_HOSTNAME}) \
          -H 'content-type: application/json' \
          -H "Authorization: Bearer ${RANCHER_ACCESS_TOKEN}" \
          -X PUT \
          --data-binary '{"name":"server-url","value":"'${rancher_server_url}'"}' \
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

OCI_DNS_CONFIG_SECRET=$(get_config_value ".dns.oci.ociConfigSecret")
NAME=$(get_config_value ".environmentName")
DNS_TYPE=$(get_config_value ".dns.type")
CERT_ISSUER_TYPE=$(get_config_value ".certificates.issuerType")

action "Installing NGINX Ingress Controller" install_nginx_ingress_controller || exit 1

# We can only know the ingress IP after installing nginx ingress controller
INGRESS_IP=$(get_verrazzano_ingress_ip)

# DNS_SUFFIX is only used by install_rancher
DNS_SUFFIX=$(get_dns_suffix ${INGRESS_IP})

RANCHER_HOSTNAME=rancher.${NAME}.${DNS_SUFFIX}

action "Installing cert manager" install_cert_manager || exit 1
action "Installing external DNS" install_external_dns || exit 1
action "Installing Rancher" install_rancher || exit 1
action "Setting Rancher Server URL" set_rancher_server_url || true
action "Patching Rancher Agents" patch_rancher_agents || true
