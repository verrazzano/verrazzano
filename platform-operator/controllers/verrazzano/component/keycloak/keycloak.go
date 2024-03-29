// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzpassword "github.com/verrazzano/verrazzano/pkg/security/password"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	dnsTarget               = "dnsTarget"
	rulesHost               = "rulesHost"
	tlsHosts                = "tlsHosts"
	tlsSecret               = "tlsSecret"
	keycloakCertificateName = "keycloak-tls"
	vzUsersGroup            = "verrazzano-users"
	vzAdminGroup            = "verrazzano-admins"
	vzMonitorGroup          = "verrazzano-monitors"
	vzSystemGroup           = "verrazzano-system-users"
	vzAPIAccessRole         = "vz_api_access"
	vzLogPusherRole         = "vz_log_pusher"
	vzOpenSearchAdminRole   = "vz_opensearch_admin"
	vzUserName              = "verrazzano"
	vzInternalPromUser      = "verrazzano-prom-internal"
	vzInternalEsUser        = "verrazzano-es-internal"
	keycloakPodName         = "keycloak-0"
	realmManagement         = "realm-management"
	viewUsersRole           = "view-users"
	noRouterAddr            = "mysql-instances"
	routerAddr              = "mysql"
	dbHostKey               = "database.hostname"
	headlessService         = "keycloak-headless"
	kcAdminScript           = "/opt/keycloak/bin/kcadm.sh"
	keycloakSecretName      = "keycloak-http" //nolint:gosec //#gosec G101
	keycloakIngressName     = "keycloak"
)

// Define the Keycloak Key:Value pair for init container.
// We need to replace image using the real image in the bom
const kcIngressClassKey = "ingress.ingressClassName"
const kcInitContainerKey = "extraInitContainers"
const kcInitContainerValueTemplate = `
    - name: theme-provider
      image: {{.Image}}
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
          mountPath: /cacerts
      securityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop:
            - ALL
        privileged: false
        runAsGroup: 0
        runAsNonRoot: true
        runAsUser: 1000
`

const pkceTmpl = `
{
      "clientId" : "verrazzano-pkce",
      "enabled": true,
      "surrogateAuthRequired": false,
      "alwaysDisplayInConsole": false,
      "clientAuthenticatorType": "client-secret",
      ` + pkceClientUrisTemplate + `,
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
`
const ManagedClusterClientTmpl = `
{
      "clientId" : "{{.ClientID}}",
      "enabled": true,
      "surrogateAuthRequired": false,
      "alwaysDisplayInConsole": false,
      "clientAuthenticatorType": "client-secret",
      ` + ManagedClusterClientUrisTemplate + `,
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
`

const pgClient = `
{
      "clientId" : "verrazzano-pg",
      "enabled" : true,
      "rootUrl" : "",
      "adminUrl" : "",
      "surrogateAuthRequired" : false,
      "directAccessGrantsEnabled" : "true",
      "clientAuthenticatorType" : "client-secret",
      "redirectUris" : [ ],
      "webOrigins" : [ "+" ],
      "notBefore" : 0,
      "bearerOnly" : false,
      "consentRequired" : false,
      "standardFlowEnabled" : false,
      "implicitFlowEnabled" : false,
      "directAccessGrantsEnabled" : true,
      "serviceAccountsEnabled" : false,
      "publicClient" : true,
      "frontchannelLogout" : false,
      "protocol" : "openid-connect",
      "attributes" : { },
      "authenticationFlowBindingOverrides" : { },
      "fullScopeAllowed" : true,
      "nodeReRegistrationTimeout" : -1,
      "protocolMappers" : [ {
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
`
const rancherClientTmpl = `
{
      "clientId": "rancher",
      "name": "rancher",
      "surrogateAuthRequired": false,
      "enabled": true,
      "alwaysDisplayInConsole": false,
      "clientAuthenticatorType": "client-secret",
      ` + rancherClientUrisTemplate + `,
      "notBefore": 0,
      "bearerOnly": false,
      "consentRequired": false,
      "standardFlowEnabled": true,
      "implicitFlowEnabled": false,
      "directAccessGrantsEnabled": true,
      "serviceAccountsEnabled": false,
      "publicClient": false,
      "frontchannelLogout": false,
      "protocol": "openid-connect",
      "attributes": {
        "id.token.as.detached.signature": "false",
        "saml.assertion.signature": "false",
        "saml.force.post.binding": "false",
        "saml.multivalued.roles": "false",
        "saml.encrypt": "false",
        "oauth2.device.authorization.grant.enabled": "false",
        "backchannel.logout.revoke.offline.tokens": "false",
        "saml.server.signature": "false",
        "saml.server.signature.keyinfo.ext": "false",
        "use.refresh.tokens": "true",
        "exclude.session.state.from.auth.response": "false",
        "oidc.ciba.grant.enabled": "false",
        "saml.artifact.binding": "false",
        "backchannel.logout.session.required": "true",
        "client_credentials.use_refresh_token": "false",
        "saml_force_name_id_format": "false",
        "require.pushed.authorization.requests": "false",
        "saml.client.signature": "false",
        "tls.client.certificate.bound.access.tokens": "false",
        "saml.authnstatement": "false",
        "display.on.consent.screen": "false",
        "saml.onetimeuse.condition": "false"
      },
      "authenticationFlowBindingOverrides": {},
      "fullScopeAllowed": true,
      "nodeReRegistrationTimeout": -1,
      "protocolMappers": [
        {
          "name": "Client Audience",
          "protocol": "openid-connect",
          "protocolMapper": "oidc-audience-mapper",
          "consentRequired": false,
          "config": {
            "included.client.audience": "rancher",
            "id.token.claim": "false",
            "access.token.claim": "true"
          }
        },
        {
          "name": "Groups Mapper",
          "protocol": "openid-connect",
          "protocolMapper": "oidc-group-membership-mapper",
          "consentRequired": false,
          "config": {
            "full.path": "true",
            "id.token.claim": "false",
            "access.token.claim": "false",
            "claim.name": "groups",
            "userinfo.token.claim": "true"
          }
        },
        {
          "name": "Group Path",
          "protocol": "openid-connect",
          "protocolMapper": "oidc-group-membership-mapper",
          "consentRequired": false,
          "config": {
            "full.path": "true",
            "id.token.claim": "false",
            "access.token.claim": "false",
            "claim.name": "full_group_path",
            "userinfo.token.claim": "true"
          }
        },
		{
		  "name": "full name",
	      "protocol": "openid-connect",
		  "protocolMapper": "oidc-full-name-mapper",
		  "consentRequired": false,
		  "config": {
			  "id.token.claim": "true",
			  "access.token.claim": "true",
			  "userinfo.token.claim": "true"
		  }
		}
      ],
      "defaultClientScopes": [
        "web-origins",
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
`

const argocdClientTmpl = `
{
      "clientId": "argocd",
      "name": "argocd",
      "surrogateAuthRequired": false,
      "enabled": true,
      "alwaysDisplayInConsole": false,
      "clientAuthenticatorType": "client-secret",
      ` + argocdClientUrisTemplate + `,
      "notBefore": 0,
      "bearerOnly": false,
      "consentRequired": false,
      "standardFlowEnabled": true,
      "implicitFlowEnabled": false,
      "directAccessGrantsEnabled": true,
      "serviceAccountsEnabled": false,
      "publicClient": false,
      "frontchannelLogout": false,
      "protocol": "openid-connect",
      "attributes": {
        "id.token.as.detached.signature": "false",
        "saml.assertion.signature": "false",
        "saml.force.post.binding": "false",
        "saml.multivalued.roles": "false",
        "saml.encrypt": "false",
        "oauth2.device.authorization.grant.enabled": "false",
        "backchannel.logout.revoke.offline.tokens": "false",
        "saml.server.signature": "false",
        "saml.server.signature.keyinfo.ext": "false",
        "use.refresh.tokens": "true",
        "exclude.session.state.from.auth.response": "false",
        "oidc.ciba.grant.enabled": "false",
        "saml.artifact.binding": "false",
        "backchannel.logout.session.required": "true",
        "client_credentials.use_refresh_token": "false",
        "saml_force_name_id_format": "false",
        "require.pushed.authorization.requests": "false",
        "saml.client.signature": "false",
        "tls.client.certificate.bound.access.tokens": "false",
        "saml.authnstatement": "false",
        "display.on.consent.screen": "false",
        "saml.onetimeuse.condition": "false"
      },
      "authenticationFlowBindingOverrides": {},
      "fullScopeAllowed": true,
      "nodeReRegistrationTimeout": -1,
      "protocolMappers": [
        {
          "name": "groups",
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
		}
      ],
      "defaultClientScopes": [
        "web-origins",
        "roles",
        "profile",
        "groups",
        "email"
      ],
      "optionalClientScopes": [
        "address",
        "phone",
        "offline_access",
        "microprofile-jwt"
      ]
}
`

