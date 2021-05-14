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
KCADMIN_REALM=master
KCADMIN_USERNAME=keycloakadmin
KCADMIN_SECRET=keycloak-http
VERRAZZANO_INTERNAL_PROM_USER=verrazzano-prom-internal
VERRAZZANO_INTERNAL_ES_USER=verrazzano-es-internal
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
  if is_chart_deployed mysql ${KEYCLOAK_NS} ${MYSQL_CHART_DIR} ; then
    return 0
  fi

  log "Check for Keycloak namespace"
  if ! kubectl get namespace ${KEYCLOAK_NS} 2> /dev/null ; then
    log "Create Keycloak namespace"
    kubectl create namespace ${KEYCLOAK_NS}
    # Label the keycloak namespace so that we istio injection is enabled
    log "Adding label needed for istio sidecar injection to keycloak namespace"
    kubectl label namespace keycloak "istio-injection=enabled" --overwrite
    # Label the keycloak namespace so that we can apply network policies
    log "Adding label needed by network policies to keycloak namespace"
    kubectl label namespace keycloak "verrazzano.io/namespace=keycloak" --overwrite
  fi

  # Handle any additional MySQL install args that cannot be in mysql-values.yaml
  local EXTRA_MYSQL_ARGUMENTS=$(get_mysql_helm_args_from_config)
  EXTRA_MYSQL_ARGUMENTS="$EXTRA_MYSQL_ARGUMENTS --set mysqlUser=${MYSQL_USERNAME}"

  echo "CREATE DATABASE IF NOT EXISTS keycloak DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_general_ci;" > ${TMP_DIR}/create-db.sql
  echo "USE keycloak;" >> ${TMP_DIR}/create-db.sql
  # Allow the keycloak user to create/drop tables, indices, foreign key references, and read/write to all tables in keycloak schema
  echo "GRANT CREATE, ALTER, DROP, INDEX, REFERENCES, SELECT, INSERT, UPDATE, DELETE ON keycloak.* TO '${MYSQL_USERNAME}'@'%';" >> ${TMP_DIR}/create-db.sql
  echo "FLUSH PRIVILEGES;" >> ${TMP_DIR}/create-db.sql
  EXTRA_MYSQL_ARGUMENTS="$EXTRA_MYSQL_ARGUMENTS --set-file initializationFiles.create-db\.sql=${TMP_DIR}/create-db.sql"

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

  if ! is_chart_deployed keycloak ${KEYCLOAK_NS} ${KEYCLOAK_CHART_DIR} ; then

    # Create a random secret for the keycloakadmin user
    update_secret_from_literal ${KCADMIN_SECRET} ${KEYCLOAK_NS} "$(generate_password)"

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
  KEYCLOAK_ARGUMENTS="$KEYCLOAK_ARGUMENTS --set-string keycloak.ingress.annotations.nginx\.ingress\.kubernetes\.io/service-upstream=true"
  KEYCLOAK_ARGUMENTS="$KEYCLOAK_ARGUMENTS --set-string keycloak.ingress.annotations.nginx\.ingress\.kubernetes\.io/upstream-vhost=keycloak-http.keycloak.svc.cluster.local"
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
        --timeout 10m \
        --wait
  fi

  VZ_ADMIN_GROUP=$(helm show values ${VZ_CHARTS_DIR}/verrazzano | grep "adminsGroup: &default_adminsGroup " | awk '{ print $3 }')
  VZ_MONITOR_GROUP=$(helm show values ${VZ_CHARTS_DIR}/verrazzano | grep "monitorsGroup: &default_monitorsGroup " | awk '{ print $3 }')
  VZ_USER_GROUP=$(helm show values ${VZ_CHARTS_DIR}/verrazzano | grep "usersGroup: &default_usersGroup " | awk '{ print $3 }')
  VZ_SYSTEM_GROUP=$(helm show values ${VZ_CHARTS_DIR}/verrazzano | grep "systemGroup: &default_systemGroup " | awk '{ print $3 }')

  VPROM=$(generate_password)
  VES=$(generate_password)

  # Create a random secret for the verrazzano-prom-internal user
  kubectl apply -f <(echo "
apiVersion: v1
kind: Secret
metadata:
  name: ${VERRAZZANO_INTERNAL_PROM_USER}
  namespace: ${VERRAZZANO_NS}
type: Opaque
data:
  username: $(echo -n ${VERRAZZANO_INTERNAL_PROM_USER} | base64)
  password: $(echo -n ${VPROM} | base64)
")

  # Create a random secret for the verrazzano-es-internal user
  kubectl apply -f <(echo "
apiVersion: v1
kind: Secret
metadata:
  name: ${VERRAZZANO_INTERNAL_ES_USER}
  namespace: ${VERRAZZANO_NS}
type: Opaque
data:
  username: $(echo -n ${VERRAZZANO_INTERNAL_ES_USER} | base64)
  password: $(echo -n ${VES} | base64)
")

  # Create the verrazzano-system realm and populate it with users, groups, clients, etc.
  configure_keycloak_realms $VZ_SYS_REALM $VZ_ADMIN_GROUP $VZ_MONITOR_GROUP $VZ_USER_GROUP $VZ_SYSTEM_GROUP

  # Wait for TLS cert from Cert Manager to go into a ready state
  kubectl wait cert/${ENV_NAME}-secret -n keycloak --for=condition=Ready
}

function configure_keycloak_realms() {
  local _VZ_REALM="$1"
  local _VZ_ADMIN_GRP="$2"
  local _VZ_MONITOR_GRP="$3"
  local _VZ_USER_GRP="$4"
  local _VZ_SYSTEM_GRP="$5"

  local PW=$(kubectl get secret -n ${VERRAZZANO_NS} verrazzano -o jsonpath="{.data.password}" | base64 -d)

  kubectl exec --stdin keycloak-0 -n keycloak -c keycloak -- bash -s <<EOF
    export PATH="/opt/jboss/keycloak/bin:\$PATH"
    unset JAVA_TOOL_OPTIONS

    function log () {
      echo "\$1"
    }

    function fail () {
      log "\$1"
      exit 1
    }

    log "Logging in as '$KCADMIN_USERNAME'"
    kcadm.sh config credentials --server http://localhost:8080/auth --realm master --user ${KCADMIN_USERNAME} --password \$(cat /etc/${KCADMIN_SECRET}/password) || fail "Login failed"

  ##  log "Deleting realm $_VZ_REALM"
  ##  kcadm.sh delete realms/$_VZ_REALM || log "Failed to delete realm"

    log "Creating $_VZ_REALM realm"
    kcadm.sh create realms -s realm=$_VZ_REALM -s enabled=false || fail "Failed to create realm"

    log "Creating $_VZ_USER_GRP group"
    USERS_GID=\$(kcadm.sh create groups -r $_VZ_REALM -s name=$_VZ_USER_GRP 2>&1 | sed -e "s;^.*'\([^']*\)'.*$;\\1;") || fail "Failed to create group"
    log "Created group \$USERS_GID"

    log "Creating $_VZ_ADMIN_GRP group"
    ADMINS_GID=\$(kcadm.sh create groups -r $_VZ_REALM -s name=$_VZ_ADMIN_GRP 2>&1 | sed -e "s;^.*'\([^']*\)'.*$;\\1;") || fail "Failed to create group"
    log "Created group \$ADMINS_GID"

    log "Creating $_VZ_MONITOR_GRP group"
    MONITORS_GID=\$(kcadm.sh create groups -r $_VZ_REALM -s name=$_VZ_MONITOR_GRP 2>&1 | sed -e "s;^.*'\([^']*\)'.*$;\\1;") || fail "Failed to create group"
    log "Created group \$MONITORS_GID"

    log "Creating $_VZ_SYSTEM_GRP group"
    SYSTEM_GID=\$(kcadm.sh create groups -r $_VZ_REALM -s name=$_VZ_SYSTEM_GRP 2>&1 | sed -e "s;^.*'\([^']*\)'.*$;\\1;") || fail "Failed to create group"
    log "Created group \$SYSTEM_GID"

    log "Creating console_users role"
    kcadm.sh create roles -r $_VZ_REALM -s name=console_users || fail "Failed to create role"

    log "Creating Admin role"
    kcadm.sh create roles -r $_VZ_REALM -s name=Admin || fail "Failed to create role"

    log "Creating Viewer role"
    kcadm.sh create roles -r $_VZ_REALM -s name=Viewer || fail "Failed to create role"

    log "Granting console_users role to $_VZ_USER_GRP group"
    kcadm.sh add-roles -r $_VZ_REALM --gid \$USERS_GID --rolename console_users || log "Failed to grant role"

    log "Granting Admin role to $_VZ_ADMIN_GRP group"
    kcadm.sh add-roles -r $_VZ_REALM --gid \$ADMINS_GID --rolename Admin || log "Failed to grant role"

    log "Granting Viewer role to $_VZ_MONITOR_GRP group"
    kcadm.sh add-roles -r $_VZ_REALM --gid \$MONITORS_GID --rolename Viewer || log "Failed to grant role"

    log "Adding console_users to default roles"
    EXISTING=\$(kcadm.sh get realms/$_VZ_REALM --fields defaultRoles --format csv) || fail "Failed to get existing default roles"
    kcadm.sh update realms/$_VZ_REALM -s "defaultRoles=[ \${EXISTING},\"console_users\" ]" || fail "Failed to update default roles"

    # we'd like to use a default group instead of a default role, but there
    # seems to be a bug that causes an exception when the defaultGroups field
    # (but not the defaultRoles field) is set.

##    log "Adding verrazzano-users to default groups"
##    EXISTING=\$(kcadm.sh get realms/$_VZ_REALM --fields defaultGroups --format csv) || fail "Failed to get existing default groups"
##    kcadm.sh update realms/$_VZ_REALM -s "defaultGroups=[ \${EXISTING},\"verrazzano-users\" ]" || fail "Failed to update default groups"

    log "Creating verrazzano user"
    kcadm.sh create users -r $_VZ_REALM -s username=verrazzano -s groups[0]=verrazzano-admins -s enabled=true || fail "Failed to create user"

    log "Granting realm admin roles to verrazzano user"
    kcadm.sh add-roles -r $_VZ_REALM --uusername verrazzano --cclientid realm-management --rolename realm-admin || fail "Failed to grant roles"

    log "Setting verrazzano user password"
    kcadm.sh set-password -r $_VZ_REALM --username verrazzano --new-password $PW || fail "Failed to set user password"

    log "Creating ${VERRAZZANO_INTERNAL_PROM_USER} user"
    kcadm.sh create users -r $_VZ_REALM -s username=${VERRAZZANO_INTERNAL_PROM_USER} -s enabled=true || fail "Failed to create user"

    log "Setting ${VERRAZZANO_INTERNAL_PROM_USER} user password"
    kcadm.sh set-password -r $_VZ_REALM --username ${VERRAZZANO_INTERNAL_PROM_USER} --new-password ${VPROM} || fail "Failed to set user password"

    log "Creating ${VERRAZZANO_INTERNAL_ES_USER} user"
    kcadm.sh create users -r $_VZ_REALM -s username=${VERRAZZANO_INTERNAL_ES_USER} -s enabled=true || fail "Failed to create user"

    log "Setting ${VERRAZZANO_INTERNAL_ES_USER} user password"
    kcadm.sh set-password -r $_VZ_REALM --username ${VERRAZZANO_INTERNAL_ES_USER} --new-password ${VES} || fail "Failed to set user password"

##    log "(Re-)Creating admin-cli client"
##    ADMIN_CLI=\$(kcadm.sh get clients -r $_VZ_REALM -q "clientId=admin-cli" --fields id --format csv --noquotes 2>&1) || fail "Failed to create group"
##    if [ -n "\$ADMIN_CLI" ] ; then
##        log "Deleting admin-cli client (id: clients/\${ADMIN_CLI})"
##set -x
##        kcadm.sh delete clients/\$ADMIN_CLI -r $_VZ_REALM || fail "Failed to delete client"
##set +x
##    else
##        log "Existing admin-cli client not found"
##    fi
##    log "Creating admin-cli client"
##    kcadm.sh create clients -r $_VZ_REALM -f - <<\END
##  {
##      "clientId" : "admin-cli",
##      "name" : "${client_admin-cli}",
##      "surrogateAuthRequired" : false,
##      "enabled" : true,
##      "clientAuthenticatorType" : "client-secret",
##      "secret" : "**********",
##      "redirectUris" : [ ],
##      "webOrigins" : [ ],
##      "notBefore" : 0,
##      "bearerOnly" : false,
##      "consentRequired" : false,
##      "standardFlowEnabled" : false,
##      "implicitFlowEnabled" : false,
##      "directAccessGrantsEnabled" : true,
##      "serviceAccountsEnabled" : false,
##      "publicClient" : true,
##      "frontchannelLogout" : false,
##      "protocol" : "openid-connect",
##      "attributes" : { },
##      "authenticationFlowBindingOverrides" : { },
##      "fullScopeAllowed" : false,
##      "nodeReRegistrationTimeout" : 0,
##      "protocolMappers": [
##          {
##            "name": "groupmember",
##            "protocol": "openid-connect",
##            "protocolMapper": "oidc-group-membership-mapper",
##            "consentRequired": false,
##            "config": {
##              "full.path": "false",
##              "id.token.claim": "true",
##              "access.token.claim": "true",
##              "claim.name": "groups",
##              "userinfo.token.claim": "true"
##            }
##          }
##        ],
##        "defaultClientScopes": [
##          "web-origins",
##          "role_list",
##          "roles",
##          "profile",
##          "email"
##        ],
##        "optionalClientScopes": [
##          "address",
##          "phone",
##          "offline_access",
##          "microprofile-jwt"
##        ]
##  }
##END
##    [ \$? -eq 0 ] || fail "Failed to create client"

    log "Creating webui client"
    kcadm.sh create clients -r $_VZ_REALM -f - <<\END
{
      "clientId": "webui",
      "enabled": true,
      "surrogateAuthRequired": false,
      "alwaysDisplayInConsole": false,
      "clientAuthenticatorType": "client-secret",
      "redirectUris": [
        "https://verrazzano.ENV_NAME.DNS_SUFFIX/*",
        "https://verrazzano.ENV_NAME.DNS_SUFFIX/verrazzano/authcallback",
        "https://elasticsearch.vmi.system.ENV_NAME.DNS_SUFFIX/*",
        "https://elasticsearch.vmi.system.ENV_NAME.DNS_SUFFIX/_authentication_callback",
        "https://prometheus.vmi.system.ENV_NAME.DNS_SUFFIX/*",
        "https://prometheus.vmi.system.ENV_NAME.DNS_SUFFIX/_authentication_callback",
        "https://grafana.vmi.system.ENV_NAME.DNS_SUFFIX/*",
        "https://grafana.vmi.system.ENV_NAME.DNS_SUFFIX/_authentication_callback",
        "https://kibana.vmi.system.ENV_NAME.DNS_SUFFIX/*",
        "https://kibana.vmi.system.ENV_NAME.DNS_SUFFIX/_authentication_callback"
      ],
      "webOrigins": [
        "https://verrazzano.ENV_NAME.DNS_SUFFIX",
        "https://elasticsearch.vmi.system.ENV_NAME.DNS_SUFFIX",
        "https://prometheus.vmi.system.ENV_NAME.DNS_SUFFIX",
        "https://grafana.vmi.system.ENV_NAME.DNS_SUFFIX",
        "https://kibana.vmi.system.ENV_NAME.DNS_SUFFIX"
      ],
      "notBefore": 0,
      "bearerOnly": false,
      "consentRequired": false,
      "standardFlowEnabled": true,
      "implicitFlowEnabled": false,
      "directAccessGrantsEnabled": false,
      "serviceAccountsEnabled": false,
      "publicClient": true,
      "frontchannelLogout": false,
      "protocol": "openid-connect",
      "attributes": {
        "saml.assertion.signature": "false",
        "saml.multivalued.roles": "false",
        "saml.force.post.binding": "false",
        "saml.encrypt": "false",
        "saml.server.signature": "false",
        "saml.server.signature.keyinfo.ext": "false",
        "exclude.session.state.from.auth.response": "false",
        "saml_force_name_id_format": "false",
        "saml.client.signature": "false",
        "tls.client.certificate.bound.access.tokens": "false",
        "saml.authnstatement": "false",
        "display.on.consent.screen": "false",
        "pkce.code.challenge.method": "S256",
        "saml.onetimeuse.condition": "false"
      },
      "authenticationFlowBindingOverrides": {},
      "fullScopeAllowed": true,
      "nodeReRegistrationTimeout": -1,
      "protocolMappers": [
          {
            "name": "groupmember",
            "protocol": "openid-connect",
            "protocolMapper": "oidc-group-membership-mapper",
            "consentRequired": false,
            "config": {
              "full.path": "false",
              "id.token.claim": "true",
              "access.token.claim": "true",
              "claim.name": "groups",
              "userinfo.token.claim": "true"
            }
          },
          {
            "name": "realm roles",
            "protocol": "openid-connect",
            "protocolMapper": "oidc-usermodel-realm-role-mapper",
            "consentRequired": false,
            "config": {
              "multivalued": "true",
              "user.attribute": "foo",
              "id.token.claim": "true",
              "access.token.claim": "true",
              "claim.name": "realm_access.roles",
              "jsonType.label": "String"
            }
          }
        ],
      "defaultClientScopes": [
        "web-origins",
        "role_list",
        "roles",
        "profile",
        "email"
      ],
      "optionalClientScopes": [
        "address",
        "phone",
        "offline_access",
        "microprofile-jwt"
      ]
}
END
    [ \$? -eq 0 ] || fail "Failed to create client"

    log "Creating verrazzano-oath-client client"
    kcadm.sh create clients -r $_VZ_REALM -f - <<\END
{
      "clientId" : "verrazzano-oauth-client",
      "enabled" : true,
      "rootUrl" : "",
      "adminUrl" : "",
      "surrogateAuthRequired" : false,
      "directAccessGrantsEnabled" : "true",
      "clientAuthenticatorType" : "client-secret",
      "secret" : "de05ccdc-67df-47f3-81f6-37e61d195aba",
      "redirectUris" : [ ],
      "webOrigins" : [ ],
      "notBefore" : 0,
      "bearerOnly" : false,
      "consentRequired" : false,
      "standardFlowEnabled" : true,
      "implicitFlowEnabled" : false,
      "directAccessGrantsEnabled" : true,
      "serviceAccountsEnabled" : true,
      "publicClient" : true,
      "frontchannelLogout" : false,
      "protocol" : "openid-connect",
      "attributes" : { },
      "authenticationFlowBindingOverrides" : { },
      "fullScopeAllowed" : false,
      "nodeReRegistrationTimeout" : -1,
      "protocolMappers" : [ {
        "name" : "Client ID",
        "protocol" : "openid-connect",
        "protocolMapper" : "oidc-usersessionmodel-note-mapper",
        "consentRequired" : false,
        "config" : {
          "user.session.note" : "clientId",
          "userinfo.token.claim" : "true",
          "id.token.claim" : "true",
          "access.token.claim" : "true",
          "claim.name" : "clientId",
          "jsonType.label" : "String"
        }
      }, {
        "name" : "Client IP Address",
        "protocol" : "openid-connect",
        "protocolMapper" : "oidc-usersessionmodel-note-mapper",
        "consentRequired" : false,
        "config" : {
          "user.session.note" : "clientAddress",
          "userinfo.token.claim" : "true",
          "id.token.claim" : "true",
          "access.token.claim" : "true",
          "claim.name" : "clientAddress",
          "jsonType.label" : "String"
        }
      }, {
        "name" : "groups",
        "protocol" : "openid-connect",
        "protocolMapper" : "oidc-group-membership-mapper",
        "consentRequired" : false,
        "config" : {
          "multivalued" : "true",
          "userinfo.token.claim" : "false",
          "id.token.claim" : "true",
          "access.token.claim" : "true",
          "claim.name" : "groups",
          "jsonType.label" : "String"
        }
      }, {
        "name": "realm roles",
        "protocol": "openid-connect",
        "protocolMapper": "oidc-usermodel-realm-role-mapper",
        "consentRequired": false,
        "config": {
          "multivalued": "true",
          "user.attribute": "foo",
          "id.token.claim": "true",
          "access.token.claim": "true",
          "claim.name": "realm_access.roles",
          "jsonType.label": "String"
        }
      }, {
        "name" : "Client Host",
        "protocol" : "openid-connect",
        "protocolMapper" : "oidc-usersessionmodel-note-mapper",
        "consentRequired" : false,
        "config" : {
          "user.session.note" : "clientHost",
          "userinfo.token.claim" : "true",
          "id.token.claim" : "true",
          "access.token.claim" : "true",
          "claim.name" : "clientHost",
          "jsonType.label" : "String"
        }
      } ],
      "defaultClientScopes" : [ "web-origins", "role_list", "roles", "profile", "email" ],
      "optionalClientScopes" : [ "address", "phone", "offline_access", "microprofile-jwt" ]
}
END
    [ \$? -eq 0 ] || fail "Failed to create client"

    # default password policy
    POLICY="length(8) and notUsername"

    log "Setting password policy for master"
    kcadm.sh update realms/master -s "passwordPolicy=\${POLICY}" || fail "Failed to set password policy for master"

    log "Setting password policy for $_VZ_REALM"
    kcadm.sh update realms/$_VZ_REALM -s "passwordPolicy=\${POLICY}" || fail "Failed to set password policy for $_VZ_REALM"

    log "Configuring login theme for master"
    kcadm.sh update realms/master -s loginTheme=oracle || fail "Failed to configure login theme"

    log "Configuring login theme for $_VZ_REALM"
    kcadm.sh update realms/$_VZ_REALM -s loginTheme=oracle || fail "Failed to configure login theme"

    log "Enabling $_VZ_REALM realm"
    kcadm.sh update realms/$_VZ_REALM -s enabled=true || fail "Failed to enable $_VZ_REALM realm"

    log "Removing login config file"
    rm \$HOME/.keycloak/kcadm.config || fail "Failed to remove login config file"

EOF

}

# configure the prometheus deployment to limit istio proxy based communication to the keycloak service only.  Other
# outbound requests (scrapings) are done by prometheus using the mounted istio certs.
function patch_prometheus {
  # get the keycloak service IP
  keycloak_service_ip=$(kubectl get service/keycloak-http -n keycloak -o jsonpath='{.spec.clusterIP}')
  log "Setting ${keycloak_service_ip} as keycloak http pod IP for prometheus deployment"
  # patch the prometheus deployment
  if ! kubectl patch deployment vmi-system-prometheus-0 -n verrazzano-system --type='json' -p='[{"op": "add", "path": "/spec/template/metadata/annotations/traffic.sidecar.istio.io~1includeOutboundIPRanges", "value":'\"${keycloak_service_ip}'/32"}]'; then
    fail "Failed to patch the prometheus deployment"
  fi
}

