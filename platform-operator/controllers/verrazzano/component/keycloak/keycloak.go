// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	vzpassword "github.com/verrazzano/verrazzano/pkg/security/password"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
)

// Define the keycloak Key:Value pair for init container.
// We need to replace image using the real image in the bom
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

const pkceTmpl = `
{
      "clientId" : "verrazzano-pkce",
      "enabled": true,
      "surrogateAuthRequired": false,
      "alwaysDisplayInConsole": false,
      "clientAuthenticatorType": "client-secret",
      "redirectUris": [
        "https://verrazzano.{{.DNSSubDomain}}/*",
        "https://verrazzano.{{.DNSSubDomain}}/verrazzano/authcallback",
        "https://elasticsearch.vmi.system.{{.DNSSubDomain}}/*",
        "https://elasticsearch.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
        "https://prometheus.vmi.system.{{.DNSSubDomain}}/*",
        "https://prometheus.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
        "https://grafana.vmi.system.{{.DNSSubDomain}}/*",
        "https://grafana.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
        "https://kibana.vmi.system.{{.DNSSubDomain}}/*",
        "https://kibana.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
        "https://kiali.vmi.system.{{.DNSSubDomain}}/*",
        "https://kiali.vmi.system.{{.DNSSubDomain}}/_authentication_callback"
      ],
      "webOrigins": [
        "https://verrazzano.{{.DNSSubDomain}}",
        "https://elasticsearch.vmi.system.{{.DNSSubDomain}}",
        "https://prometheus.vmi.system.{{.DNSSubDomain}}",
        "https://grafana.vmi.system.{{.DNSSubDomain}}",
        "https://kibana.vmi.system.{{.DNSSubDomain}}",
		"https://kiali.vmi.system.{{.DNSSubDomain}}"
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
      "secret" : "de05ccdc-67df-47f3-81f6-37e61d195aba",
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

// KeycloakUsers is an array of users configured in Keycloak
type KeycloakUsers []struct {
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

type templateData struct {
	DNSSubDomain string
}

// Unit testing support
type bashFuncSig func(inArgs ...string) (string, string, error)

var bashFunc bashFuncSig = vzos.RunBash

func setBashFunc(f bashFuncSig) {
	bashFunc = f
}

var execCommand = exec.Command

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

	// this secret contains the keycloak TLS certificate created by cert-manager during the original keycloak installation
	kvs = append(kvs, bom.KeyValue{
		Key:   tlsSecret,
		Value: keycloakCertificateName,
	})

	return kvs, nil
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

// updateKeycloakUris calls a bash script to update the Keycloak rewrite and weborigin uris
func updateKeycloakUris(ctx spi.ComponentContext) error {

	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return err
	}

	err = loginKeycloak(ctx, cfg, cli)
	if err != nil {
		return err
	}

	// Get the Client ID JSON array
	keycloakClients, err := getKeycloakClients(ctx)
	if err != nil {
		return err
	}

	// Get the client ID for verrazzano-pkce
	id := getClientID(keycloakClients, "verrazzano-pkce")
	if id == "" {
		err := errors.New("Component Keycloak failed retrieving ID for Keycloak user, zero length")
		ctx.Log().Error(err)
		return err
	}
	ctx.Log().Debug("Keycloak Post Upgrade: Successfully retrieved clientID")

	// Get DNS Domain Configuration
	dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed retrieving DNS sub domain: %v", err)
		return err
	}
	ctx.Log().Debugf("Keycloak Post Upgrade: DNSDomain returned %s", dnsSubDomain)

	// Call the Script and Update the URIs
	scriptName := filepath.Join(config.GetInstallDir(), "update-kiali-redirect-uris.sh")
	if _, stderr, err := bashFunc(scriptName, id, dnsSubDomain); err != nil {
		ctx.Log().Errorf("Component Keycloak failed updating KeyCloak URIs %v: %s", err, stderr)
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
	userGroupID, err := createVerrazzanoUsersGroup(ctx)
	if err != nil {
		return err
	}
	if userGroupID == "" {
		err := errors.New("Component Keycloak failed; user Group ID from Keycloak is zero length")
		ctx.Log().Error(err)
		return err
	}

	// Create Verrazzano Admin Group
	adminGroupID, err := createVerrazzanoAdminGroup(ctx, userGroupID)
	if err != nil {
		return err
	}
	if adminGroupID == "" {
		err := errors.New("Component Keycloak failed; admin group ID from Keycloak is zero length")
		ctx.Log().Error(err)
		return err
	}

	// Create Verrazzano Project Monitors Group
	monitorGroupID, err := createVerrazzanoMonitorsGroup(ctx, userGroupID)
	if err != nil {
		return err
	}
	if monitorGroupID == "" {
		err = errors.New("Component Keycloak failed; monitor group ID from Keycloak is zero length")
		ctx.Log().Error(err)
		return err
	}

	// Create Verrazzano System Group
	err = createVerrazzanoSystemGroup(ctx, cfg, cli, userGroupID)
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
	err = createUser(ctx, cfg, cli, vzUserName, "verrazzano", vzAdminGroup)
	if err != nil {
		return err
	}

	// Creating Verrazzano Internal Prometheus User
	err = createUser(ctx, cfg, cli, vzInternalPromUser, "verrazzano-prom-internal", vzSystemGroup)
	if err != nil {
		return err
	}

	// Creating Verrazzano Internal ES User
	err = createUser(ctx, cfg, cli, vzInternalEsUser, "verrazzano-es-internal", vzSystemGroup)
	if err != nil {
		return err
	}

	// Create verrazzano-pkce client
	err = createOrUpdateVerrazzanoPkceClient(ctx, cfg, cli)
	if err != nil {
		return err
	}

	// Creating verrazzano-pg client
	err = createVerrazzanoPgClient(ctx, cfg, cli)
	if err != nil {
		return err
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
		ctx.Log().Errorf("Component Keycloak failed logging into Keycloak: stdout = %s: stderr = %s", stdOut, stdErr)
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

func keycloakPod() *v1.Pod {
	return &v1.Pod{
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

func createVerrazzanoUsersGroup(ctx spi.ComponentContext) (string, error) {
	keycloakGroups, err := getKeycloakGroups(ctx)
	if err == nil && groupExists(keycloakGroups, vzUsersGroup) {
		// Group already exists
		return getGroupID(keycloakGroups, vzUsersGroup), nil
	}

	userGroup := "name=" + vzUsersGroup
	cmd := execCommand("kubectl", "exec", keycloakPodName, "-n", ComponentNamespace, "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "groups", "-r", vzSysRealm, "-s", userGroup)
	ctx.Log().Debugf("createVerrazzanoUsersGroup: Create Verrazzano Users Group Cmd = %s", cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed creating Verrazzano Users Group: command output = %s", out)
		return "", err
	}
	ctx.Log().Debugf("createVerrazzanoUsersGroup: Create Verrazzano Users Group Output = %s", out)
	if len(string(out)) == 0 {
		err = errors.New("Component Keycloak failed; user group ID from Keycloak is zero length")
		ctx.Log().Error(err)
		return "", err
	}
	arr := strings.Split(string(out), "'")
	if len(arr) != 3 {
		return "", fmt.Errorf("Component Keycloak failed parsing output returned from Users Group create stdout returned = %s", out)
	}
	ctx.Log().Debugf("createVerrazzanoUsersGroup: User Group ID = %s", arr[1])
	ctx.Log().Once("Component Keycloak successfully created the Verrazzano user group")
	return arr[1], nil
}

func createVerrazzanoAdminGroup(ctx spi.ComponentContext, userGroupID string) (string, error) {
	keycloakGroups, err := getKeycloakGroups(ctx)
	if err == nil && groupExists(keycloakGroups, vzAdminGroup) {
		// Group already exists
		return getGroupID(keycloakGroups, vzAdminGroup), nil
	}
	adminGroup := "groups/" + userGroupID + "/children"
	adminGroupName := "name=" + vzAdminGroup
	cmd := execCommand("kubectl", "exec", keycloakPodName, "-n", ComponentNamespace, "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", adminGroup, "-r", vzSysRealm, "-s", adminGroupName)
	ctx.Log().Debugf("createVerrazzanoAdminGroup: Create Verrazzano Admin Group Cmd = %s", cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed creating Verrazzano Admin Group: command output = %s", out)
		return "", err
	}
	ctx.Log().Debugf("createVerrazzanoAdminGroup: Create Verrazzano Admin Group Output = %s", out)
	if len(string(out)) == 0 {
		err = errors.New("Component Keycloak failed; admin group ID from Keycloak is zero length")
		ctx.Log().Error(err)
		return "", err
	}
	arr := strings.Split(string(out), "'")
	if len(arr) != 3 {
		return "", fmt.Errorf("Component Keycloak failed parsing output returned from Admin Group create stdout returned = %s", out)
	}
	ctx.Log().Debugf("createVerrazzanoAdminGroup: Admin Group ID = %s", arr[1])
	ctx.Log().Once("Component Keycloak successfully created the Verrazzano admin group")
	return arr[1], nil
}

func createVerrazzanoMonitorsGroup(ctx spi.ComponentContext, userGroupID string) (string, error) {
	keycloakGroups, err := getKeycloakGroups(ctx)
	if err == nil && groupExists(keycloakGroups, vzMonitorGroup) {
		// Group already exists
		return getGroupID(keycloakGroups, vzMonitorGroup), nil
	}
	monitorGroup := "groups/" + userGroupID + "/children"
	monitorGroupName := "name=" + vzMonitorGroup
	cmd := execCommand("kubectl", "exec", keycloakPodName, "-n", ComponentNamespace, "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", monitorGroup, "-r", vzSysRealm, "-s", monitorGroupName)
	ctx.Log().Debugf("createVerrazzanoProjectMonitorsGroup: Create Verrazzano Monitors Group Cmd = %s", cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed creating Verrazzano Monitor Group: command output = %s", out)
		return "", err
	}
	ctx.Log().Debugf("createVerrazzanoProjectMonitorsGroup: Create Verrazzano Project Monitors Group Output = %s", out)
	if len(string(out)) == 0 {
		err = errors.New("Component Keycloak failed; monitor group ID from Keycloak is zero length")
		ctx.Log().Error(err)
		return "", err
	}
	arr := strings.Split(string(out), "'")
	if len(arr) != 3 {
		return "", fmt.Errorf("Component Keycloak failed parsing output returned from Monitor Group create stdout returned = %s", out)
	}
	ctx.Log().Debugf("createVerrazzanoProjectMonitorsGroup: Monitor Group ID = %s", arr[1])
	ctx.Log().Once("Component Keycloak successfully created the Verrazzano monitors group")

	return arr[1], nil
}

func createVerrazzanoSystemGroup(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, userGroupID string) error {

	keycloakGroups, err := getKeycloakGroups(ctx)
	if err == nil && groupExists(keycloakGroups, vzSystemGroup) {
		return nil
	}

	kcPod := keycloakPod()
	systemGroup := "groups/" + userGroupID + "/children"
	systemGroupName := "name=" + vzSystemGroup
	createVzSystemGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh create " + systemGroup + " -r " + vzSysRealm + " -s " + systemGroupName
	ctx.Log().Debugf("createVerrazzanoSystemGroup: Create Verrazzano System Group Cmd = %s", createVzSystemGroupCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createVzSystemGroupCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed creating Verrazzano System Group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Once("Component Keycloak successfully created the Verrazzano system group")
	return nil
}

func createVerrazzanoRole(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, roleName string) error {
	keycloakRoles, err := getKeycloakRoles(ctx)
	if err == nil && roleExists(keycloakRoles, roleName) {
		return nil
	}
	kcPod := keycloakPod()
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

func createUser(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, userName string, secretName string, groupName string) error {
	keycloakUsers, err := getKeycloakUsers(ctx)
	if err == nil && userExists(keycloakUsers, userName) {
		return nil
	}
	kcPod := keycloakPod()
	vzUser := "username=" + userName
	vzUserGroup := "groups[0]=/" + vzUsersGroup + "/" + groupName
	createVzUserCmd := "/opt/jboss/keycloak/bin/kcadm.sh create users -r " + vzSysRealm + " -s " + vzUser + " -s " + vzUserGroup + " -s enabled=true"
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

func createOrUpdateVerrazzanoPkceClient(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface) error {
	data := templateData{}

	keycloakClients, err := getKeycloakClients(ctx)
	if err != nil {
		return err
	}
	if clientExists(keycloakClients, "verrazzano-pkce") {
		if err := updateKeycloakUris(ctx); err != nil {
			return err
		}
		return nil
	}

	kcPod := keycloakPod()
	// Get DNS Domain Configuration
	dnsSubDomain, err := getDNSDomain(ctx.Client(), ctx.EffectiveCR())
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed retrieving DNS sub domain: %v", err)
		return err
	}
	ctx.Log().Debugf("createOrUpdateVerrazzanoPkceClient: DNSDomain returned %s", dnsSubDomain)
	cr := ctx.EffectiveCR()
	ingressType, err := vzconfig.GetServiceType(cr)
	if err != nil {
		return nil
	}
	switch ingressType {
	case vzapi.NodePort:
		for _, ports := range cr.Spec.Components.Ingress.Ports {
			if ports.Port == 443 {
				dnsSubDomain = fmt.Sprintf("%s:%s", dnsSubDomain, strconv.Itoa(int(ports.NodePort)))
			}
		}
	}

	data.DNSSubDomain = dnsSubDomain

	// use template to get populate template with data
	var b bytes.Buffer
	t, err := template.New("verrazzanoPkceClient").Parse(pkceTmpl)
	if err != nil {
		return err
	}
	err = t.Execute(&b, &data)
	if err != nil {
		return err
	}

	// Create verrazzano-pkce client
	vzPkceCreateCmd := "/opt/jboss/keycloak/bin/kcadm.sh create clients -r " + vzSysRealm + " -f - <<\\END" +
		b.String() +
		"END"

	ctx.Log().Debugf("createOrUpdateVerrazzanoPkceClient: Create verrazzano-pkce client Cmd = %s", vzPkceCreateCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(vzPkceCreateCmd))
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed creating verrazzano-pkce client: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("createOrUpdateVerrazzanoPkceClient: Created verrazzano-pkce client")
	return nil
}

func createVerrazzanoPgClient(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface) error {
	keycloakClients, err := getKeycloakClients(ctx)
	if err == nil && clientExists(keycloakClients, "verrazzano-pg") {
		return nil
	}

	kcPod := keycloakPod()
	vzPgCreateCmd := "/opt/jboss/keycloak/bin/kcadm.sh create clients -r " + vzSysRealm + " -f - <<\\END" +
		pgClient +
		"END"
	ctx.Log().Debugf("createVerrazzanoPgClient: Create verrazzano-pg client Cmd = %s", vzPgCreateCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(vzPgCreateCmd))
	if err != nil {
		ctx.Log().Errorf("createVerrazzanoPgClient: Error creating verrazzano-pg client: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("createVerrazzanoPgClient: Created verrazzano-pg client")
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
func getKeycloakGroups(ctx spi.ComponentContext) (KeycloakGroups, error) {
	var keycloakGroups KeycloakGroups
	// Get the Client ID JSON array
	cmd := execCommand("kubectl", "exec", keycloakPodName, "-n", ComponentNamespace, "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "get", "groups", "-r", vzSysRealm)
	out, err := cmd.Output()
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed retrieving Groups: %s", err)
		return nil, err
	}
	if len(string(out)) == 0 {
		err = errors.New("Component Keycloak failed; groups JSON from Keycloak is zero length")
		ctx.Log().Error(err)
		return nil, err
	}
	err = json.Unmarshal(out, &keycloakGroups)
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
func getKeycloakRoles(ctx spi.ComponentContext) (KeycloakRoles, error) {
	var keycloakRoles KeycloakRoles
	// Get the Client ID JSON array
	cmd := execCommand("kubectl", "exec", keycloakPodName, "-n", ComponentNamespace, "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "get-roles", "-r", vzSysRealm)
	out, err := cmd.Output()
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed retrieving Roles: %s", err)
		return nil, err
	}
	if len(string(out)) == 0 {
		err = errors.New("Component Keycloak failed; roles JSON from Keycloak is zero length")
		ctx.Log().Error(err)
		return nil, err
	}
	err = json.Unmarshal(out, &keycloakRoles)
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
func getKeycloakUsers(ctx spi.ComponentContext) (KeycloakUsers, error) {
	var keycloakUsers KeycloakUsers
	// Get the Client ID JSON array
	cmd := execCommand("kubectl", "exec", keycloakPodName, "-n", ComponentNamespace, "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "get", "users", "-r", vzSysRealm)
	out, err := cmd.Output()
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed retrieving Users: %s", err)
		return nil, err
	}
	if len(string(out)) == 0 {
		err := errors.New("Component Keycloak failed; users JSON from Keycloak is zero length")
		ctx.Log().Error(err)
		return nil, err
	}
	err = json.Unmarshal(out, &keycloakUsers)
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed ummarshalling users json: %v", err)
		return nil, err
	}
	return keycloakUsers, nil
}

func userExists(keycloakUsers KeycloakUsers, userName string) bool {
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
	// Get the Client ID JSON array
	cmd := execCommand("kubectl", "exec", keycloakPodName, "-n", ComponentNamespace, "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "get", "clients", "-r", "verrazzano-system", "--fields", "id,clientId")
	out, err := cmd.Output()
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed retrieving clients: %s", err)
		return nil, err
	}
	if len(string(out)) == 0 {
		err := errors.New("Component Keycloak failed; clients JSON from Keycloak is zero length")
		ctx.Log().Error(err)
		return nil, err
	}
	err = json.Unmarshal(out, &keycloakClients)
	if err != nil {
		ctx.Log().Errorf("Component Keycloak failed ummarshalling client json: %v", err)
		return nil, err
	}
	return keycloakClients, nil
}

func clientExists(keycloakClients KeycloakClients, clientName string) bool {

	for _, keycloakClient := range keycloakClients {
		if keycloakClient.ClientID == clientName {
			return true
		}
	}
	return false
}

func getClientID(keycloakClients KeycloakClients, clientName string) string {

	for _, keycloakClient := range keycloakClients {
		if keycloakClient.ClientID == clientName {
			return keycloakClient.ID
		}
	}
	return ""
}

func isKeycloakReady(ctx spi.ComponentContext) bool {
	statefulset := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.StatefulSetsAreReady(ctx.Log(), ctx.Client(), statefulset, 1, prefix)
}

// isPodReady determines if the pod is running by checking for a Ready condition with Status equal True
func isPodReady(pod *v1.Pod) bool {
	conditions := pod.Status.Conditions
	for j := range conditions {
		if conditions[j].Type == "Ready" && conditions[j].Status == "True" {
			return true
		}
	}
	return false
}