const pkceClientUrisTemplate = `
	"redirectUris": [
	  "https://verrazzano.{{.DNSSubDomain}}/*",
	  "https://verrazzano.{{.DNSSubDomain}}/_authentication_callback",
	  "https://opensearch.vmi.system.{{.DNSSubDomain}}/*",
	  "https://opensearch.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
	  "https://prometheus.vmi.system.{{.DNSSubDomain}}/*",
	  "https://prometheus.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
	  "https://grafana.vmi.system.{{.DNSSubDomain}}/*",
	  "https://grafana.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
	  "https://osd.vmi.system.{{.DNSSubDomain}}/*",
	  "https://osd.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
	  "https://kiali.vmi.system.{{.DNSSubDomain}}/*",
	  "https://kiali.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
	  "https://thanos-query-store.{{.DNSSubDomain}}/*",
	  "https://thanos-query-store.{{.DNSSubDomain}}/_authentication_callback",
	  "https://opensearch.logging.{{.DNSSubDomain}}/_authentication_callback",
	  "https://opensearch.logging.{{.DNSSubDomain}}/*",
	  "https://osd.logging.{{.DNSSubDomain}}/*",
	  "https://osd.logging.{{.DNSSubDomain}}/_authentication_callback",
	  "https://thanos-query.{{.DNSSubDomain}}/*",
	  "https://thanos-query.{{.DNSSubDomain}}/_authentication_callback",
	  "https://thanos-ruler.{{.DNSSubDomain}}/*",
	  "https://thanos-ruler.{{.DNSSubDomain}}/_authentication_callback",
	  "https://jaeger.{{.DNSSubDomain}}/*",
	  "https://alertmanager.{{.DNSSubDomain}}/*",
	  "https://alertmanager.{{.DNSSubDomain}}/_authentication_callback"{{ if .OSHostExists}},
      "https://elasticsearch.vmi.system.{{.DNSSubDomain}}/*",
      "https://elasticsearch.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
      "https://kibana.vmi.system.{{.DNSSubDomain}}/*",
      "https://kibana.vmi.system.{{.DNSSubDomain}}/_authentication_callback"{{end}}
	],
	"webOrigins": [
	  "https://verrazzano.{{.DNSSubDomain}}",
	  "https://opensearch.vmi.system.{{.DNSSubDomain}}",
	  "https://prometheus.vmi.system.{{.DNSSubDomain}}",
	  "https://grafana.vmi.system.{{.DNSSubDomain}}",
	  "https://osd.vmi.system.{{.DNSSubDomain}}",
	  "https://kiali.vmi.system.{{.DNSSubDomain}}",
	  "https://thanos-query-store.{{.DNSSubDomain}}",
	  "https://osd.logging.{{.DNSSubDomain}}",
	  "https://opensearch.logging.{{.DNSSubDomain}}",
	  "https://thanos-query.{{.DNSSubDomain}}",
	  "https://thanos-ruler.{{.DNSSubDomain}}",
	  "https://jaeger.{{.DNSSubDomain}}",
	  "https://alertmanager.{{.DNSSubDomain}}"{{ if .OSHostExists}},
      "https://elasticsearch.vmi.system.{{.DNSSubDomain}}",
      "https://kibana.vmi.system.{{.DNSSubDomain}}"
 {{end}} 
	]
`
const ManagedClusterClientUrisTemplate = `
	"redirectUris": [
	  "https://prometheus.vmi.system.{{.DNSSubDomain}}/*",
	  "https://prometheus.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
	  "https://thanos-query-store.{{.DNSSubDomain}}/*",
	  "https://thanos-query-store.{{.DNSSubDomain}}/_authentication_callback",
	  "https://thanos-query.{{.DNSSubDomain}}/*",
	  "https://thanos-query.{{.DNSSubDomain}}/_authentication_callback"
	],
	"webOrigins": [
	  "https://prometheus.vmi.system.{{.DNSSubDomain}}",
	  "https://thanos-query-store.{{.DNSSubDomain}}",
	  "https://thanos-query.{{.DNSSubDomain}}"
	]
`

const rancherClientUrisTemplate = `
	"redirectUris": [
        "https://rancher.{{.DNSSubDomain}}/verify-auth"
    ],
	"webOrigins": [
		"https://rancher.{{.DNSSubDomain}}"
	]
`

const argocdClientUrisTemplate = `
    "rootUrl": "https://argocd.{{.DNSSubDomain}}",
	"redirectUris": [
        "https://argocd.{{.DNSSubDomain}}/auth/callback"
    ],
    "baseUrl": "/applications",
    "adminUrl": "https://argocd.{{.DNSSubDomain}}",
	"webOrigins": [
		"https://argocd.{{.DNSSubDomain}}"
	]
`

// KeycloakClients represents an array of clients currently configured in Keycloak
type KeycloakClients []struct {
	ID       string `json:"id"`
	ClientID string `json:"clientId"`
}

// KeycloakClientScopes represents an array of client-scopes currently configured in Keycloak
type KeycloakClientScopes []struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SubGroup represents the subgroups that Keycloak groups may contain
type SubGroup struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Path      string        `json:"path"`
	SubGroups []interface{} `json:"subGroups"`
}

// KeycloakGroups is an array of groups configured in Keycloak
type KeycloakGroups []struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	SubGroups []SubGroup
}

// KeycloakRoles is an array of roles configured in Keycloak
type KeycloakRoles []struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Composite   bool   `json:"composite"`
	ClientRole  bool   `json:"clientRole"`
	ContainerID string `json:"containerId"`
}

// KeycloakUser is an user configured in Keycloak
type KeycloakUser struct {
	ID                         string        `json:"id"`
	CreatedTimestamp           int64         `json:"createdTimestamp"`
	Username                   string        `json:"username"`
	Enabled                    bool          `json:"enabled"`
	Totp                       bool          `json:"totp"`
	EmailVerified              bool          `json:"emailVerified"`
	DisableableCredentialTypes []interface{} `json:"disableableCredentialTypes"`
	RequiredActions            []interface{} `json:"requiredActions"`
	NotBefore                  int           `json:"notBefore"`
	Access                     struct {
		ManageGroupMembership bool `json:"manageGroupMembership"`
		View                  bool `json:"view"`
		MapRoles              bool `json:"mapRoles"`
		Impersonate           bool `json:"impersonate"`
		Manage                bool `json:"manage"`
	} `json:"access"`
}

// KeycloakClientSecret represents a client-secret of a client currently configured in Keycloak
type KeycloakClientSecret struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type templateData struct {
	DNSSubDomain string
	OSHostExists bool
	ClientID     string
}

// imageData needed for template rendering
type imageData struct {
	Image string
}

// maskPw will mask passwords in strings with '******'
var maskPw = vzpassword.MaskFunction("password ")

