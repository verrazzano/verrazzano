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
VERRAZZANO_INTERNAL_PROM_USER=verrazzano-prom-internal
VERRAZZANO_INTERNAL_ES_USER=verrazzano-es-internal
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

# build_extra_init_containers_override overrides the keycloak extraInitContainers helm value with YAML that
# includes the image path constructed from the bill of materials
function build_extra_init_containers_override {
  build_image_overrides keycloak keycloak-oracle-theme
  EXTRA_INIT_CONTAINERS_OVERRIDE="
    - name: theme-provider
      image: ${HELM_RAW_IMAGE}
      imagePullPolicy: IfNotPresent
      command:
        - sh
      args:
        - -c
        - |
          echo \"Copying theme...\"
          cp -R /oracle/* /theme
      volumeMounts:
        - name: theme
          mountPath: /theme
        - name: cacerts
          mountPath: /cacerts"
}


DNS_TARGET_NAME=verrazzano-ingress.${ENV_NAME}.${DNS_SUFFIX}
REGISTRY_SECRET_EXISTS=$(check_registry_secret_exists)

if [ "${REGISTRY_SECRET_EXISTS}" == "TRUE" ]; then
  if ! kubectl get secret ${GLOBAL_IMAGE_PULL_SECRET} -n ${KEYCLOAK_NS} > /dev/null 2>&1 ; then
      copy_registry_secret "${KEYCLOAK_NS}"
  fi
fi

# Scaffolding while we move things into the VPO; we need to wait for MySQL before installing keyclaok
function wait_for_mysql() {
  wait_for_deployment keycloak mysql
  return $?
}


if [ $(is_keycloak_enabled) == "true" ]; then
  action "Waiting for MySQL to become available" wait_for_mysql || exit 1
  action "Installing Keycloak" install_keycloak || exit 1
else
  log "Skip Keycloak installation, disabled"
fi

rm -rf $TMP_DIR

consoleout
consoleout "Insallation Complete."

# Determine the consoles enabled for the profile and display the URLs accordingly
consoleArr=()
if [ "$(is_grafana_console_enabled)" == "true" ]; then
  consoleArr+=("Grafana - https://grafana.vmi.system.${ENV_NAME}.${DNS_SUFFIX}")
fi

if [ "$(is_prometheus_console_enabled)" == "true" ]; then
  consoleArr+=("Prometheus - https://prometheus.vmi.system.${ENV_NAME}.${DNS_SUFFIX}")
fi

if [ "$(is_kibana_console_enabled)" == "true" ]; then
  consoleArr+=("Kibana - https://kibana.vmi.system.${ENV_NAME}.${DNS_SUFFIX}")
fi

if [ "$(is_elasticsearch_console_enabled)" == "true" ]; then
  consoleArr+=("Elasticsearch - https://elasticsearch.vmi.system.${ENV_NAME}.${DNS_SUFFIX}")
fi

if [[ "$(is_vz_console_enabled)" == "true" ]]; then
  consoleArr+=("Verrazzano Console - https://verrazzano.${ENV_NAME}.${DNS_SUFFIX}")
fi

display_warning_for_secret="false"
console_count=${#consoleArr[@]}
if [ $console_count -gt 0 ];then
  display_warning_for_secret="true"
  consoleout
  if [ $console_count -eq 1 ];then
    consoleout "Verrazzano provides one user interface."
  else
    consoleout "Verrazzano provides various user interfaces."
  fi
  consoleout
  consoleout "To get the URL for each Verrazzano interface, run the following command:"
  consoleout "kubectl get vz -o jsonpath={.items[].status.instance} | jq ."
  consoleout
  if [ $console_count -eq 1 ];then
    consoleout "You will need the credentials to access the preceding user interface. The user interface can be accessed by the username/password."
  else
    consoleout "You will need the credentials to access the preceding user interfaces. They are all accessed by the same username/password."
  fi
  consoleout "User: verrazzano"
  consoleout "Password: kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo"
  consoleout
fi
if [ $(is_rancher_enabled) == "true" ]; then
  consoleout "User: admin"
  consoleout "Password: kubectl get secret --namespace cattle-system rancher-admin-secret -o jsonpath={.data.password} | base64 --decode; echo"
  consoleout
  display_warning_for_secret="true"
fi
if [ $(is_keycloak_enabled) == "true" ]; then
  consoleout "Keycloak - https://keycloak.${ENV_NAME}.${DNS_SUFFIX}"
  consoleout "User: keycloakadmin"
  consoleout "Password: kubectl get secret --namespace keycloak ${KCADMIN_SECRET} -o jsonpath={.data.password} | base64 --decode; echo"
  display_warning_for_secret="true"
fi
if [ $display_warning_for_secret == "true" ]; then
  consoleout
  consoleout "WARNING: Editing the secrets will not change the passwords for those accounts."
  consoleout "To change a password, use the appropriate console (Keycloak or Rancher), then update the corresponding secret with the new password."
  consoleout "If you change the password, then you must update the secret."
  consoleout
fi
if [ $(get_application_ingress_ip) == "null" ]; then
  consoleout
  consoleout "WARNING: istio-ingressgateway service does not have a valid external IP assigned yet. Public access to deployed applications will not work."
  consoleout "Use the following command to check if an External IP has been assigned to the gateway."
  consoleout "kubectl get svc istio-ingressgateway -n istio-system"
fi
if [ $(is_oci_dns) == "true" ]; then
  secret_name=$(get_config_value ".dns.oci.ociConfigSecret")
  consoleout
  consoleout "NOTE: The secret \"${secret_name}\" created in the \"verrazzano-install\" namespace prior to installation is only used during the actual installation."
  consoleout "You may delete it now.  DO NOT delete the secret of the same name in the cert-manager namespace."
  consoleout
fi

sync