DNS_TARGET_NAME=verrazzano-ingress.${ENV_NAME}.${DNS_SUFFIX}
REGISTRY_SECRET_EXISTS=$(check_registry_secret_exists)

if [ $(is_keycloak_enabled) == "true" ]; then
  action "Installing MySQL" install_mysql
    if [ "$?" -ne 0 ]; then
      "$SCRIPT_DIR"/k8s-dump-objects.sh -o "pods" -n "${KEYCLOAK_NS}" -m "install_mysql"
      "$SCRIPT_DIR"/k8s-dump-objects.sh -o "jobs" -n "${KEYCLOAK_NS}" -m "install_mysql"
      "$SCRIPT_DIR"/k8s-dump-objects.sh -o "nodes" -n "default" -m "install_mysql"
      log "For additional detailed information on the cluster at the time of this error, please check the diagnostics log file"
      fail "Installation of MySQL failed"
    fi

  action "Installing Keycloak" install_keycloak || exit 1

  action "patching the prometheus deployment to enable communication with keycloak" patch_prometheus || exit 1

else
  log "Skip Keycloak installation, disabled"
fi

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
if [ $(is_rancher_enabled) == "true" ]; then
  consoleout "Rancher - https://rancher.${ENV_NAME}.${DNS_SUFFIX}"
  consoleout "User: admin"
  consoleout "Password: kubectl get secret --namespace cattle-system rancher-admin-secret -o jsonpath={.data.password} | base64 --decode; echo"
  consoleout
fi
if [ $(is_keycloak_enabled) == "true" ]; then
  consoleout "Keycloak - https://keycloak.${ENV_NAME}.${DNS_SUFFIX}"
  consoleout "User: keycloakadmin"
  consoleout "Password: kubectl get secret --namespace keycloak ${KCADMIN_SECRET} -o jsonpath={.data.password} | base64 --decode; echo"
fi
if [ $(get_application_ingress_ip) == "null" ]; then
  consoleout
  consoleout "WARNING: istio-ingressgateway service does not have a valid external IP assigned yet. Public access to deployed applications will not work."
  consoleout "Use the following command to check if an External IP has been assigned to the gateway."
  consoleout "kubectl get svc istio-ingressgateway -n istio-system"
fi