// AppendKeycloakOverrides appends the Keycloak theme for the Key keycloak.extraInitContainers.
// A go template is used to replace the image in the init container spec.
func AppendKeycloakOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Create a Bom and get the Key Value overrides
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}

	// Get Keycloak theme images
	images, err := bomFile.BuildImageOverrides("keycloak-oracle-theme")
	if err != nil {
		return nil, err
	}
	if len(images) != 1 {
		return nil, fmt.Errorf("Component Keycloak failed, expected 1 image for Keycloak theme, found %v", len(images))
	}

	// use template to get populate template with image:tag
	var b bytes.Buffer
	t, err := template.New("image").Parse(kcInitContainerValueTemplate)
	if err != nil {
		return nil, err
	}

	// Render the template
	data := imageData{Image: images[0].Value}
	err = t.Execute(&b, data)
	if err != nil {
		return nil, err
	}

	kvs = append(kvs, bom.KeyValue{
		Key:   kcInitContainerKey,
		Value: b.String(),
	})

	// Get DNS Domain Configuration
	dnsSubDomain, err := getDNSDomain(compContext.Client(), compContext.EffectiveCR())
	if err != nil {
		compContext.Log().Errorf("Component Keycloak failed retrieving DNS sub domain: %v", err)
		return nil, err
	}
	compContext.Log().Debugf("AppendKeycloakOverrides: DNSDomain returned %s", dnsSubDomain)

	host := "keycloak." + dnsSubDomain

	kvs = append(kvs, bom.KeyValue{
		Key:       dnsTarget,
		Value:     host,
		SetString: true,
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   rulesHost,
		Value: host,
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   tlsHosts,
		Value: host,
	})

	// this secret contains the Keycloak TLS certificate created by cert-manager during the original Keycloak installation
	kvs = append(kvs, bom.KeyValue{
		Key:   tlsSecret,
		Value: keycloakCertificateName,
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   kcIngressClassKey,
		Value: vzconfig.GetIngressClassName(compContext.EffectiveCR()),
	})

	// set the appropriate host address for DB based on the availability of the MySQL router
	mysqlAddr := noRouterAddr
	if isMySQLRouterDeployed(compContext, err) {
		mysqlAddr = routerAddr
	}
	kvs = append(kvs, bom.KeyValue{
		Key:   dbHostKey,
		Value: mysqlAddr,
	})

	return kvs, nil
}

func isMySQLRouterDeployed(compContext spi.ComponentContext, err error) bool {
	deployment := appv1.Deployment{}
	err = compContext.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: "mysql-router"}, &deployment)
	return err == nil
}

// getEnvironmentName returns the name of the Verrazzano install environment
func getEnvironmentName(envName string) string {
	if envName == "" {
		return constants.DefaultEnvironmentName
	}

	return envName
}

// updateKeycloakIngress updates the Ingress
func updateKeycloakIngress(ctx spi.ComponentContext) error {
	ingress := networkv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: keycloakIngressName, Namespace: constants.KeycloakNamespace},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &ingress, func() error {
		dnsSuffix, _ := vzconfig.GetDNSSuffix(ctx.Client(), ctx.EffectiveCR())
		ingress.Annotations["cert-manager.io/common-name"] = fmt.Sprintf("%s.%s.%s",
			ComponentName, ctx.EffectiveCR().Spec.EnvironmentName, dnsSuffix)
		ingress.Annotations["cert-manager.io/cluster-issuer"] = vzconst.VerrazzanoClusterIssuerName
		// update target annotation on Keycloak Ingress for external DNS
		if vzcr.IsExternalDNSEnabled(ctx.EffectiveCR()) {
			dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
			if err != nil {
				return err
			}
			ingressTarget := fmt.Sprintf("verrazzano-ingress.%s", dnsSubDomain)
			ctx.Log().Debugf("updateKeycloakIngress: Updating Keycloak Ingress with ingressTarget = %s", ingressTarget)
			ingress.Annotations["external-dns.alpha.kubernetes.io/target"] = ingressTarget
		}
		return nil
	})
	ctx.Log().Debugf("updateKeycloakIngress: Keycloak ingress operation result: %v", err)
	return err
}

// updateKeycloakUris invokes kcadm.sh in Keycloak pod to update the client with Keycloak rewrite and weborigin uris
func updateKeycloakUris(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, kcPod *corev1.Pod, clientID string, uriTemplate string, dnsSubdomain *string) error {
	data, err := populateClientTemplate(ctx, "{"+uriTemplate+"}", "", dnsSubdomain)
	if err != nil {
		return err
	}

	// Update client
	updateClientCmd := kcAdminScript + " update clients/" + clientID + " -r " + vzconst.VerrazzanoOIDCSystemRealm + " -b '" +
		strings.TrimSpace(data) +
		"'"
	ctx.Log().Debugf("updateKeycloakUris: Update client with Id = %s, Cmd = %s", clientID, updateClientCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(updateClientCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed updating client with Id = %s stdout = %s, stderr = %s", clientID, stdout, stderr)
		return err
	}

	ctx.Log().Debug("Component Keycloak successfully updated Keycloak URIs")
	return nil
}

// configureKeycloakRealms configures the Verrazzano system realm
func configureKeycloakRealms(ctx spi.ComponentContext) error {
	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return err
	}

	// Login to Keycloak
	err = LoginKeycloak(ctx, cfg, cli)
	if err != nil {
		return err
	}

	// Create VerrazzanoSystem Realm
	err = createVerrazzanoSystemRealm(ctx, cfg, cli)
	if err != nil {
		return err
	}

	// Create Verrazzano Users Group
	userGroupID, err := createVerrazzanoGroup(ctx, cfg, cli, vzUsersGroup, "")
	if err != nil {
		return err
	}
	if userGroupID == "" {
		err := errors.New("Component Keycloak failed; user Group ID from Keycloak is zero length")
		ctx.Log().Error(err)
		return err
	}

	// Create Verrazzano Admin Group
	adminGroupID, err := createVerrazzanoGroup(ctx, cfg, cli, vzAdminGroup, userGroupID)
	if err != nil {
		return err
	}
	if adminGroupID == "" {
		err := errors.New("Component Keycloak failed; admin group ID from Keycloak is zero length")
		ctx.Log().Error(err)
		return err
	}

	// Create Verrazzano Project Monitors Group
	monitorGroupID, err := createVerrazzanoGroup(ctx, cfg, cli, vzMonitorGroup, userGroupID)
	if err != nil {
		return err
	}
	if monitorGroupID == "" {
		err = errors.New("Component Keycloak failed; monitor group ID from Keycloak is zero length")
		ctx.Log().Error(err)
		return err
	}

	// Create Verrazzano System Group
	_, err = createVerrazzanoGroup(ctx, cfg, cli, vzSystemGroup, userGroupID)
	if err != nil {
		return err
	}

	// Create Verrazzano API Access Role
	err = createVerrazzanoRole(ctx, cfg, cli, vzAPIAccessRole)
	if err != nil {
		return err
	}

	// Create Verrazzano Log Pusher Role
	err = createVerrazzanoRole(ctx, cfg, cli, vzLogPusherRole)
	if err != nil {
		return err
	}

	// Create Verrazzano OpenSearch Admin role
	err = createVerrazzanoRole(ctx, cfg, cli, vzOpenSearchAdminRole)
	if err != nil {
		return err
	}

	// Granting Roles to Groups
	err = grantRolesToGroups(ctx, cfg, cli, userGroupID, adminGroupID, monitorGroupID)
	if err != nil {
		return err
	}

	// Creating Verrazzano User
	err = createUser(ctx, cfg, cli, vzUserName, "verrazzano", constants.VerrazzanoSystemNamespace, vzAdminGroup, "Verrazzano", "Admin")
	if err != nil {
		return err
	}

	// Creating Verrazzano Internal Prometheus User
	err = createUser(ctx, cfg, cli, vzInternalPromUser, "verrazzano-prom-internal", constants.VerrazzanoSystemNamespace, vzSystemGroup, "", "")
	if err != nil {
		return err
	}

	// Create Verrazzano Internal Thanos User if the corresponding secret exists. The secret is installed via the Thanos Helm chart.
	secret := &corev1.Secret{}
	err = ctx.Client().Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: constants.ThanosInternalUserSecretName}, secret)
	if client.IgnoreNotFound(err) != nil {
		return err
	}
	if err == nil {
		err = createUser(ctx, cfg, cli, constants.ThanosInternalUserSecretName, constants.ThanosInternalUserSecretName, constants.VerrazzanoMonitoringNamespace, vzSystemGroup, "", "")
		if err != nil {
			return err
		}
	}

	// Creating Verrazzano Internal ES User
	err = createUser(ctx, cfg, cli, vzInternalEsUser, "verrazzano-es-internal", constants.VerrazzanoSystemNamespace, vzSystemGroup, "", "")
	if err != nil {
		return err
	}

	// Create verrazzano-pkce client
	err = CreateOrUpdateClient(ctx, cfg, cli, "verrazzano-pkce", pkceTmpl, pkceClientUrisTemplate, false, nil)
	if err != nil {
		return err
	}

	// Creating verrazzano-pg client
	err = CreateOrUpdateClient(ctx, cfg, cli, "verrazzano-pg", pgClient, "", true, nil)
	if err != nil {
		return err
	}

	// Grant vz_opensearch_admin role to verrazzano user
	err = addRealmRoleToUser(ctx, cfg, cli, vzUserName, vzconst.VerrazzanoOIDCSystemRealm, vzOpenSearchAdminRole)
	if err != nil {
		return err
	}

	// Grant vz_log_pusher role to verrazzano-es-internal user
	err = addRealmRoleToUser(ctx, cfg, cli, vzInternalEsUser, vzconst.VerrazzanoOIDCSystemRealm, vzLogPusherRole)
	if err != nil {
		return err
	}

	if vzcr.IsRancherEnabled(ctx.EffectiveCR()) {
		// Creating rancher client
		err = CreateOrUpdateClient(ctx, cfg, cli, "rancher", rancherClientTmpl, rancherClientUrisTemplate, true, nil)
		if err != nil {
			return err
		}

		// Update Keycloak AuthConfig for Rancher with client secret
		err = updateRancherClientSecretForKeycloakAuthConfig(ctx)
		if err != nil {
			return err
		}

		// Add view-users role to verrazzano user
		err = addClientRoleToUser(ctx, cfg, cli, vzUserName, realmManagement, vzconst.VerrazzanoOIDCSystemRealm, viewUsersRole)
		if err != nil {
			return err
		}
	}

	if vzcr.IsArgoCDEnabled(ctx.EffectiveCR()) {
		// Creating groups client scope
		err = createOrUpdateClientScope(ctx, cfg, cli, "groups")
		if err != nil {
			return err
		}

		// Creating Argo CD client
		err = CreateOrUpdateClient(ctx, cfg, cli, "argocd", argocdClientTmpl, argocdClientUrisTemplate, true, nil)
		if err != nil {
			return err
		}

		// Setting the Access Token Lifespan value to 20mins.
		// Required to ensure Argo CD UI does not log out the user until the Access Token lifespan expires
		// Setting password policy for Verrazzano realm
		err = setAccessTokenLifespanForRealm(ctx, cfg, cli, "verrazzano-system")
		if err != nil {
			return err
		}
	}

	// Setting password policy for master
	err = setPasswordPolicyForRealm(ctx, cfg, cli, "master", "passwordPolicy=length(8) and notUsername")
	if err != nil {
		return err
	}

	// Setting password policy for Verrazzano realm
	err = setPasswordPolicyForRealm(ctx, cfg, cli, "verrazzano-system", "passwordPolicy=length(8) and notUsername")
	if err != nil {
		return err
	}

	// Configuring login theme for master
	err = configureLoginThemeForRealm(ctx, cfg, cli, "master", "oracle")
	if err != nil {
		return err
	}

	// Configuring login theme for verrazzano-system
	err = configureLoginThemeForRealm(ctx, cfg, cli, "verrazzano-system", "oracle")
	if err != nil {
		return err
	}

	// Enabling vzconst.VerrazzanoOIDCSystemRealm realm
	err = enableVerrazzanoSystemRealm(ctx, cfg, cli)
	if err != nil {
		return err
	}

	// Removing login config file
	err = removeLoginConfigFile(ctx, cfg, cli)
	if err != nil {
		return err
	}

	ctx.Log().Oncef("Component Keycloak successfully configured realm %s", vzconst.VerrazzanoOIDCSystemRealm)
	return nil
}

