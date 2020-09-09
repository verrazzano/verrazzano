#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh

set -u

KEYCLOAK_NS=keycloak
KCADMIN_USERNAME=keycloakadmin
MYSQL_USERNAME=keycloak
VERRAZZANO_NS=verrazzano-system
VZ_SYS_REALM=verrazzano-system
VZ_USERNAME=verrazzano
DNS_PREFIX="verrazzano-ingress"
TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

function set_INGRESS_IP() {
  if [ ${CLUSTER_TYPE} == "OKE" ]; then
    INGRESS_IP=$(kubectl get svc ingress-controller-nginx-ingress-controller -n ingress-nginx -o json | jq -r '.status.loadBalancer.ingress[0].ip')
  elif [ ${CLUSTER_TYPE} == "OLCNE" ]; then
    # Test for IP from status, if that is not present then assume an on premises installation and use the externalIPs hint
    INGRESS_IP=$(kubectl get svc ingress-controller-nginx-ingress-controller -n ingress-nginx -o json | jq -r '.status.loadBalancer.ingress[0].ip')
    if [ ${INGRESS_IP} == "null" ]; then
      INGRESS_IP=$(kubectl get svc ingress-controller-nginx-ingress-controller -n ingress-nginx -o json  | jq -r '.spec.externalIPs[0]')
    fi
  fi
}

function install_mysql {
  log "Check for Keycloak namespace"
  if ! kubectl get namespace ${KEYCLOAK_NS} 2> /dev/null ; then
    log "Create Keycloak namespace"
    kubectl create namespace ${KEYCLOAK_NS}
  fi

  log "Update MySQL configuration template"
  sed -e "s|MYSQL_IMAGE_TAG|${MYSQL_IMAGE_TAG}|g" \
      -e "s|MYSQL_USERNAME|${MYSQL_USERNAME}|g" \
      $SCRIPT_DIR/config/mysql-values-template.yaml > ${TMP_DIR}/mysql-values-sed.yaml
  
  log "Install MySQL helm chart"
  helm upgrade mysql stable/mysql \
      --install \
      --namespace ${KEYCLOAK_NS} \
      --timeout 10m \
      --wait \
      -f ${TMP_DIR}/mysql-values-sed.yaml
}

function install_keycloak {
  if ! kubectl get secret --namespace ${VERRAZZANO_NS} verrazzano ; then
    error "ERROR: Must run 3-install-verrazzano.sh and then rerun this script."
    exit 1
  fi
  # Replace strings in keycloak.json file
  VZ_PW_SALT=$(kubectl get secret -n ${VERRAZZANO_NS} verrazzano -o jsonpath="{.data.salt}")
  VZ_PW_HASH=$(kubectl get secret -n ${VERRAZZANO_NS} verrazzano -o jsonpath="{.data.hash}")

  sed "s|ENV_NAME|${ENV_NAME}|g;s|DNS_SUFFIX|${DNS_SUFFIX}|g;s|VZ_SYS_REALM|${VZ_SYS_REALM}|g;s|VZ_USERNAME|${VZ_USERNAME}|g;s|VZ_PW_SALT|${VZ_PW_SALT}|g;s|VZ_PW_HASH|${VZ_PW_HASH}|g" $SCRIPT_DIR/config/keycloak.json > ${TMP_DIR}/keycloak-sed.json

  # Create keycloak secret
  kubectl create secret generic keycloak-realm-cacert \
      -n ${KEYCLOAK_NS} \
      --from-file=realm.json=${TMP_DIR}/keycloak-sed.json

  # Add keycloak helm repo
  helm repo add codecentric https://codecentric.github.io/helm-charts
  
  if ! kubectl get secret --namespace ${KEYCLOAK_NS} mysql ; then
    error "ERROR installing mysql. Please rerun this script."
    exit 1
  fi
  # sed keycloak-values-template.yaml file
  sed -e "s|ENV_NAME|${ENV_NAME}|g;s|DNS_SUFFIX|${DNS_SUFFIX}|g" \
      -e "s|KEYCLOAK_IMAGE_TAG|${KEYCLOAK_IMAGE_TAG}|g;s|KCADMIN_USERNAME|${KCADMIN_USERNAME}|g" \
      -e "s|DNS_TARGET_NAME|${DNS_TARGET_NAME}|g;s|MYSQL_USERNAME|${MYSQL_USERNAME}|g" \
      -e "s|MYSQL_PASSWORD|$(kubectl get secret --namespace ${KEYCLOAK_NS} mysql -o jsonpath="{.data.mysql-password}" | base64 --decode; echo)|g" \
      -e "s|KEYCLOAK_IMAGE|$KEYCLOAK_IMAGE|g;s|KEYCLOAK_THEME_IMAGE|$KEYCLOAK_THEME_IMAGE|g" \
      $SCRIPT_DIR/config/keycloak-values-template.yaml > ${TMP_DIR}/keycloak-values-sed.yaml

  # Install keycloak helm chart
  helm upgrade keycloak codecentric/keycloak \
      --install \
      --namespace ${KEYCLOAK_NS} \
      --version ${KEYCLOAK_CHART_VERSION} \
      --wait \
      -f ${TMP_DIR}/keycloak-values-sed.yaml

  kubectl -it exec keycloak-0 \
    -n ${KEYCLOAK_NS} \
    -c keycloak \
    -- bash -c \
    "/opt/jboss/keycloak/bin/kcadm.sh update realms/master -s loginTheme=oracle --no-config --server http://localhost:8080/auth --realm master --user ${KCADMIN_USERNAME} --password \$(cat /etc/keycloak-http/password)"

  # Wait for TLS cert from Cert Manager to go into a ready state
  kubectl wait cert/${ENV_NAME}-secret -n keycloak --for=condition=Ready
}

