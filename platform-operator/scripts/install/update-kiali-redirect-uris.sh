#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

. ${SCRIPT_DIR}/logging.sh

CONFIG_DIR=$SCRIPT_DIR/config
TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

set -ueo pipefail

function update-kiali-redirect-uris {
  local VERRAZZANO_NS=verrazzano-system
  local KEYCLOAK_NS=keycloak
  local KCADMIN_USERNAME=keycloakadmin
  local KCADMIN_SECRET=keycloak-http
  local KC_ADM_PWD=$(kubectl get secret --namespace ${KEYCLOAK_NS} ${KCADMIN_SECRET} -o jsonpath="{.data.password}" | base64 --decode; echo)
  local VZ_SYS_REALM=verrazzano-system

  log "Client ID = $1"
  log "DNS Domain = $2"
  log "Logging in as '$KCADMIN_USERNAME'"

  kubectl exec --stdin keycloak-0 -n keycloak -c keycloak -- bash -s <<EOF
    export PATH="/opt/jboss/keycloak/bin:\$PATH"
    unset JAVA_TOOL_OPTIONS

    kcadm.sh config credentials --server http://localhost:8080/auth --realm master --user ${KCADMIN_USERNAME} --password ${KC_ADM_PWD} || fail "Login failed"

    kcadm.sh update clients/"$1" -r verrazzano-system -f - <<\END
    {
      "redirectUris": [
        "https://verrazzano.$2/*",
        "https://verrazzano.$2/verrazzano/authcallback",
        "https://elasticsearch.vmi.system.$2/*",
        "https://elasticsearch.vmi.system.$2/_authentication_callback",
        "https://prometheus.vmi.system.$2/*",
        "https://prometheus.vmi.system.$2/_authentication_callback",
        "https://grafana.vmi.system.$2/*",
        "https://grafana.vmi.system.$2/_authentication_callback",
        "https://kibana.vmi.system.$2/*",
        "https://kibana.vmi.system.$2/_authentication_callback",
        "https://kiali.vmi.system.$2/*",
        "https://kiali.vmi.system.$2/_authentication_callback",
        "https://jaeger.$2/*"
      ],
      "webOrigins": [
        "https://verrazzano.$2",
        "https://elasticsearch.vmi.system.$2",
        "https://prometheus.vmi.system.$2",
        "https://grafana.vmi.system.$2",
        "https://kibana.vmi.system.$2",
        "https://kiali.vmi.system.$2",
        "https://jaeger.$2"
      ]
    }
END
EOF
}

# Update RedirectUris and WebOrigins in Keycloak
echo "Updating RedirectUris and WebOrigins in Keycloak"
update-kiali-redirect-uris "$1" "$2"
if [ $? -ne 0 ]; then
  echo "Failed to update Keycloak URIs"
    exit 1
fi