// loginKeycloak logs into Keycloak so kcadm API calls can be made
func LoginKeycloak(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface) error {
	// Make sure the Keycloak pod is ready
	kcPod := keycloakPod()
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: kcPod.Namespace, Name: kcPod.Name}, kcPod)
	if err != nil {
		ctx.Log().Progressf("Component Keycloak failed to get pod %s: %v", kcPod.Name, err)
		return err
	}
	if !isPodReady(kcPod) {
		ctx.Log().Progressf("Component Keycloak waiting for pod %s to be ready", kcPod.Name)
		return fmt.Errorf("Waiting for pod %s to be ready", kcPod.Name)
	}

	// Get the Keycloak admin password
	secret := &corev1.Secret{}
	err = ctx.Client().Get(context.TODO(), client.ObjectKey{
		Namespace: constants.KeycloakNamespace,
		Name:      keycloakSecretName,
	}, secret)
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed retrieving Keycloak password: %s", err)
		return err
	}
	pw := secret.Data["password"]
	keycloakpw := string(pw)
	if keycloakpw == "" {
		err = errors.New("Component Keycloak failed; Keycloak password is an empty string")
		ctx.Log().Error(err)
		return err
	}
	ctx.Log().Debug("LoginKeycloak: Successfully retrieved Keycloak password")

	// Login to Keycloak
	loginCmd := kcAdminScript + " config credentials --server http://localhost:8080/auth --realm master --user keycloakadmin --password " + keycloakpw
	ctx.Log().Debugf("LoginKeycloak: Login Cmd = %s", maskPw(loginCmd))
	stdOut, stdErr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(loginCmd))
	if err != nil {
		ctx.Log().Progressf("Component Keycloak failed logging into Keycloak: stdout = %s: stderr = %s, err = %v", stdOut, stdErr, maskPw(err.Error()))
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	ctx.Log().Once("Component Keycloak successfully logged into Keycloak")

	return nil
}

func bashCMD(command string) []string {
	return []string{
		"bash",
		"-c",
		command,
	}
}

func keycloakPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakPodName,
			Namespace: ComponentNamespace,
		},
	}
}

// createAuthSecret verifies the secret doesn't already exists and creates it
func createAuthSecret(ctx spi.ComponentContext, namespace string, secretname string, username string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretname, Namespace: namespace},
	}
	err := ctx.Client().Get(context.TODO(), client.ObjectKey{
		Namespace: namespace,
		Name:      secretname,
	}, secret)
	// If the secret doesn't exist, create it
	if err != nil {
		pw, err := vzpassword.GeneratePassword(15)
		if err != nil {
			return err
		}
		_, err = controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), secret, func() error {
			// Build the secret data
			secret.Data = map[string][]byte{
				"username": []byte(username),
				"password": []byte(pw),
			}
			return nil
		})
		ctx.Log().Debugf("Keycloak secret operation result: %v", err)

		if err != nil {
			return err
		}
		ctx.Log().Once("Component Keycloak successfully created the auth secret")

	}
	return nil
}

// getSecretPassword retrieves the password associated with a secret
func getSecretPassword(ctx spi.ComponentContext, namespace string, secretname string) (string, error) {
	secret := &corev1.Secret{}
	err := ctx.Client().Get(context.TODO(), client.ObjectKey{
		Namespace: namespace,
		Name:      secretname,
	}, secret)
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed retrieving secret %s/%s: %v", namespace, secretname, err)
		return "", err
	}
	pw := secret.Data["password"]
	stringpw := string(pw)
	if stringpw == "" {
		err := fmt.Errorf("Component Keycloak failed, password field empty in secret %s/%s", namespace, secretname)
		ctx.Log().Error(err)
		return "", err
	}
	return stringpw, nil
}

// getDNSDomain returns the DNS Domain
func getDNSDomain(c client.Client, vz *vzapi.Verrazzano) (string, error) {
	var dnsDomain string
	dnsSuffix, err := vzconfig.GetDNSSuffix(c, vz)
	if err != nil {
		return "", err
	}
	if vz != nil {
		dnsDomain = fmt.Sprintf("%s.%s", vz.Spec.EnvironmentName, dnsSuffix)
	} else {
		dnsDomain = dnsSuffix
	}
	return dnsDomain, nil
}