function set_rancher_server_url
{
    local rancher_host_name="rancher.${ENV_NAME}.${DNS_SUFFIX}"
    local rancher_server_url="https://${rancher_host_name}"
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
    get_rancher_access_token "${rancher_host_name}" "${rancher_admin_password}"
    if [ $? -ne 0 ] ; then
      echo "Failed to get Rancher access token. Continuing without setting Rancher server URL."
      return 0
    fi

    if [ -z "${RANCHER_ACCESS_TOKEN}" ]; then
      echo "Failed to get valid Rancher access token. Continuing without setting Rancher server URL."
      return 0
    fi
    echo "Set Rancher server URL to ${rancher_server_url}"
    curl_args=("${rancher_server_url}/v3/settings/server-url" \
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

function usage {
    error
    error "usage: $0 [-n name] [-d dns_type] [-s dns_suffix]"
    error "  -n name        Environment Name. Optional.  Optional.  Defaults to default."
    error "  -d dns_type    DNS type [xip.io|manual|oci]. Optional.  Defaults to xip.io."
    error "  -s dns_suffix  DNS suffix (e.g v8o.example.com). Not valid for dns_type xip.io. Required for dns-type oci or manual"
    error "  -h             Help"
    error
    exit 1
}

ENV_NAME="default"
DNS_TYPE="xip.io"
DNS_SUFFIX=""

while getopts n:d:s:h flag
do
    case "${flag}" in
        n) ENV_NAME=${OPTARG};;
        d) DNS_TYPE=${OPTARG};;
        s) DNS_SUFFIX=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done

# check environment name length
validate_environment_name ENV_NAME
if [ $? -ne 0 ]; then
  exit 1
fi

# check for valid DNS type
if [ $DNS_TYPE != "xip.io" ] && [ $DNS_TYPE != "oci" ] && [ $DNS_TYPE != "manual" ]; then
  error
  error "Unknown DNS type ${DNS_TYPE}"
  usage
fi
# check for name
if [ $DNS_TYPE = "oci" ]; then
  if [ -z "$ENV_NAME" ]; then
    error
    error "Name must be given with dns_type oci!"
    usage
  fi
fi

if [ $DNS_TYPE = "xip.io" ]; then
  set_INGRESS_IP
fi

# check expected dns suffix for given dns type
if [ -z "$DNS_SUFFIX" ]; then
  if [ $DNS_TYPE == "oci" ] || [ $DNS_TYPE == "manual" ]; then
    error
    error "-s option is required for ${DNS_TYPE}"
    usage
  else
    DNS_SUFFIX="${INGRESS_IP}".xip.io
  fi
else
  if [ $DNS_TYPE = "xip.io" ]; then
    error
    error "A dns_suffix should not be given with dns_type xip.io!"
    usage
  fi
fi

DNS_TARGET_NAME=${DNS_PREFIX}.${ENV_NAME}.${DNS_SUFFIX}

action "Installing MySQL" install_mysql
  if [ "$?" -ne 0 ]; then
    "$SCRIPT_DIR"/k8s-dump-objects.sh -o "pods" -n "${KEYCLOAK_NS}" -m "install_mysql"
    "$SCRIPT_DIR"/k8s-dump-objects.sh -o "jobs" -n "${KEYCLOAK_NS}" -m "install_mysql"
    "$SCRIPT_DIR"/k8s-dump-objects.sh -o "nodes" -n "default" -m "install_mysql"
    log "For additional detailed information on the cluster at the time of this error, please check the diagnostics log file"
    fail "Installation of MySQL failed"
  fi

action "Installing Keycloak" install_keycloak || exit 1
action "Setting Rancher Server URL" set_rancher_server_url || true

rm -rf $TMP_DIR

consoleout
consoleout "Installation Complete."
consoleout
consoleout "Verrazzano provides various user interfaces."
consoleout
consoleout "Grafana - https://grafana.vmi.system.${ENV_NAME}.${DNS_SUFFIX}"
consoleout "Prometheus - https://prometheus.vmi.system.${ENV_NAME}.${DNS_SUFFIX}"
consoleout "Kibana - https://kibana.vmi.system.${ENV_NAME}.${DNS_SUFFIX}"
consoleout "Elasticsearch - https://elasticsearch.vmi.system.${ENV_NAME}.${DNS_SUFFIX}"
consoleout
consoleout "You will need the credentials to access the preceding user interfaces.  They are all accessed by the same username/password."
consoleout "User: verrazzano"
consoleout "Password: kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo"
consoleout
consoleout "Rancher - https://rancher.${ENV_NAME}.${DNS_SUFFIX}"
consoleout "User: admin"
consoleout "Password: kubectl get secret --namespace cattle-system rancher-admin-secret -o jsonpath={.data.password} | base64 --decode; echo"
consoleout
consoleout "Keycloak - https://keycloak.${ENV_NAME}.${DNS_SUFFIX}"
consoleout "User: keycloakadmin"
consoleout "Password: kubectl get secret --namespace keycloak keycloak-http -o jsonpath={.data.password} | base64 --decode; echo"
