// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"reflect"
	"strings"
	"text/template"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzpassword "github.com/verrazzano/verrazzano/pkg/security/password"
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
	"sigs.k8s.io/yaml"
)

const (
	dnsTarget               = "dnsTarget"
	rulesHost               = "rulesHost"
	tlsHosts                = "tlsHosts"
	tlsSecret               = "tlsSecret"
	keycloakCertificateName = "keycloak-tls"
	vzSysRealm              = "verrazzano-system"
	vzUsersGroup            = "verrazzano-users"
	vzAdminGroup            = "verrazzano-admins"
	vzMonitorGroup          = "verrazzano-monitors"
	vzSystemGroup           = "verrazzano-system-users"
	vzAPIAccessRole         = "vz_api_access"
	vzUserName              = "verrazzano"
	vzInternalPromUser      = "verrazzano-prom-internal"
	vzInternalEsUser        = "verrazzano-es-internal"
	keycloakPodName         = "keycloak-0"
	realmManagement         = "realm-management"
	viewUsersRole           = "view-users"
	noRouterAddr            = "mysql-instances"
	routerAddr              = "mysql"
	dbHostKey               = "mysql.dbHost"
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
`

var pkceTmpl = `
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

var pkceClientUrisTemplate = `
	"redirectUris": [
	  "https://verrazzano.{{.DNSSubDomain}}/*",
	  "https://verrazzano.{{.DNSSubDomain}}/verrazzano/authcallback",
	  "https://opensearch.vmi.system.{{.DNSSubDomain}}/*",
	  "https://opensearch.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
	  "https://prometheus.vmi.system.{{.DNSSubDomain}}/*",
	  "https://prometheus.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
	  "https://grafana.vmi.system.{{.DNSSubDomain}}/*",
	  "https://grafana.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
	  "https://opensearchdashboards.vmi.system.{{.DNSSubDomain}}/*",
	  "https://opensearchdashboards.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
	  "https://kiali.vmi.system.{{.DNSSubDomain}}/*",
	  "https://kiali.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
	  "https://jaeger.{{.DNSSubDomain}}/*"
	],
	"webOrigins": [
	  "https://verrazzano.{{.DNSSubDomain}}",
	  "https://opensearch.vmi.system.{{.DNSSubDomain}}",
	  "https://prometheus.vmi.system.{{.DNSSubDomain}}",
	  "https://grafana.vmi.system.{{.DNSSubDomain}}",
	  "https://opensearchdashboards.vmi.system.{{.DNSSubDomain}}",
	  "https://kiali.vmi.system.{{.DNSSubDomain}}",
	  "https://jaeger.{{.DNSSubDomain}}"
	]
`

var pkceClientUrisTemplateForDeprecatedOSHosts = `

		"redirectUris": [
		  "https://verrazzano.{{.DNSSubDomain}}/*",
		  "https://verrazzano.{{.DNSSubDomain}}/verrazzano/authcallback",
		  "https://opensearch.vmi.system.{{.DNSSubDomain}}/*",
		  "https://opensearch.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
		  "https://elasticsearch.vmi.system.{{.DNSSubDomain}}/*",
		  "https://elasticsearch.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
		  "https://prometheus.vmi.system.{{.DNSSubDomain}}/*",
		  "https://prometheus.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
		  "https://grafana.vmi.system.{{.DNSSubDomain}}/*",
		  "https://grafana.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
		  "https://kibana.vmi.system.{{.DNSSubDomain}}/*",
		  "https://kibana.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
		  "https://opensearchdashboards.vmi.system.{{.DNSSubDomain}}/*",
		  "https://opensearchdashboards.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
		  "https://kiali.vmi.system.{{.DNSSubDomain}}/*",
		  "https://kiali.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
		  "https://jaeger.{{.DNSSubDomain}}/*"
		],
		"webOrigins": [
		  "https://verrazzano.{{.DNSSubDomain}}",
		  "https://opensearch.vmi.system.{{.DNSSubDomain}}",
	      "https://elasticsearch.vmi.system.{{.DNSSubDomain}}",
		  "https://prometheus.vmi.system.{{.DNSSubDomain}}",
		  "https://grafana.vmi.system.{{.DNSSubDomain}}",
	      "https://kibana.vmi.system.{{.DNSSubDomain}}",
		  "https://opensearchdashboards.vmi.system.{{.DNSSubDomain}}",
		  "https://kiali.vmi.system.{{.DNSSubDomain}}",
		  "https://jaeger.{{.DNSSubDomain}}"
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

// KeycloakClients represents an array of clients currently configured in Keycloak
type KeycloakClients []struct {
	ID       string `json:"id"`
	ClientID string `json:"clientId"`
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
		ObjectMeta: metav1.ObjectMeta{Name: "keycloak", Namespace: "keycloak"},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &ingress, func() error {
		dnsSuffix, _ := vzconfig.GetDNSSuffix(ctx.Client(), ctx.EffectiveCR())
		ingress.Annotations["cert-manager.io/common-name"] = fmt.Sprintf("%s.%s.%s",
			ComponentName, ctx.EffectiveCR().Spec.EnvironmentName, dnsSuffix)
		// update target annotation on Keycloak Ingress for external DNS
		if vzconfig.IsExternalDNSEnabled(ctx.EffectiveCR()) {
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
func updateKeycloakUris(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, kcPod *corev1.Pod, clientID string, uriTemplate string) error {
	data, err := populateSubdomainInTemplate(ctx, "{"+uriTemplate+"}")
	if err != nil {
		return err
	}

	// Update client
	updateClientCmd := "/opt/jboss/keycloak/bin/kcadm.sh update clients/" + clientID + " -r " + vzSysRealm + " -b '" +
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
	// Make sure the Keycloak pod is ready
	pod := keycloakPod()
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}, pod)
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed to get pod %s: %v", pod.Name, err)
		return err
	}
	if !isPodReady(pod) {
		ctx.Log().Progressf("Component Keycloak waiting for pod %s to be ready", pod.Name)
		return fmt.Errorf("Waiting for pod %s to be ready", pod.Name)
	}

	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return err
	}

	// If ephemeral storage is configured, additional steps may be required to
	// rebuild the configuration lost due to MySQL pod getting restarted.
	if (ctx.EffectiveCR().Spec.Components.Keycloak != nil) && (ctx.EffectiveCR().Spec.Components.Keycloak.MySQL.VolumeSource == nil) {
		// When the MySQL pod restarts and using ephemeral storage, the
		// login to Keycloak will fail.  Need to recycle the Keycloak pod
		// to resolve the condition.
		err = loginKeycloak(ctx, cfg, cli)
		if err != nil {
			err2 := ctx.Client().Delete(context.TODO(), pod)
			if err2 != nil {
				ctx.Log().Errorf("Component Keycloak failed to recycle pod %s: %v", pod.Name, err2)
			}
			return err
		}
	}

	// Login to Keycloak
	err = loginKeycloak(ctx, cfg, cli)
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

	// Granting Roles to Groups
	err = grantRolesToGroups(ctx, cfg, cli, userGroupID, adminGroupID, monitorGroupID)
	if err != nil {
		return err
	}

	// Creating Verrazzano User
	err = createUser(ctx, cfg, cli, vzUserName, "verrazzano", vzAdminGroup, "Verrazzano", "Admin")
	if err != nil {
		return err
	}

	// Creating Verrazzano Internal Prometheus User
	err = createUser(ctx, cfg, cli, vzInternalPromUser, "verrazzano-prom-internal", vzSystemGroup, "", "")
	if err != nil {
		return err
	}

	// Creating Verrazzano Internal ES User
	err = createUser(ctx, cfg, cli, vzInternalEsUser, "verrazzano-es-internal", vzSystemGroup, "", "")
	if err != nil {
		return err
	}

	// Update verrazzano-pkce client redirect and web origin uris if deprecated OS host exists in the ingress
	osHostExists, err := pkg.DoesIngressHostExist(constants.VerrazzanoSystemNamespace, constants.OpensearchIngress)
	if err != nil {
		return err
	}
	if osHostExists {
		pkceClientUrisTemplate = pkceClientUrisTemplateForDeprecatedOSHosts
	}

	// Create verrazzano-pkce client
	err = createOrUpdateClient(ctx, cfg, cli, "verrazzano-pkce", pkceTmpl, pkceClientUrisTemplate, false)
	if err != nil {
		return err
	}

	// Creating verrazzano-pg client
	err = createOrUpdateClient(ctx, cfg, cli, "verrazzano-pg", pgClient, "", true)
	if err != nil {
		return err
	}

	if vzconfig.IsRancherEnabled(ctx.ActualCR()) {
		// Creating rancher client
		err = createOrUpdateClient(ctx, cfg, cli, "rancher", rancherClientTmpl, rancherClientUrisTemplate, true)
		if err != nil {
			return err
		}

		// Update Keycloak AuthConfig for Rancher with client secret
		err = updateRancherClientSecretForKeycloakAuthConfig(ctx)
		if err != nil {
			return err
		}

		// Add view-users role to verrazzano user
		err = addClientRoleToUser(ctx, cfg, cli, vzUserName, realmManagement, vzSysRealm, viewUsersRole)
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

	// Enabling vzSysRealm realm
	err = enableVerrazzanoSystemRealm(ctx, cfg, cli)
	if err != nil {
		return err
	}

	// Removing login config file
	err = removeLoginConfigFile(ctx, cfg, cli)
	if err != nil {
		return err
	}

	ctx.Log().Oncef("Component Keycloak successfully configured realm %s", vzSysRealm)
	return nil
}

// loginKeycloak logs into Keycloak so kcadm API calls can be made
func loginKeycloak(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface) error {
	// Get the Keycloak admin password
	secret := &corev1.Secret{}
	err := ctx.Client().Get(context.TODO(), client.ObjectKey{
		Namespace: "keycloak",
		Name:      "keycloak-http",
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
	ctx.Log().Debug("loginKeycloak: Successfully retrieved Keycloak password")

	// Login to Keycloak
	kcPod := keycloakPod()
	loginCmd := "/opt/jboss/keycloak/bin/kcadm.sh config credentials --server http://localhost:8080/auth --realm master --user keycloakadmin --password " + keycloakpw
	ctx.Log().Debugf("loginKeycloak: Login Cmd = %s", maskPw(loginCmd))
	stdOut, stdErr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(loginCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed logging into Keycloak: stdout = %s: stderr = %s, err = %v", stdOut, stdErr, maskPw(err.Error()))
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
	dnsSuffix, err := vzconfig.GetDNSSuffix(c, vz)
	if err != nil {
		return "", err
	}
	dnsDomain := fmt.Sprintf("%s.%s", vz.Spec.EnvironmentName, dnsSuffix)
	return dnsDomain, nil
}

func createVerrazzanoSystemRealm(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface) error {
	kcPod := keycloakPod()
	realm := "realm=" + vzSysRealm
	checkRealmExistsCmd := "/opt/jboss/keycloak/bin/kcadm.sh get realms/" + vzSysRealm
	ctx.Log().Debugf("createVerrazzanoSystemRealm: Check Verrazzano System Realm Exists Cmd = %s", checkRealmExistsCmd)
	_, _, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(checkRealmExistsCmd))
	if err != nil {
		ctx.Log().Debug("createVerrazzanoSystemRealm: Verrazzano System Realm doesn't exist: Creating it")
		createRealmCmd := "/opt/jboss/keycloak/bin/kcadm.sh create realms -s " + realm + " -s enabled=false"
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

	cmd := fmt.Sprintf("/opt/jboss/keycloak/bin/kcadm.sh create %s -r %s -s %s", groupsResource, vzSysRealm, groupName)
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
	createRoleCmd := "/opt/jboss/keycloak/bin/kcadm.sh create roles -r " + vzSysRealm + " -s " + role
	ctx.Log().Debugf("createVerrazzanoRole: Create Verrazzano API Access Role Cmd = %s", createRoleCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createRoleCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed creating Verrazzano API Access Role: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Once("Component Keycloak successfully created the Verrazzano API access role")
	return nil
}

func grantRolesToGroups(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, userGroupID string, adminGroupID string, monitorGroupID string) error {
	// Keycloak API does not fail if Role already exists as of 15.0.3
	kcPod := keycloakPod()
	// Granting vz_api_access role to verrazzano users group
	grantAPIAccessToVzUserGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + vzSysRealm + " --gid " + userGroupID + " --rolename " + vzAPIAccessRole
	ctx.Log().Debugf("grantRolesToGroups: Grant API Access to VZ Users Cmd = %s", grantAPIAccessToVzUserGroupCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(grantAPIAccessToVzUserGroupCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed granting api access role to Verrazzano users group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Once("Component Keycloak successfully granted the access role to the Verrazzano user group")

	return nil
}

func createUser(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, userName string, secretName string, groupName string, firstName string, lastName string) error {
	kcPod := keycloakPod()
	keycloakUsers, err := getKeycloakUsers(ctx, cfg, cli, kcPod)
	if err == nil && userExists(keycloakUsers, userName) {
		return nil
	}
	vzUser := "username=" + userName
	vzUserGroup := "groups[0]=/" + vzUsersGroup + "/" + groupName
	createVzUserCmd := "/opt/jboss/keycloak/bin/kcadm.sh create users -r " + vzSysRealm + " -s " + vzUser + " -s " + vzUserGroup + " -s enabled=true"
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

	vzpw, err := getSecretPassword(ctx, "verrazzano-system", secretName)
	if err != nil {
		return err
	}
	setVZUserPwCmd := "/opt/jboss/keycloak/bin/kcadm.sh set-password -r " + vzSysRealm + " --username " + userName + " --new-password " + vzpw
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
func createOrUpdateClient(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, clientName string, clientTemplate string, uriTemplate string, generateSecret bool) error {
	keycloakClients, err := getKeycloakClients(ctx)
	if err != nil {
		return err
	}

	kcPod := keycloakPod()
	if clientID := getClientID(keycloakClients, clientName); clientID != "" {
		if uriTemplate != "" {
			err := updateKeycloakUris(ctx, cfg, cli, kcPod, clientID, uriTemplate)
			if err != nil {
				return err
			}
		}

		return nil
	}

	data, err := populateSubdomainInTemplate(ctx, clientTemplate)
	if err != nil {
		return err
	}

	// Create client
	clientCreateCmd := "/opt/jboss/keycloak/bin/kcadm.sh create clients -r " + vzSysRealm + " -f - <<\\END" +
		data +
		"END"

	ctx.Log().Debugf("createOrUpdateClient: Create %s client Cmd = %s", clientName, clientCreateCmd)
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

	ctx.Log().Debugf("createOrUpdateClient: Created %s client", clientName)
	ctx.Log().Oncef("Component Keycloak successfully created client %s", clientName)
	return nil
}

func setPasswordPolicyForRealm(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, realmName string, policy string) error {
	kcPod := keycloakPod()
	setPolicyCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/" + realmName + " -s \"" + policy + "\""
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
	setLoginThemeCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/" + realmName + " -s loginTheme=" + loginTheme
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
	setVzEnableRealmCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/" + vzSysRealm + " -s enabled=true"
	ctx.Log().Debugf("enableVerrazzanoSystemRealm: Enabling vzSysRealm realm Cmd = %s", setVzEnableRealmCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setVzEnableRealmCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed enabling vzSysRealm realm: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("enableVerrazzanoSystemRealm: Enabled vzSysRealm realm")
	ctx.Log().Once("Component Keycloak successfully enabled the vzSysRealm realm")

	return nil
}

func removeLoginConfigFile(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface) error {
	kcPod := keycloakPod()
	removeLoginConfigFileCmd := "rm /root/.keycloak/kcadm.config"
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
	cmd := fmt.Sprintf("/opt/jboss/keycloak/bin/kcadm.sh get groups -r %s", vzSysRealm)
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
	out, _, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD("/opt/jboss/keycloak/bin/kcadm.sh get-roles -r "+vzSysRealm))
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
	out, _, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD("/opt/jboss/keycloak/bin/kcadm.sh get users -r "+vzSysRealm))
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
	out, _, err := k8sutil.ExecPod(cli, cfg, keycloakPod(), ComponentName, bashCMD("/opt/jboss/keycloak/bin/kcadm.sh get clients -r "+vzSysRealm+" --fields id,clientId"))
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

func getClientID(keycloakClients KeycloakClients, clientName string) string {

	for _, keycloakClient := range keycloakClients {
		if keycloakClient.ClientID == clientName {
			return keycloakClient.ID
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

// upgradeStatefulSet - determine if the replica count for the StatefulSet needs
// to be scaled down before the upgrade.  The affinity rules installed by default
// prior to the 1.4 release conflict with the new affinity rules being overridden
// by Verrazzano (the upgrade will never complete).  The work around is to scale
// down the replica count prior to upgrade, which terminates the Keycloak pods, and
// then do the upgrade.
func upgradeStatefulSet(ctx spi.ComponentContext) error {
	keycloakComp := ctx.EffectiveCR().Spec.Components.Keycloak
	if keycloakComp == nil {
		return nil
	}

	// Get the combine set of value overrides into a single array of string
	overrides, err := common.GetInstallOverridesYAML(ctx, keycloakComp.ValueOverrides)
	if err != nil {
		return err
	}

	// Is there an override for affinity?
	found := false
	affinityOverride := &corev1.Affinity{}
	for _, overrideYaml := range overrides {
		if strings.Contains(overrideYaml, "affinity: |") {
			found = true

			// Convert the affinity override from yaml to a struct
			affinityField, err := common.ExtractValueFromOverrideString(overrideYaml, "affinity")
			if err != nil {
				return err
			}
			err = yaml.Unmarshal([]byte(fmt.Sprintf("%v", affinityField)), affinityOverride)
			if err != nil {
				return err
			}
			break
		}
	}
	if !found {
		return nil
	}

	// Get the StatefulSet for Keycloak
	client := ctx.Client()
	statefulSet := appv1.StatefulSet{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, &statefulSet)
	if err != nil {
		return err
	}

	// Nothing to do if the affinity definitions are the same
	if reflect.DeepEqual(affinityOverride, statefulSet.Spec.Template.Spec.Affinity) {
		return nil
	}

	// Scale replica count to 0 to cause all pods to terminate, upgrade will restore replica count
	*statefulSet.Spec.Replicas = 0
	err = client.Update(context.TODO(), &statefulSet)
	if err != nil {
		return err
	}

	return nil
}

func populateSubdomainInTemplate(ctx spi.ComponentContext, tmpl string) (string, error) {
	data := templateData{}
	// Get DNS Domain Configuration
	dnsSubDomain, err := getDNSDomain(ctx.Client(), ctx.EffectiveCR())
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed retrieving DNS sub domain: %v", err)
		return "", err
	}
	ctx.Log().Debugf("populateSubdomainInTemplate: DNSDomain returned %s", dnsSubDomain)

	data.DNSSubDomain = dnsSubDomain

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
	err = loginKeycloak(ctx, cfg, cli)
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
	out, _, err := k8sutil.ExecPod(cli, cfg, keycloakPod(), ComponentName, bashCMD("/opt/jboss/keycloak/bin/kcadm.sh get clients/"+id+"/client-secret -r "+vzSysRealm))
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
	clientCreateSecretCmd := "/opt/jboss/keycloak/bin/kcadm.sh create clients/" + clientID + "/client-secret" + " -r " + vzSysRealm
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
	err = loginKeycloak(ctx, cfg, cli)
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

// addClientRoleToUser adds client role to the given user in the target realm
func addClientRoleToUser(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, userName, clientID, targetRealm, roleName string) error {
	kcPod := keycloakPod()
	addRoleCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + targetRealm + " --uusername " + userName + " --cclientid " + clientID + " --rolename " + roleName
	ctx.Log().Debugf("Adding client role %s to the user %s, using command: %s", roleName, userName, addRoleCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(addRoleCmd))
	if err != nil {
		ctx.Log().Errorf("Adding client role %s to the user %s failed: stdout = %s, stderr = %s, error = %s", roleName, userName, stdout, stderr, err.Error())
		return err
	}
	ctx.Log().Oncef("Added client role %s to the user %s", roleName, userName)
	return nil
}