func createVerrazzanoSystemRealm(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface) error {
	kcPod := keycloakPod()
	realm := "realm=" + vzconst.VerrazzanoOIDCSystemRealm
	checkRealmExistsCmd := kcAdminScript + " get realms/" + vzconst.VerrazzanoOIDCSystemRealm
	ctx.Log().Debugf("createVerrazzanoSystemRealm: Check Verrazzano System Realm Exists Cmd = %s", checkRealmExistsCmd)
	_, _, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(checkRealmExistsCmd))
	if err != nil {
		ctx.Log().Debug("createVerrazzanoSystemRealm: Verrazzano System Realm doesn't exist: Creating it")
		createRealmCmd := kcAdminScript + " create realms -s " + realm + " -s enabled=false"
		ctx.Log().Debugf("createVerrazzanoSystemRealm: Create Verrazzano System Realm Cmd = %s", createRealmCmd)
		stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createRealmCmd))
		if err != nil {
			ctx.Log().Errorf("Component Keycloak failed creating Verrazzano System Realm: stdout = %s, stderr = %s", stdout, stderr)
			return err
		}
		ctx.Log().Once("Component Keycloak successfully created the Verrazzano system realm")
	}
	return nil
}

func createVerrazzanoGroup(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, group string, parentID string) (string, error) {
	kcPod := keycloakPod()
	keycloakGroups, err := getKeycloakGroups(ctx, cfg, cli, kcPod)
	if err == nil && groupExists(keycloakGroups, group) {
		// Group already exists
		return getGroupID(keycloakGroups, group), nil
	}
	groupsResource := "groups"
	groupName := "name=" + group
	if parentID != "" {
		groupsResource = fmt.Sprintf("groups/%s/children", parentID)
	}

	cmd := fmt.Sprintf("%s create %s -r %s -s %s", kcAdminScript, groupsResource, vzconst.VerrazzanoOIDCSystemRealm, groupName)
	ctx.Log().Debugf("createVerrazzanoGroup: Create Verrazzano %s Group Cmd = %s", group, cmd)
	out, _, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(cmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed creating Verrazzano %s Group: command output = %s", group, out)
		return "", err
	}
	ctx.Log().Debugf("createVerrazzanoGroup: Create Verrazzano %s Group Output = %s", group, out)
	if len(out) == 0 {
		err = fmt.Errorf("Component Keycloak failed; %s group ID from Keycloak is zero length", group)
		ctx.Log().Error(err)
		return "", err
	}
	arr := strings.Split(string(out), "'")
	if len(arr) != 3 {
		return "", fmt.Errorf("Component Keycloak failed parsing output returned from %s Group create stdout returned = %s", group, out)
	}
	ctx.Log().Debugf("createVerrazzanoGroup: %s Group ID = %s", group, arr[1])
	ctx.Log().Oncef("Component Keycloak successfully created the Verrazzano %s group", group)
	return arr[1], nil
}

func createVerrazzanoRole(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, roleName string) error {
	kcPod := keycloakPod()
	keycloakRoles, err := getKeycloakRoles(ctx, cfg, cli, kcPod)
	if err == nil && roleExists(keycloakRoles, roleName) {
		return nil
	}
	role := "name=" + roleName
	createRoleCmd := kcAdminScript + " create roles -r " + vzconst.VerrazzanoOIDCSystemRealm + " -s " + role
	ctx.Log().Debugf("createVerrazzanoRole: Create %s role cmd = %s", roleName, createRoleCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createRoleCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed creating %s role: stdout = %s, stderr = %s", roleName, stdout, stderr)
		return err
	}
	ctx.Log().Oncef("Component Keycloak successfully created the %s role", roleName)
	return nil
}

