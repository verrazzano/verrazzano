#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/config.sh

set -u

KEYCLOAK_NS=keycloak
KCADMIN_USERNAME=keycloakadmin
KCADMIN_SECRET=keycloak-http
MYSQL_USERNAME=keycloak
VERRAZZANO_NS=verrazzano-system
VZ_SYS_REALM=verrazzano-system
VZ_USERNAME=verrazzano
TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

ENV_NAME=$(get_config_value ".environmentName")

INGRESS_IP=$(get_verrazzano_ingress_ip)
if [ -n "${INGRESS_IP:-}" ]; then
  log "Found ingress address ${INGRESS_IP}"
else
  fail "Failed to find ingress address."
fi

DNS_SUFFIX=$(get_dns_suffix ${INGRESS_IP})

function install_mysql {
  MYSQL_CHART_DIR=${CHARTS_DIR}/mysql

  log "Check for Keycloak namespace"
  if ! kubectl get namespace ${KEYCLOAK_NS} 2> /dev/null ; then
    log "Create Keycloak namespace"
    kubectl create namespace ${KEYCLOAK_NS}
  fi

  # Handle any additional MySQL install args that cannot be in mysql-values.yaml
  local EXTRA_MYSQL_ARGUMENTS=$(get_mysql_helm_args_from_config)
  EXTRA_MYSQL_ARGUMENTS="$EXTRA_MYSQL_ARGUMENTS --set mysqlUser=${MYSQL_USERNAME}"

  echo "CREATE DATABASE IF NOT EXISTS keycloak DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_general_ci;" > ${TMP_DIR}/create-db.sql
  echo "USE keycloak;" >> ${TMP_DIR}/create-db.sql
  echo "GRANT ALL ON keycloak.* TO '${MYSQL_USERNAME}'@'%';" >> ${TMP_DIR}/create-db.sql
  echo "FLUSH PRIVILEGES;" >> ${TMP_DIR}/create-db.sql
  EXTRA_MYSQL_ARGUMENTS="$EXTRA_MYSQL_ARGUMENTS --set-file initializationFiles.create-db=${TMP_DIR}/create-db.sql"

  log "Install MySQL helm chart"
  helm upgrade mysql ${MYSQL_CHART_DIR} \
      --install \
      --namespace ${KEYCLOAK_NS} \
      --timeout 10m \
      --wait \
      -f $VZ_OVERRIDES_DIR/mysql-values.yaml \
      ${EXTRA_MYSQL_ARGUMENTS}
}

