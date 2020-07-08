#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle Corporation and/or its affiliates.
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
trap "rm -rf $TMP_DIR" EXIT

function set_INGRESS_IP() {
  if [ ${CLUSTER_TYPE} == "OKE" ]; then
    INGRESS_IP=$(kubectl get svc ingress-controller-nginx-ingress-controller -n ingress-nginx -o json | jq -r '.status.loadBalancer.ingress[0].ip')
  elif [ ${CLUSTER_TYPE} == "KIND" ]; then
    INGRESS_IP=$(kubectl get node ${KIND_CLUSTER_NAME}-control-plane -o json | jq -r '.status.addresses[] | select (.type == "InternalIP") | .address')
  elif [ ${CLUSTER_TYPE} == "OLCNE" ]; then
    # Test for IP from status, if that is not present then assume an on premises installation and use the externalIPs hint
    INGRESS_IP=$(kubectl get svc ingress-controller-nginx-ingress-controller -n ingress-nginx -o json | jq -r '.status.loadBalancer.ingress[0].ip')
    if [ ${INGRESS_IP} == "null" ]; then
      INGRESS_IP=$(kubectl get svc ingress-controller-nginx-ingress-controller -n ingress-nginx -o json  | jq -r '.spec.externalIPs[0]')
    fi
  fi
}

function cleanup_all {
  set +e
  helm uninstall keycloak --namespace ${KEYCLOAK_NS} > /dev/null 2>&1
  helm uninstall mysql --namespace ${KEYCLOAK_NS} > /dev/null 2>&1
  kubectl delete --all pvc --namespace ${KEYCLOAK_NS} > /dev/null 2>&1
  set -e
}

function install_mysql {
  # Create keycloak namespace if it does not exist
  if ! kubectl get namespace ${KEYCLOAK_NS} 2> /dev/null ; then
    kubectl create namespace ${KEYCLOAK_NS}
  fi

  # sed mysql-values-template.yaml file
  sed -e "s|MYSQL_IMAGE_TAG|${MYSQL_IMAGE_TAG}|g" \
      -e "s|MYSQL_USERNAME|${MYSQL_USERNAME}|g" \
      $SCRIPT_DIR/config/mysql-values-template.yaml > ${TMP_DIR}/mysql-values-sed.yaml
  
  # Install mysql helm chart
  helm upgrade mysql stable/mysql \
      --install \
      --namespace ${KEYCLOAK_NS} \
      --wait \
      -f ${TMP_DIR}/mysql-values-sed.yaml
}

function install_keycloak {
  if ! kubectl get secret --namespace ${VERRAZZANO_NS} verrazzano ; then
    consoleerr "ERROR: Must run 3-install-verrazzano.sh and then rerun this script."
    exit 1
  fi
  # Replace strings in keycloak.json file
  VZ_PW_SALT=$(kubectl get secret -n ${VERRAZZANO_NS} verrazzano -o jsonpath="{.data.salt}")
  VZ_PW_HASH=$(kubectl get secret -n ${VERRAZZANO_NS} verrazzano -o jsonpath="{.data.hash}")

  sed "s|ENV_NAME|${ENV_NAME}|g;s|DNS_SUFFIX|${DNS_SUFFIX}|g;s|VZ_SYS_REALM|${VZ_SYS_REALM}|g;s|VZ_USERNAME|${VZ_USERNAME}|g;s|VZ_PW_SALT|${VZ_PW_SALT}|g;s|VZ_PW_HASH|${VZ_PW_HASH}|g" $SCRIPT_DIR/config/keycloak.json > ${TMP_DIR}/keycloak-sed.json

  set +e
  if ! kubectl get secret ${KEYCLOAK_NS} keycloak-realm-cacert 2> /dev/null ; then
      kubectl delete secret keycloak-realm-cacert -n ${KEYCLOAK_NS}
  fi
  set -e

  # Create keycloak secret
  kubectl create secret generic keycloak-realm-cacert \
      -n ${KEYCLOAK_NS} \
      --from-file=realm.json=${TMP_DIR}/keycloak-sed.json

  # Add keycloak helm repo
  helm repo add codecentric https://codecentric.github.io/helm-charts
  
  if ! kubectl get secret --namespace ${KEYCLOAK_NS} mysql ; then
    consoleerr "ERROR installing mysql. Please rerun this script."
    exit 1
  fi
  # sed keycloak-values-template.yaml file
  sed -e "s|ENV_NAME|${ENV_NAME}|g;s|DNS_SUFFIX|${DNS_SUFFIX}|g" \
      -e "s|KEYCLOAK_IMAGE_TAG|${KEYCLOAK_IMAGE_TAG}|g;s|KCADMIN_USERNAME|${KCADMIN_USERNAME}|g" \
      -e "s|DNS_TARGET_NAME|${DNS_TARGET_NAME}|g;s|MYSQL_USERNAME|${MYSQL_USERNAME}|g" \
      -e "s|MYSQL_PASSWORD|$(kubectl get secret --namespace ${KEYCLOAK_NS} mysql -o jsonpath="{.data.mysql-password}" | base64 --decode; echo)|g" \
      -e "s|KEYCLOAK_IMAGE|$KEYCLOAK_IMAGE|g" \
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

function usage {
    consoleerr
    consoleerr "usage: $0 [-n name] [-d dns_type] [-s dns_suffix]"
    consoleerr "  -n name        Environment Name. Optional.  Optional.  Defaults to default."
    consoleerr "  -d dns_type    DNS type [xip.io|manual|oci]. Optional.  Defaults to xip.io."
    consoleerr "  -s dns_suffix  DNS suffix (e.g v8o.example.com). Not valid for dns_type xip.io. Required for dns-type oci or manual"
    consoleerr "  -h             Help"
    consoleerr
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
# check for valid DNS type
if [ $DNS_TYPE != "xip.io" ] && [ $DNS_TYPE != "oci" ] && [ $DNS_TYPE != "manual" ]; then
  consoleerr
  consoleerr "Unknown DNS type ${DNS_TYPE}"
  usage
fi
# check for name
if [ $DNS_TYPE = "oci" ]; then
  if [ -z "$ENV_NAME" ]; then
    consoleerr
    consoleerr "Name must be given with dns_type oci!"
    usage
  fi
fi

if [ $DNS_TYPE = "xip.io" ]; then
  set_INGRESS_IP
fi

# check expected dns suffix for given dns type
if [ -z "$DNS_SUFFIX" ]; then
  if [ $DNS_TYPE == "oci" ] || [ $DNS_TYPE == "manual" ]; then
    consoleerr
    consoleerr "-s option is required for ${DNS_TYPE}"
    usage
  else
    DNS_SUFFIX="${INGRESS_IP}".xip.io
  fi
else
  if [ $DNS_TYPE = "xip.io" ]; then
    consoleerr
    consoleerr "A dns_suffix should not be given with dns_type xip.io!"
    usage
  fi
fi

DNS_TARGET_NAME=${DNS_PREFIX}.${ENV_NAME}.${DNS_SUFFIX}

action "Cleaning up previous installation" cleanup_all || exit 1
action "Installing MySQL" install_mysql || exit 1
action "Installing Keycloak" install_keycloak || exit 1

rm -rf $TMP_DIR

consoleout 
consoleout "Installation Complete."
consoleout 
consoleout "Verrazzano provides various user interfaces."
consoleout 
consoleout "Verrazzano Console - https://console.${ENV_NAME}.${DNS_SUFFIX}"
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