func grantRolesToGroups(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, userGroupID string, adminGroupID string, monitorGroupID string) error {
	// Keycloak API does not fail if Role already exists as of 15.0.3
	kcPod := keycloakPod()
	// Granting vz_api_access role to verrazzano users group
	grantAPIAccessToVzUserGroupCmd := kcAdminScript + " add-roles -r " + vzconst.VerrazzanoOIDCSystemRealm + " --gid " + userGroupID + " --rolename " + vzAPIAccessRole
	ctx.Log().Debugf("grantRolesToGroups: Grant API Access to VZ Users Cmd = %s", grantAPIAccessToVzUserGroupCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(grantAPIAccessToVzUserGroupCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed granting api access role to Verrazzano users group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Once("Component Keycloak successfully granted the access role to the Verrazzano user group")

	return nil
}

func createUser(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, userName, secretName, secretNamespace, groupName, firstName, lastName string) error {
	kcPod := keycloakPod()
	keycloakUsers, err := getKeycloakUsers(ctx, cfg, cli, kcPod)
	if err == nil && userExists(keycloakUsers, userName) {
		return nil
	}
	vzUser := "username=" + userName
	vzUserGroup := "groups[0]=/" + vzUsersGroup + "/" + groupName
	createVzUserCmd := kcAdminScript + " create users -r " + vzconst.VerrazzanoOIDCSystemRealm + " -s " + vzUser + " -s " + vzUserGroup + " -s enabled=true"
	if firstName != "" {
		createVzUserCmd = createVzUserCmd + " -s firstName=" + firstName
	}

	if lastName != "" {
		createVzUserCmd = createVzUserCmd + " -s lastName=" + lastName
	}

	ctx.Log().Debugf("createUser: Create Verrazzano User Cmd = %s", createVzUserCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createVzUserCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed creating Verrazzano user: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debugf("createUser: Successfully Created VZ User %s", userName)

	vzpw, err := getSecretPassword(ctx, secretNamespace, secretName)
	if err != nil {
		return err
	}
	setVZUserPwCmd := kcAdminScript + " set-password -r " + vzconst.VerrazzanoOIDCSystemRealm + " --username " + userName + " --new-password " + vzpw
	ctx.Log().Debugf("createUser: Set Verrazzano User PW Cmd = %s", maskPw(setVZUserPwCmd))
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setVZUserPwCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed setting Verrazzano user password: stdout = %s, stderr = %s", stdout, stderr)
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	ctx.Log().Debugf("createUser: Created VZ User %s PW", userName)
	ctx.Log().Oncef("Component Keycloak successfully created user %s", userName)

	return nil
}

func createOrUpdateClientScope(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, groupname string) error {
	keycloakClientScopes, err := getKeycloakClientScopes(ctx)
	if err != nil {
		return err
	}

	kcPod := keycloakPod()
	if clientScopeName := getClientScopeName(keycloakClientScopes, groupname); clientScopeName != "" {
		return nil
	}

	// Create client scope
	clientCreateCmd := kcAdminScript + " create -x client-scopes -r " + vzconst.VerrazzanoOIDCSystemRealm + " -s name=groups -s protocol=openid-connect"
	ctx.Log().Debugf("createOrUpdateClient: Create %s client Cmd = %s", groupname, clientCreateCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(clientCreateCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed creating %s client scope stdout = %s, stderr = %s", groupname, stdout, stderr)
		return err
	}
	ctx.Log().Debugf("createOrUpdateClientScope: Created %s client-scope", groupname)
	ctx.Log().Oncef("Component Keycloak successfully created client-scope %s", groupname)
	return nil
}

func CreateOrUpdateClient(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, clientName string, clientTemplate string, uriTemplate string, generateSecret bool, dnsSubdomain *string) error {
	keycloakClients, err := getKeycloakClients(ctx)
	if err != nil {
		return err
	}

	kcPod := keycloakPod()
	if clientID := getClientID(keycloakClients, clientName); clientID != "" {
		if uriTemplate != "" {
			err := updateKeycloakUris(ctx, cfg, cli, kcPod, clientID, uriTemplate, dnsSubdomain)
			if err != nil {
				return err
			}
		}

		return nil
	}

	data, err := populateClientTemplate(ctx, clientTemplate, clientName, dnsSubdomain)
	if err != nil {
		return err
	}

	// Create client
	clientCreateCmd := kcAdminScript + " create clients -r " + vzconst.VerrazzanoOIDCSystemRealm + " -f - <<\\END" +
		data +
		"END"

	ctx.Log().Debugf("CreateOrUpdateClient: Create %s client Cmd = %s", clientName, clientCreateCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(clientCreateCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed creating %s client: stdout = %s, stderr = %s", clientName, stdout, stderr)
		return err
	}

	if generateSecret {
		err = generateClientSecret(ctx, cfg, cli, clientName, stdout, kcPod)
		if err != nil {
			ctx.Log().Errorf("Component Keycloak failed creating %s client secret: err = %s", clientName, err.Error())
			return err
		}
	}

	ctx.Log().Debugf("CreateOrUpdateClient: Created %s client", clientName)
	ctx.Log().Oncef("Component Keycloak successfully created client %s", clientName)
	return nil
}

func setAccessTokenLifespanForRealm(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, realmName string) error {
	kcPod := keycloakPod()
	setTokenCmd := kcAdminScript + " update realms/" + realmName + " -s accessTokenLifespan=1200"
	ctx.Log().Debugf("setAccessTokenLifespanForRealm: Setting Access Token Lifespan for realm %s Cmd = %s", realmName, setTokenCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setTokenCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed setting access token lifespan for verrazzano-system: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debugf("setAccessTokenLifespanForRealm: Set Access Token Lifespan for realm %s", realmName)
	ctx.Log().Oncef("Component Keycloak successfully set the Access Token Lifespan for realm %s", realmName)
	return nil
}

func setPasswordPolicyForRealm(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, realmName string, policy string) error {
	kcPod := keycloakPod()
	setPolicyCmd := kcAdminScript + " update realms/" + realmName + " -s \"" + policy + "\""
	ctx.Log().Debugf("setPasswordPolicyForRealm: Setting password policy for realm %s Cmd = %s", realmName, setPolicyCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setPolicyCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed setting password policy for master: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debugf("setPasswordPolicyForRealm: Set password policy for realm %s", realmName)
	ctx.Log().Oncef("Component Keycloak successfully set the password policy for realm %s", realmName)
	return nil
}

func configureLoginThemeForRealm(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, realmName string, loginTheme string) error {
	kcPod := keycloakPod()
	setLoginThemeCmd := kcAdminScript + " update realms/" + realmName + " -s loginTheme=" + loginTheme
	ctx.Log().Debugf("configureLoginThemeForRealm: Configuring login theme Cmd = %s", setLoginThemeCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setLoginThemeCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed configuring login theme for master: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureLoginThemeForRealm: Configured login theme for master Cmd")
	ctx.Log().Oncef("Component Keycloak successfully set the login theme for realm %s", realmName)
	return nil
}

func enableVerrazzanoSystemRealm(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface) error {
	kcPod := keycloakPod()
	setVzEnableRealmCmd := kcAdminScript + " update realms/" + vzconst.VerrazzanoOIDCSystemRealm + " -s enabled=true"
	ctx.Log().Debugf("enableVerrazzanoSystemRealm: Enabling vzconst.VerrazzanoOIDCSystemRealm realm Cmd = %s", setVzEnableRealmCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setVzEnableRealmCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed enabling vzconst.VerrazzanoOIDCSystemRealm realm: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("enableVerrazzanoSystemRealm: Enabled vzconst.VerrazzanoOIDCSystemRealm realm")
	ctx.Log().Once("Component Keycloak successfully enabled the vzconst.VerrazzanoOIDCSystemRealm realm")

	return nil
}

func removeLoginConfigFile(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface) error {
	kcPod := keycloakPod()
	removeLoginConfigFileCmd := "rm ~/.keycloak/kcadm.config"
	ctx.Log().Debugf("removeLoginConfigFile: Removing login config file Cmd = %s", removeLoginConfigFileCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(removeLoginConfigFileCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed removing login config file: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("removeLoginConfigFile: Removed login config file")
	return nil
}

// getKeycloakGroups returns a structure of Groups in Realm verrazzano-system
func getKeycloakGroups(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, kcPod *corev1.Pod) (KeycloakGroups, error) {
	var keycloakGroups KeycloakGroups
	// Get the Client ID JSON array
	cmd := fmt.Sprintf("%s get groups -r %s", kcAdminScript, vzconst.VerrazzanoOIDCSystemRealm)
	out, _, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(cmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed retrieving Groups: %s", err)
		return nil, err
	}
	if len(out) == 0 {
		err = errors.New("Component Keycloak failed; groups JSON from Keycloak is zero length")
		ctx.Log().Error(err)
		return nil, err
	}
	err = json.Unmarshal([]byte(out), &keycloakGroups)
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed ummarshalling groups json: %v", err)
		return nil, err
	}

	return keycloakGroups, nil
}

func groupExists(keycloakGroups KeycloakGroups, groupName string) bool {
	for _, keycloakGroup := range keycloakGroups {
		if keycloakGroup.Name == groupName {
			return true
		}
		for _, subGroup := range keycloakGroup.SubGroups {
			if subGroup.Name == groupName {
				return true
			}
		}
	}
	return false
}

func getGroupID(keycloakGroups KeycloakGroups, groupName string) string {
	for _, keycloakGroup := range keycloakGroups {
		if keycloakGroup.Name == groupName {
			return keycloakGroup.ID
		}
		for _, subGroup := range keycloakGroup.SubGroups {
			if subGroup.Name == groupName {
				return subGroup.ID
			}
		}
	}
	return ""
}

// getKeycloakRoless returns a structure of Groups in Realm verrazzano-system
func getKeycloakRoles(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, kcPod *corev1.Pod) (KeycloakRoles, error) {
	var keycloakRoles KeycloakRoles
	// Get the Client ID JSON array
	out, _, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(kcAdminScript+" get-roles -r "+vzconst.VerrazzanoOIDCSystemRealm))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed retrieving Roles: %s", err)
		return nil, err
	}
	if len(out) == 0 {
		err = errors.New("Component Keycloak failed; roles JSON from Keycloak is zero length")
		ctx.Log().Error(err)
		return nil, err
	}
	err = json.Unmarshal([]byte(out), &keycloakRoles)
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed ummarshalling groups json: %v", err)
		return nil, err
	}

	return keycloakRoles, nil
}

func roleExists(keycloakRoles KeycloakRoles, roleName string) bool {
	for _, keycloakRole := range keycloakRoles {
		if keycloakRole.Name == roleName {
			return true
		}
	}
	return false
}

// getKeycloakUsers returns a structure of Users in Realm verrazzano-system
func getKeycloakUsers(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, kcPod *corev1.Pod) ([]KeycloakUser, error) {
	var keycloakUsers []KeycloakUser
	// Get the Client ID JSON array
	out, _, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(kcAdminScript+" get users -r "+vzconst.VerrazzanoOIDCSystemRealm))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed retrieving Users: %s", err)
		return nil, err
	}
	if len(out) == 0 {
		err := errors.New("Component Keycloak failed; users JSON from Keycloak is zero length")
		ctx.Log().Error(err)
		return nil, err
	}
	err = json.Unmarshal([]byte(out), &keycloakUsers)
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed ummarshalling users json: %v", err)
		return nil, err
	}
	return keycloakUsers, nil
}

func userExists(keycloakUsers []KeycloakUser, userName string) bool {
	for _, keycloakUser := range keycloakUsers {
		if keycloakUser.Username == userName {
			return true
		}
	}
	return false
}

// getKeycloakClients returns a structure of Users in Realm verrazzano-system
func getKeycloakClients(ctx spi.ComponentContext) (KeycloakClients, error) {
	var keycloakClients KeycloakClients
	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return nil, err
	}
	// Get the Client ID JSON array
	out, _, err := k8sutil.ExecPod(cli, cfg, keycloakPod(), ComponentName, bashCMD(kcAdminScript+" get clients -r "+vzconst.VerrazzanoOIDCSystemRealm+" --fields id,clientId"))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed retrieving clients: %s", err)
		return nil, err
	}
	if len(out) == 0 {
		err := errors.New("Component Keycloak failed; clients JSON from Keycloak is zero length")
		ctx.Log().Error(err)
		return nil, err
	}
	err = json.Unmarshal([]byte(out), &keycloakClients)
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed ummarshalling client json: %v", err)
		return nil, err
	}
	return keycloakClients, nil
}

// getKeycloakClientsScopes returns a structure of ClientScopes in Realm verrazzano-system
func getKeycloakClientScopes(ctx spi.ComponentContext) (KeycloakClientScopes, error) {
	var keycloakClientScopes KeycloakClientScopes
	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return nil, err
	}
	// Get the Client ID JSON array
	out, _, err := k8sutil.ExecPod(cli, cfg, keycloakPod(), ComponentName, bashCMD(kcAdminScript+" get client-scopes -r "+vzconst.VerrazzanoOIDCSystemRealm+" --fields id,name"))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed retrieving client-scopes: %s", err)
		return nil, err
	}
	if len(out) == 0 {
		err := errors.New("Component Keycloak failed; clients JSON from Keycloak is zero length")
		ctx.Log().Error(err)
		return nil, err
	}
	err = json.Unmarshal([]byte(out), &keycloakClientScopes)
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed ummarshalling client json: %v", err)
		return nil, err
	}
	return keycloakClientScopes, nil
}