function install_keycloak {
  KEYCLOAK_CHART_DIR=${CHARTS_DIR}/keycloak

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

  # Create a random secret for the keycloakadmin user
  kubectl apply -f <(echo "
apiVersion: v1
kind: Secret
metadata:
  name: ${KCADMIN_SECRET}
  namespace: ${KEYCLOAK_NS}
type: Opaque
data:
  password: $(cat /dev/urandom | LC_ALL=C tr -dc "a-zA-Z0-9" | fold -w 10 | head -n 1 | base64)
")

  # Check if using the optional imagePullSecret
  local KEYCLOAK_ARGUMENTS=""
  if [ "${REGISTRY_SECRET_EXISTS}" == "TRUE" ]; then
    if ! kubectl get secret ${GLOBAL_IMAGE_PULL_SECRET} -n ${KEYCLOAK_NS} > /dev/null 2>&1 ; then
        copy_registry_secret "${KEYCLOAK_NS}"
        KEYCLOAK_ARGUMENTS=" --set keycloak.image.pullSecrets[0]=${GLOBAL_IMAGE_PULL_SECRET}"
    fi
  fi

  if ! kubectl get secret --namespace ${KEYCLOAK_NS} mysql ; then
    error "ERROR installing mysql. Please rerun this script."
    exit 1
  fi

  KEYCLOAK_ARGUMENTS="$KEYCLOAK_ARGUMENTS --set keycloak.username=${KCADMIN_USERNAME}"
  KEYCLOAK_ARGUMENTS="$KEYCLOAK_ARGUMENTS --set-string keycloak.ingress.annotations.external-dns\.alpha\.kubernetes\.io/target=${DNS_TARGET_NAME}"
  KEYCLOAK_ARGUMENTS="$KEYCLOAK_ARGUMENTS --set keycloak.ingress.hosts={keycloak.${ENV_NAME}.${DNS_SUFFIX}}"
  KEYCLOAK_ARGUMENTS="$KEYCLOAK_ARGUMENTS --set keycloak.ingress.tls[0].hosts={keycloak.${ENV_NAME}.${DNS_SUFFIX}}"
  KEYCLOAK_ARGUMENTS="$KEYCLOAK_ARGUMENTS --set keycloak.ingress.tls[0].secretName=${ENV_NAME}-secret"
  KEYCLOAK_ARGUMENTS="$KEYCLOAK_ARGUMENTS --set keycloak.persistence.dbPassword=$(kubectl get secret --namespace ${KEYCLOAK_NS} mysql -o jsonpath="{.data.mysql-password}" | base64 --decode; echo)"
  KEYCLOAK_ARGUMENTS="$KEYCLOAK_ARGUMENTS --set keycloak.persistence.dbUser=${MYSQL_USERNAME}"

  # Handle any additional Keycloak install args
  KEYCLOAK_ARGUMENTS="$KEYCLOAK_ARGUMENTS $(get_keycloak_helm_args_from_config)"

  # Install keycloak helm chart
  helm upgrade keycloak ${KEYCLOAK_CHART_DIR} \
      --install \
      --namespace ${KEYCLOAK_NS} \
      -f $VZ_OVERRIDES_DIR/keycloak-values.yaml \
      ${KEYCLOAK_ARGUMENTS} \
      --wait

  kubectl exec keycloak-0 \
    -n ${KEYCLOAK_NS} \
    -c keycloak \
    -- bash -c \
    "/opt/jboss/keycloak/bin/kcadm.sh update realms/master -s loginTheme=oracle --no-config --server http://localhost:8080/auth --realm master --user ${KCADMIN_USERNAME} --password \$(cat /etc/${KCADMIN_SECRET}/password)"

  # Wait for TLS cert from Cert Manager to go into a ready state
  kubectl wait cert/${ENV_NAME}-secret -n keycloak --for=condition=Ready
}

DNS_TARGET_NAME=verrazzano-ingress.${ENV_NAME}.${DNS_SUFFIX}
REGISTRY_SECRET_EXISTS=$(check_registry_secret_exists)

action "Installing MySQL" install_mysql
  if [ "$?" -ne 0 ]; then
    "$SCRIPT_DIR"/k8s-dump-objects.sh -o "pods" -n "${KEYCLOAK_NS}" -m "install_mysql"
    "$SCRIPT_DIR"/k8s-dump-objects.sh -o "jobs" -n "${KEYCLOAK_NS}" -m "install_mysql"
    "$SCRIPT_DIR"/k8s-dump-objects.sh -o "nodes" -n "default" -m "install_mysql"
    log "For additional detailed information on the cluster at the time of this error, please check the diagnostics log file"
    fail "Installation of MySQL failed"
  fi

action "Installing Keycloak" install_keycloak || exit 1

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
consoleout "Verrazzano Console - https://verrazzano.${ENV_NAME}.${DNS_SUFFIX}"
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
consoleout "Password: kubectl get secret --namespace keycloak ${KCADMIN_SECRET} -o jsonpath={.data.password} | base64 --decode; echo"
if [ $(get_application_ingress_ip) == "null" ];
then
  consoleout
  consoleout "WARNING: istio-ingressgateway service does not have a valid external IP assigned yet. Public access to deployed applications will not work."
  consoleout "Use the following command to check if an External IP has been assigned to the gateway."
  consoleout "kubectl get svc istio-ingressgateway -n istio-system"
fi