func getClientID(keycloakClients KeycloakClients, clientName string) string {
	for _, keycloakClient := range keycloakClients {
		if keycloakClient.ClientID == clientName {
			return keycloakClient.ID
		}
	}
	return ""
}

func getClientScopeName(keycloakClientScopes KeycloakClientScopes, groupname string) string {
	for _, keycloakClient := range keycloakClientScopes {
		if keycloakClient.Name == groupname {
			return keycloakClient.Name
		}
	}
	return ""
}

func (c KeycloakComponent) isKeycloakReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return ready.StatefulSetsAreReady(ctx.Log(), ctx.Client(), c.AvailabilityObjects.StatefulsetNames, 1, prefix)
}

// isPodReady determines if the pod is running by checking for a Ready condition with Status equal True
func isPodReady(pod *corev1.Pod) bool {
	conditions := pod.Status.Conditions
	for j := range conditions {
		if conditions[j].Type == "Ready" && conditions[j].Status == "True" {
			return true
		}
	}
	return false
}

// GetOverrides gets the install overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.Keycloak != nil {
			return effectiveCR.Spec.Components.Keycloak.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.Keycloak != nil {
			return effectiveCR.Spec.Components.Keycloak.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

// deleteStatefulSet deletes the Keycloak StatefulSet before upgrade
func deleteStatefulSet(ctx spi.ComponentContext) error {
	keycloakComp := ctx.EffectiveCR().Spec.Components.Keycloak
	if keycloakComp == nil {
		return nil
	}

	// Get the StatefulSet for Keycloak
	ctxClient := ctx.Client()
	statefulSet := appv1.StatefulSet{}

	ctx.Log().Infof("Delete StatefulSet %s/%s, if it exists", ComponentNamespace, ComponentName)
	err := ctxClient.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, &statefulSet)
	if err != nil {
		ctx.Log().Infof("StatefulSet %s/%s doesn't exist", ComponentNamespace, ComponentName)
		return nil
	}

	// Delete the StatefulSet
	deleteOpts := []client.DeleteOption{client.PropagationPolicy(metav1.DeletePropagationOrphan)}
	if err := ctxClient.Delete(context.TODO(), &statefulSet, deleteOpts...); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to delete StatefulSet %s/%s: %v", ComponentNamespace, ComponentName, err)
	}
	return nil
}

// deleteHeadlessService deletes the keycloak-headless service after deleting the StatefulSet, before the upgrade
func deleteHeadlessService(ctx spi.ComponentContext) error {
	keycloakComp := ctx.EffectiveCR().Spec.Components.Keycloak
	if keycloakComp == nil {
		return nil
	}
	// Get and delete the headless service associated with the StatefulSet
	service := &corev1.Service{}
	ctxClient := ctx.Client()
	ctx.Log().Infof("Delete headless service %s/%s, if it exists", ComponentNamespace, headlessService)
	if err := ctxClient.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: headlessService}, service); err != nil {
		ctx.Log().Infof("Headless service %s/%s doesn't exist", ComponentNamespace, headlessService)
		return nil
	}
	if err := ctxClient.Delete(context.TODO(), service); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to delete headless service %s/%s: %v", ComponentNamespace, headlessService, err)
	}
	return nil
}

func populateClientTemplate(ctx spi.ComponentContext, tmpl string, clientID string, subdomain *string) (string, error) {
	data := templateData{}

	// Update verrazzano-pkce client redirect and web origin uris if deprecated host exists in the ingress
	osHostExists, err := DoesDeprecatedIngressHostExist(ctx, constants.VerrazzanoSystemNamespace)
	if err != nil {
		ctx.Log().Errorf("Error retrieving the ingressList : %v", err)
	}
	// Set bool value if deprecated host exists
	data.OSHostExists = osHostExists

	// Get DNS Domain Configuration
	var dnsSubDomain string
	if subdomain == nil {
		dnsSubDomain, err = getDNSDomain(ctx.Client(), ctx.EffectiveCR())
		if err != nil {
			ctx.Log().Errorf("Component Keycloak failed retrieving DNS sub domain: %v", err)
			return "", err
		}
	} else {
		dnsSubDomain = *subdomain
	}
	ctx.Log().Debugf("populateClientTemplate: DNSDomain = %s", dnsSubDomain)

	data.DNSSubDomain = dnsSubDomain

	data.ClientID = clientID

	// use template to get populate template with data
	var b bytes.Buffer
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return "", err
	}

	err = t.Execute(&b, &data)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

// GetRancherClientSecretFromKeycloak returns the secret from rancher client in Keycloak
func GetRancherClientSecretFromKeycloak(ctx spi.ComponentContext) (string, error) {
	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return "", err
	}

	// Login to Keycloak
	err = LoginKeycloak(ctx, cfg, cli)
	if err != nil {
		return "", err
	}

	kcClients, err := getKeycloakClients(ctx)
	if err != nil {
		return "", err
	}

	id := ""
	for _, kcClient := range kcClients {
		if kcClient.ClientID == "rancher" {
			id = kcClient.ID
		}
	}

	if id == "" {
		ctx.Log().Debugf("GetRancherClientSecretFromKeycloak: rancher client does not exist")
		return "", nil
	}

	var clientSecret KeycloakClientSecret
	// Get the Client secret JSON array
	out, _, err := k8sutil.ExecPod(cli, cfg, keycloakPod(), ComponentName, bashCMD(kcAdminScript+" get clients/"+id+"/client-secret -r "+vzconst.VerrazzanoOIDCSystemRealm))
	if err != nil {
		ctx.Log().Errorf("failed retrieving rancher client secret from keycloak: %s", err)
		return "", err
	}
	if len(out) == 0 {
		err = errors.New("client secret json from keycloak is zero length")
		ctx.Log().Error(err)
		return "", err
	}

	err = json.Unmarshal([]byte(out), &clientSecret)
	if err != nil {
		ctx.Log().Errorf("failed ummarshalling client secret json: %v", err)
		return "", err
	}

	if clientSecret.Value == "" {
		return "", ctx.Log().ErrorNewErr("client secret is empty")
	}

	return clientSecret.Value, nil
}

type (
	ArgoClientSecretProvider interface {
		GetClientSecret(ctx spi.ComponentContext) (string, error)
	}

	// Gets the client secret from keycloak
	DefaultArgoClientSecretProvider struct{}
)

// GetClientSecret returns the secret from Argo CD client in Keycloak
func (p DefaultArgoClientSecretProvider) GetClientSecret(ctx spi.ComponentContext) (string, error) {
	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return "", err
	}

	// Login to Keycloak
	err = LoginKeycloak(ctx, cfg, cli)
	if err != nil {
		return "", err
	}

	kcClients, err := getKeycloakClients(ctx)
	if err != nil {
		return "", err
	}

	id := ""
	for _, kcClient := range kcClients {
		if kcClient.ClientID == "argocd" {
			id = kcClient.ID
		}
	}

	if id == "" {
		ctx.Log().Debugf("GetArgoCDClientSecretFromKeycloak: Argo CD client does not exist")
		err = errors.New("Argo CD client does not exist")
		return "", err
	}

	var clientSecret KeycloakClientSecret
	// Get the Client secret JSON array
	out, _, err := k8sutil.ExecPod(cli, cfg, keycloakPod(), ComponentName, bashCMD(kcAdminScript+" get clients/"+id+"/client-secret -r "+vzconst.VerrazzanoOIDCSystemRealm))
	if err != nil {
		ctx.Log().Errorf("failed retrieving argocd client secret from keycloak: %s", err)
		return "", err
	}
	if len(out) == 0 {
		err = errors.New("client secret json from keycloak is zero length")
		ctx.Log().Error(err)
		return "", err
	}

	err = json.Unmarshal([]byte(out), &clientSecret)
	if err != nil {
		ctx.Log().Errorf("failed ummarshalling client secret json: %v", err)
		return "", err
	}

	if clientSecret.Value == "" {
		return "", ctx.Log().ErrorNewErr("client secret is empty")
	}

	return clientSecret.Value, nil
}

func generateClientSecret(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, clientName string, createClientOutput string, kcPod *corev1.Pod) error {
	if len(createClientOutput) == 0 {
		err := fmt.Errorf("Component Keycloak failed; %s client ID from Keycloak is zero length", clientName)
		ctx.Log().Error(err)
		return err
	}

	arr := strings.Split(string(createClientOutput), "'")
	if len(arr) != 3 {
		return fmt.Errorf("Component Keycloak failed parsing output returned from %s Client create stdout returned = %s", clientName, createClientOutput)
	}

	clientID := arr[1]
	ctx.Log().Debugf("generateClientSecret: %s Client ID = %s", clientName, clientID)

	// Create client secret
	clientCreateSecretCmd := kcAdminScript + " create clients/" + clientID + "/client-secret" + " -r " + vzconst.VerrazzanoOIDCSystemRealm
	ctx.Log().Debugf("generateClientSecret: Create %s client secret Cmd = %s", clientName, clientCreateSecretCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(clientCreateSecretCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed creating %s client secret: stdout = %s, stderr = %s", clientName, stdout, stderr)
		return err
	}

	ctx.Log().Oncef("Component Keycloak generated client secret for client: %v", clientName)
	return nil
}

// GetVerrazzanoUserFromKeycloak returns the user verrazzano in Keycloak
func GetVerrazzanoUserFromKeycloak(ctx spi.ComponentContext) (*KeycloakUser, error) {
	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return nil, err
	}

	// Login to Keycloak
	err = LoginKeycloak(ctx, cfg, cli)
	if err != nil {
		return nil, err
	}

	kcUsers, err := getKeycloakUsers(ctx, cfg, cli, keycloakPod())
	if err != nil {
		return nil, err
	}

	var vzUser KeycloakUser
	found := false
	for _, user := range kcUsers {
		if user.Username == "verrazzano" {
			vzUser = user
			found = true
			break
		}
	}

	if !found {
		return nil, ctx.Log().ErrorfThrottledNewErr("GetVerrazzanoUserIDFromKeycloak: verrazzano user does not exist")
	}

	return &vzUser, nil
}

func updateRancherClientSecretForKeycloakAuthConfig(ctx spi.ComponentContext) error {
	log := ctx.Log()
	clientSecret, err := GetRancherClientSecretFromKeycloak(ctx)
	if err != nil {
		return log.ErrorfThrottledNewErr("failed updating client secret in keycloak auth config, unable to fetch rancher client secret: %s", err.Error())
	}

	authConfig := make(map[string]interface{})
	authConfig[common.AuthConfigKeycloakAttributeClientSecret] = clientSecret
	return common.UpdateKeycloakOIDCAuthConfig(ctx, authConfig)
}

// addRealmRoleToUser adds a realm role to the given user in the target realm
func addRealmRoleToUser(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, userName, targetRealm, roleName string) error {
	kcPod := keycloakPod()
	addRoleCmd := fmt.Sprintf("%s add-roles -r %s --uusername %s --rolename %s", kcAdminScript, targetRealm, userName, roleName)
	ctx.Log().Debugf("Adding realm role %s to user %s, using command: %s", roleName, userName, addRoleCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(addRoleCmd))
	if err != nil {
		ctx.Log().Errorf("Adding realm role %s to the user %s failed: stdout = &s, stderr = %s, error = %s", roleName, userName, stdout, stderr, err.Error())
		return err
	}
	ctx.Log().Oncef("Added realm role %s to the user %s", roleName, userName)
	return nil
}

// addClientRoleToUser adds client role to the given user in the target realm
func addClientRoleToUser(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, userName, clientID, targetRealm, roleName string) error {
	kcPod := keycloakPod()
	addRoleCmd := kcAdminScript + " add-roles -r " + targetRealm + " --uusername " + userName + " --cclientid " + clientID + " --rolename " + roleName
	ctx.Log().Debugf("Adding client role %s to the user %s, using command: %s", roleName, userName, addRoleCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(addRoleCmd))
	if err != nil {
		ctx.Log().Errorf("Adding client role %s to the user %s failed: stdout = %s, stderr = %s, error = %s", roleName, userName, stdout, stderr, err.Error())
		return err
	}
	ctx.Log().Oncef("Added client role %s to the user %s", roleName, userName)
	return nil
}

// DoesDeprecatedIngressHostExist returns true if ingress host exists
func DoesDeprecatedIngressHostExist(ctx spi.ComponentContext, namespace string) (bool, error) {
	ingressList := &networkv1.IngressList{}

	listOptions := &client.ListOptions{Namespace: namespace}
	err := ctx.Client().List(context.TODO(), ingressList, listOptions)
	if err != nil && len(ingressList.Items) == 0 {
		return false, err
	}
	for _, ingress := range ingressList.Items {
		if len(ingress.Spec.Rules) > 0 && strings.HasPrefix(ingress.Spec.Rules[0].Host, "elasticsearch") {
			return true, nil
		}
	}
	return false, nil
}

// isDeleteStatefulSetRequired determines whether deleting the Keycloak StatefulSet is required.
// The StatefulSet defined by the helm chart for Keycloak 20.0.1 (introduced first in in Verrazzano 1.5.0) contains changes
// to fields other than 'replicas', 'template', and 'updateStrategy'. So an upgrade from Keycloak 15.0.2 to 20.0.1 requires
// the workaround to delete the StatefulSet before an upgrade.
func isDeleteStatefulSetRequired(ctx spi.ComponentContext) (bool, error) {
	isDeleteRequired := true
	if ctx.ActualCR() != nil {
		installedVersion, err := semver.NewSemVersion(ctx.ActualCR().Status.Version)
		if err != nil {
			return isDeleteRequired, nil
		}
		ctx.Log().Debugf("Verrazzano version installed: %s", installedVersion)

		versionToUpgrade, err := semver.NewSemVersion(ctx.ActualCR().Spec.Version)
		if err != nil {
			return isDeleteRequired, nil
		}
		ctx.Log().Debugf("Verrazzano version for upgrade: %s", versionToUpgrade)

		minVersion, err := semver.NewSemVersion(constants.VerrazzanoVersion1_5_0)
		if err != nil {
			return isDeleteRequired, nil
		}

		// Deleting StatefulSet is not required when one of the conditions is true
		// - Verrazzano version to upgrade is earlier than 1.5.0
		// - Verrazzano version installed is 1.5.0 or higher
		if versionToUpgrade.IsLessThan(minVersion) || !installedVersion.IsLessThan(minVersion) {
			ctx.Log().Oncef("Deleting Keycloak StatefulSet is not required during pre-upgrade")
			isDeleteRequired = false
		}
	}
	return isDeleteRequired, nil
}
