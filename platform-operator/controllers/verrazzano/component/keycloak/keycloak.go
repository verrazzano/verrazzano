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
	"text/template"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzpassword "github.com/verrazzano/verrazzano/pkg/security/password"

	"strings"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

const (
	dnsTarget          = "dnsTarget"
	rulesHost          = "rulesHost"
	tlsHosts           = "tlsHosts"
	tlsSecret          = "tlsSecret"
	vzSysRealm         = "verrazzano-system"
	vzUsersGroup       = "verrazzano-users"
	vzAdminGroup       = "verrazzano-admins"
	vzMonitorGroup     = "verrazzano-monitors"
	vzSystemGroup      = "verrazzano-system-users"
	vzAPIAccessRole    = "vz_api_access"
	vzConsoleUsersRole = "console_users"
	vzAdminRole        = "Admin"
	vzViewerRole       = "Viewer"
	vzUserName         = "verrazzano"
	vzInternalPromUser = "verrazzano-prom-internal"
	vzInternalEsUser   = "verrazzano-es-internal"
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
		return nil, fmt.Errorf("expected 1 image for Keycloak theme, found %v", len(images))
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
		compContext.Log().Errorf("AppendKeycloakOverrides: Error retrieving DNS sub domain: %s", err)
		return nil, err
	}
	compContext.Log().Infof("AppendKeycloakOverrides: DNSDomain returned %s", dnsSubDomain)

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
	installEnvName := getEnvironmentName(compContext.EffectiveCR().Spec.EnvironmentName)
	tlsSecretValue := fmt.Sprintf("%s-secret", installEnvName)
	kvs = append(kvs, bom.KeyValue{
		Key:   tlsSecret,
		Value: tlsSecretValue,
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

// updateKeycloakIngress updates the Ingress when using externalDNS
func updateKeycloakIngress(ctx spi.ComponentContext) error {
	ingress := networkv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "keycloak", Namespace: "keycloak"},
	}
	opResult, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &ingress, func() error {
		dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
		if err != nil {
			return err
		}
		ingressTarget := fmt.Sprintf("verrazzano-ingress.%s", dnsSubDomain)
		ctx.Log().Infof("updateKeycloakIngress: Updating Keycloak Ingress with ingressTarget = %s", ingressTarget)
		ingress.Annotations["external-dns.alpha.kubernetes.io/target"] = ingressTarget
		return nil
	})
	ctx.Log().Infof("updateKeycloakIngress: Keycloak ingress operation result: %s", opResult)
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
		return errors.New("Keycloak Post Upgrade: Error retrieving ID for Keycloak user, zero length")
	}
	ctx.Log().Info("Keycloak Post Upgrade: Successfully retrieved clientID")

	// Get DNS Domain Configuration
	dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
	if err != nil {
		ctx.Log().Errorf("Keycloak Post Upgrade: Error retrieving DNS sub domain: %s", err)
		return err
	}
	ctx.Log().Infof("Keycloak Post Upgrade: DNSDomain returned %s", dnsSubDomain)

	// Call the Script and Update the URIs
	scriptName := filepath.Join(config.GetInstallDir(), "update-kiali-redirect-uris.sh")
	if _, stderr, err := bashFunc(scriptName, id, dnsSubDomain); err != nil {
		ctx.Log().Errorf("Keycloak Post Upgrade: Failed updating KeyCloak URIs %s: %s", err, stderr)
		return err
	}
	ctx.Log().Info("Keycloak Post Upgrade: Successfully Updated Keycloak URIs")
	return nil
}

// configureKeycloakRealms configures the Verrazzano system realm
func configureKeycloakRealms(ctx spi.ComponentContext) error {
	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return err
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
		return errors.New("configureKeycloakRealms: Error creating/retrieving User Group ID from Keycloak, zero length")
	}

	// Create Verrazzano Admin Group
	adminGroupID, err := createVerrazzanoAdminGroup(ctx, userGroupID)
	if err != nil {
		return err
	}
	if adminGroupID == "" {
		return errors.New("configureKeycloakRealms: Error creating/retrieving Admin Group ID from Keycloak, zero length")
	}

	// Create Verrazzano Project Monitors Group
	monitorGroupID, err := createVerrazzanoMonitorsGroup(ctx, userGroupID)
	if err != nil {
		return err
	}
	if monitorGroupID == "" {
		return errors.New("configureKeycloakRealms: Error creating/retrieving Monitor Group ID from Keycloak, zero length")
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
	err = createVerrazzanoPkceClient(ctx, cfg, cli)
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

	ctx.Log().Infof("configureKeycloakRealm: successfully configured realm %s", vzSysRealm)
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
		ctx.Log().Errorf("loginKeycloak: Error retrieving Keycloak password: %s", err)
		return err
	}
	pw := secret.Data["password"]
	keycloakpw := string(pw)
	if keycloakpw == "" {
		return errors.New("loginKeycloak: Error retrieving Keycloak password, empty string")
	}
	ctx.Log().Debug("loginKeycloak: Successfully retrieved Keycloak password")

	// Login to Keycloak
	kcPod := keycloakPod()
	loginCmd := "/opt/jboss/keycloak/bin/kcadm.sh config credentials --server http://localhost:8080/auth --realm master --user keycloakadmin --password " + keycloakpw
	ctx.Log().Debugf("loginKeycloak: Login Cmd = %s", maskPw(loginCmd))
	stdOut, stdErr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(loginCmd))
	if err != nil {
		ctx.Log().Errorf("loginKeycloak: Error retrieving logging into Keycloak: stdout = %s: stderr = %s", stdOut, stdErr)
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	ctx.Log().Debug("loginKeycloak: Successfully logged into Keycloak")

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
			Name:      "keycloak-0",
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
		opResult, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), secret, func() error {
			// Build the secret data
			secret.Data = map[string][]byte{
				"username": []byte(username),
				"password": []byte(pw),
			}
			return nil
		})
		ctx.Log().Infof("Keycloak secret operation result: %s", opResult)

		if err != nil {
			return err
		}
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
		ctx.Log().Errorf("getSecretPassword: Error retrieving secret %s password: %s", secretname, err)
		return "", err
	}
	pw := secret.Data["password"]
	stringpw := string(pw)
	if stringpw == "" {
		return "", fmt.Errorf("getSecretPassword: Error retrieving secret %s password", secretname)
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

// getSecretName returns expected TLS secret name
func getSecretName(vz *vzapi.Verrazzano) string {
	return fmt.Sprintf("%s-secret", getEnvironmentName(vz.Spec.EnvironmentName))
}

func createVerrazzanoSystemRealm(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface) error {

	kcPod := keycloakPod()
	realm := "realm=" + vzSysRealm
	checkRealmExistsCmd := "/opt/jboss/keycloak/bin/kcadm.sh get realms/" + vzSysRealm
	ctx.Log().Debugf("createVerrazzanoSystemRealm: Check Verrazzano System Realm Exists Cmd = %s", checkRealmExistsCmd)
	_, _, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(checkRealmExistsCmd))
	if err != nil {
		ctx.Log().Info("createVerrazzanoSystemRealm: Verrazzano System Realm doesn't exist: Creating it")
		createRealmCmd := "/opt/jboss/keycloak/bin/kcadm.sh create realms -s " + realm + " -s enabled=false"
		ctx.Log().Debugf("createVerrazzanoSystemRealm: Create Verrazzano System Realm Cmd = %s", createRealmCmd)
		stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createRealmCmd))
		if err != nil {
			ctx.Log().Errorf("createVerrazzanoSystemRealm: Error creating Verrazzano System Realm: stdout = %s, stderr = %s", stdout, stderr)
			return err
		}
	}
	ctx.Log().Debug("createVerrazzanoSystemRealm: Successfully Created Verrazzano System Realm")
	return nil
}

func createVerrazzanoUsersGroup(ctx spi.ComponentContext) (string, error) {
	keycloakGroups, err := getKeycloakGroups(ctx)
	if err == nil && groupExists(keycloakGroups, vzUsersGroup) {
		// Group already exists
		return getGroupID(keycloakGroups, vzUsersGroup), nil
	}

	userGroup := "name=" + vzUsersGroup
	cmd := execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "groups", "-r", vzSysRealm, "-s", userGroup)
	ctx.Log().Debugf("createVerrazzanoUsersGroup: Create Verrazzano Users Group Cmd = %s", cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		ctx.Log().Errorf("createVerrazzanoUsersGroup: Error creating Verrazzano Users Group: command output = %s", out)
		return "", err
	}
	ctx.Log().Debugf("createVerrazzanoUsersGroup: Create Verrazzano Users Group Output = %s", out)
	if len(string(out)) == 0 {
		return "", errors.New("createVerrazzanoUsersGroup: Error retrieving User Group ID from Keycloak, zero length")
	}
	arr := strings.Split(string(out), "'")
	if len(arr) != 3 {
		return "", fmt.Errorf("createVerrazzanoUsersGroup: Error parsing output returned from Users Group create stdout returned = %s", out)
	}
	ctx.Log().Debugf("createVerrazzanoUsersGroup: User Group ID = %s", arr[1])
	ctx.Log().Debug("createVerrazzanoUsersGroup: Successfully Created Verrazzano User Group")
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
	cmd := execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", adminGroup, "-r", vzSysRealm, "-s", adminGroupName)
	ctx.Log().Debugf("createVerrazzanoAdminGroup: Create Verrazzano Admin Group Cmd = %s", cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		ctx.Log().Errorf("createVerrazzanoAdminGroup: Error creating Verrazzano Admin Group: command output = %s", out)
		return "", err
	}
	ctx.Log().Debugf("createVerrazzanoAdminGroup: Create Verrazzano Admin Group Output = %s", out)
	if len(string(out)) == 0 {
		return "", errors.New("createVerrazzanoAdminGroup: Error retrieving Admin Group ID from Keycloak, zero length")
	}
	arr := strings.Split(string(out), "'")
	if len(arr) != 3 {
		return "", fmt.Errorf("createVerrazzanoAdminGroup: Error parsing output returned from Admin Group create stdout returned = %s", out)
	}
	ctx.Log().Debugf("createVerrazzanoAdminGroup: Admin Group ID = %s", arr[1])
	ctx.Log().Debug("createVerrazzanoAdminGroup: Successfully Created Verrazzano Admin Group")
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
	cmd := execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", monitorGroup, "-r", vzSysRealm, "-s", monitorGroupName)
	ctx.Log().Debugf("createVerrazzanoProjectMonitorsGroup: Create Verrazzano Monitors Group Cmd = %s", cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		ctx.Log().Errorf("ccreateVerrazzanoProjectMonitorsGroup: Error creating Verrazzano Monitor Group: command output = %s", out)
		return "", err
	}
	ctx.Log().Debugf("createVerrazzanoProjectMonitorsGroup: Create Verrazzano Project Monitors Group Output = %s", out)
	if len(string(out)) == 0 {
		return "", errors.New("createVerrazzanoProjectMonitorsGroup: Error retrieving Monitor Group ID from Keycloak, zero length")
	}
	arr := strings.Split(string(out), "'")
	if len(arr) != 3 {
		return "", fmt.Errorf("createVerrazzanoProjectMonitorsGroup: Error parsing output returned from Monitor Group create stdout returned = %s", out)
	}
	ctx.Log().Debugf("createVerrazzanoProjectMonitorsGroup: Monitor Group ID = %s", arr[1])
	ctx.Log().Debug("createVerrazzanoProjectMonitorsGroup: Successfully Created Verrazzano Monitors Group")
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
		ctx.Log().Errorf("createVerrazzanoSystemGroup: Error creating Verrazzano System Group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("createVerrazzanoSystemGroup: Successfully Created Verrazzano System Group")
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
		ctx.Log().Errorf("createVerrazzanoRole: Error creating Verrazzano API Access Role: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("createVerrazzanoRole: Successfully Created Verrazzano API Access Role")
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
		ctx.Log().Errorf("grantRolesToGroups: Error granting api access role to Verrazzano users group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("grantRolesToGroups: Granted Access Role to User Group")

	// Granting console_users role to verrazzano users group
	grantConsoleRoleToVzUserGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + vzSysRealm + " --gid " + userGroupID + " --rolename " + vzConsoleUsersRole
	ctx.Log().Debugf("grantRolesToGroups: Grant Console Role to Vz Users Cmd = %s", grantConsoleRoleToVzUserGroupCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(grantConsoleRoleToVzUserGroupCmd))
	if err != nil {
		ctx.Log().Errorf("grantRolesToGroups: Error granting console users role to Verrazzano users group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("grantRolesToGroups: Granted Console Role to User Group")

	// Granting admin role to verrazzano admin group
	grantAdminRoleToVzAdminGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + vzSysRealm + " --gid " + adminGroupID + " --rolename " + vzAdminRole
	ctx.Log().Debugf("grantRolesToGroups: Grant Admin Role to Vz Admin Cmd = %s", grantAdminRoleToVzAdminGroupCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(grantAdminRoleToVzAdminGroupCmd))
	if err != nil {
		ctx.Log().Errorf("grantRolesToGroups: Error granting admin role to Verrazzano admin group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("grantRolesToGroups: Granted Admin Role to Admin Group")

	// Granting viewer role to verrazzano monitor group
	grantViewerRoleToVzMonitorGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + vzSysRealm + " --gid " + monitorGroupID + " --rolename " + vzViewerRole
	ctx.Log().Debugf("grantRolesToGroups: Grant Viewer Role to Monitor Group Cmd = %s", grantViewerRoleToVzMonitorGroupCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(grantViewerRoleToVzMonitorGroupCmd))
	if err != nil {
		ctx.Log().Errorf("grantRolesToGroups: Error granting viewer role to Verrazzano monitoring group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("grantRolesToGroups: Granted Viewer Role to monitor Group")

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
		ctx.Log().Errorf("createUser: Error creating Verrazzano user: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debugf("createUser: Successfully Created VZ User %s", userName)

	vzpw, err := getSecretPassword(ctx, "verrazzano-system", secretName)
	if err != nil {
		ctx.Log().Errorf("createUser: Error retrieving Verrazzano password: %s", err)
		return err
	}
	setVZUserPwCmd := "/opt/jboss/keycloak/bin/kcadm.sh set-password -r " + vzSysRealm + " --username " + userName + " --new-password " + vzpw
	ctx.Log().Debugf("createUser: Set Verrazzano User PW Cmd = %s", maskPw(setVZUserPwCmd))
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setVZUserPwCmd))
	if err != nil {
		ctx.Log().Errorf("createUser: Error setting Verrazzano user password: stdout = %s, stderr = %s", stdout, stderr)
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	ctx.Log().Debugf("createUser: Created VZ User %s PW", userName)
	return nil
}

func createVerrazzanoPkceClient(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface) error {
	keycloakClients, err := getKeycloakClients(ctx)
	if err == nil && clientExists(keycloakClients, "verrazzano-pkce") {
		return nil
	}

	kcPod := keycloakPod()
	// Get DNS Domain Configuration
	dnsSubDomain, err := getDNSDomain(ctx.Client(), ctx.EffectiveCR())
	if err != nil {
		ctx.Log().Errorf("createVerrazzanoPkceClient: Error retrieving DNS sub domain: %s", err)
		return err
	}
	ctx.Log().Infof("createVerrazzanoPkceClient: DNSDomain returned %s", dnsSubDomain)

	// Create verrazzano-pkce client
	vzPkceCreateCmd := "/opt/jboss/keycloak/bin/kcadm.sh create clients -r " + vzSysRealm + " -f - <<\\END\n" +
		"{\n      " +
		"\"clientId\" : \"verrazzano-pkce\",\n     " +
		"\"enabled\": true,\n      \"surrogateAuthRequired\": false,\n      " +
		"\"alwaysDisplayInConsole\": false,\n      " +
		"\"clientAuthenticatorType\": \"client-secret\",\n" +
		"      \"redirectUris\": [\n" +
		"        \"https://verrazzano." + dnsSubDomain + "/*\",\n" +
		"        \"https://verrazzano." + dnsSubDomain + "/verrazzano/authcallback\",\n" +
		"        \"https://elasticsearch.vmi.system." + dnsSubDomain + "/*\",\n" +
		"        \"https://elasticsearch.vmi.system." + dnsSubDomain + "/_authentication_callback\",\n" +
		"        \"https://prometheus.vmi.system." + dnsSubDomain + "/*\",\n" +
		"        \"https://prometheus.vmi.system." + dnsSubDomain + "/_authentication_callback\",\n" +
		"        \"https://grafana.vmi.system." + dnsSubDomain + "/*\",\n" +
		"        \"https://grafana.vmi.system." + dnsSubDomain + "/_authentication_callback\",\n" +
		"        \"https://kibana.vmi.system." + dnsSubDomain + "/*\",\n" +
		"        \"https://kibana.vmi.system." + dnsSubDomain + "/_authentication_callback\",\n" +
		"        \"https://kiali.vmi.system." + dnsSubDomain + "/*\",\n" +
		"        \"https://kiali.vmi.system." + dnsSubDomain + "/_authentication_callback\"\n" +
		"      ],\n" +
		"      \"webOrigins\": [\n" +
		"        \"https://verrazzano." + dnsSubDomain + "\",\n" +
		"        \"https://elasticsearch.vmi.system." + dnsSubDomain + "\",\n" +
		"        \"https://prometheus.vmi.system." + dnsSubDomain + "\",\n" +
		"        \"https://grafana.vmi.system." + dnsSubDomain + "\",\n" +
		"        \"https://kibana.vmi.system." + dnsSubDomain + "\",\n" +
		"        \"https://kiali.vmi.system." + dnsSubDomain + "\"\n" +
		"      ],\n" +
		"      \"notBefore\": 0,\n" +
		"      \"bearerOnly\": false,\n" +
		"      \"consentRequired\": false,\n" +
		"      \"standardFlowEnabled\": true,\n" +
		"      \"implicitFlowEnabled\": false,\n" +
		"      \"directAccessGrantsEnabled\": false,\n" +
		"      \"serviceAccountsEnabled\": false,\n" +
		"      \"publicClient\": true,\n" +
		"      \"frontchannelLogout\": false,\n" +
		"      \"protocol\": \"openid-connect\",\n" +
		"      \"attributes\": {\n" +
		"        \"saml.assertion.signature\": \"false\",\n" +
		"        \"saml.multivalued.roles\": \"false\",\n" +
		"        \"saml.force.post.binding\": \"false\",\n" +
		"        \"saml.encrypt\": \"false\",\n" +
		"        \"saml.server.signature\": \"false\",\n" +
		"        \"saml.server.signature.keyinfo.ext\": \"false\",\n" +
		"        \"exclude.session.state.from.auth.response\": \"false\",\n" +
		"        \"saml_force_name_id_format\": \"false\",\n" +
		"        \"saml.client.signature\": \"false\",\n" +
		"        \"tls.client.certificate.bound.access.tokens\": \"false\",\n" +
		"        \"saml.authnstatement\": \"false\",\n" +
		"        \"display.on.consent.screen\": \"false\",\n" +
		"        \"pkce.code.challenge.method\": \"S256\",\n" +
		"        \"saml.onetimeuse.condition\": \"false\"\n" +
		"      },\n" +
		"      \"authenticationFlowBindingOverrides\": {},\n" +
		"      \"fullScopeAllowed\": true,\n" +
		"      \"nodeReRegistrationTimeout\": -1,\n" +
		"      \"protocolMappers\": [\n" +
		"          {\n" +
		"            \"name\": \"groupmember\",\n" +
		"            \"protocol\": \"openid-connect\",\n" +
		"            \"protocolMapper\": \"oidc-group-membership-mapper\",\n" +
		"            \"consentRequired\": false,\n" +
		"            \"config\": {\n" +
		"              \"full.path\": \"false\",\n" +
		"              \"id.token.claim\": \"true\",\n" +
		"              \"access.token.claim\": \"true\",\n" +
		"              \"claim.name\": \"groups\",\n" +
		"              \"userinfo.token.claim\": \"true\"\n" +
		"            }\n" +
		"          },\n" +
		"          {\n" +
		"            \"name\": \"realm roles\",\n" +
		"            \"protocol\": \"openid-connect\",\n" +
		"            \"protocolMapper\": \"oidc-usermodel-realm-role-mapper\",\n" +
		"            \"consentRequired\": false,\n" +
		"            \"config\": {\n" +
		"              \"multivalued\": \"true\",\n" +
		"              \"user.attribute\": \"foo\",\n" +
		"              \"id.token.claim\": \"true\",\n" +
		"              \"access.token.claim\": \"true\",\n" +
		"              \"claim.name\": \"realm_access.roles\",\n" +
		"              \"jsonType.label\": \"String\"\n" +
		"            }\n" +
		"          }\n" +
		"        ],\n" +
		"      \"defaultClientScopes\": [\n" +
		"        \"web-origins\",\n" +
		"        \"role_list\",\n" +
		"        \"roles\",\n" +
		"        \"profile\",\n" +
		"        \"email\"\n" +
		"      ],\n" +
		"      \"optionalClientScopes\": [\n" +
		"        \"address\",\n" +
		"        \"phone\",\n" +
		"        \"offline_access\",\n" +
		"        \"microprofile-jwt\"\n" +
		"      ]\n" +
		"}\n" +
		"END"

	ctx.Log().Debugf("createVerrazzanoPkceClient: Create verrazzano-pkce client Cmd = %s", vzPkceCreateCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(vzPkceCreateCmd))
	if err != nil {
		ctx.Log().Errorf("createVerrazzanoPkceClient: Error creating verrazzano-pkce client: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("createVerrazzanoPkceClient: Created verrazzano-pkce client")
	return nil
}

func createVerrazzanoPgClient(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface) error {
	keycloakClients, err := getKeycloakClients(ctx)
	if err == nil && clientExists(keycloakClients, "verrazzano-pg") {
		return nil
	}

	kcPod := keycloakPod()
	vzPgCreateCmd := "/opt/jboss/keycloak/bin/kcadm.sh create clients -r " + vzSysRealm + " -f - <<\\END\n" +
		"{\n" +
		"      \"clientId\" : \"verrazzano-pg\",\n" +
		"      \"enabled\" : true,\n" +
		"      \"rootUrl\" : \"\",\n" +
		"      \"adminUrl\" : \"\",\n" +
		"      \"surrogateAuthRequired\" : false,\n" +
		"      \"directAccessGrantsEnabled\" : \"true\",\n" +
		"      \"clientAuthenticatorType\" : \"client-secret\",\n" +
		"      \"secret\" : \"de05ccdc-67df-47f3-81f6-37e61d195aba\",\n" +
		"      \"redirectUris\" : [ ],\n" +
		"      \"webOrigins\" : [ \"+\" ],\n" +
		"      \"notBefore\" : 0,\n" +
		"      \"bearerOnly\" : false,\n" +
		"      \"consentRequired\" : false,\n" +
		"      \"standardFlowEnabled\" : false,\n" +
		"      \"implicitFlowEnabled\" : false,\n" +
		"      \"directAccessGrantsEnabled\" : true,\n" +
		"      \"serviceAccountsEnabled\" : false,\n" +
		"      \"publicClient\" : true,\n" +
		"      \"frontchannelLogout\" : false,\n" +
		"      \"protocol\" : \"openid-connect\",\n" +
		"      \"attributes\" : { },\n" +
		"      \"authenticationFlowBindingOverrides\" : { },\n" +
		"      \"fullScopeAllowed\" : true,\n" +
		"      \"nodeReRegistrationTimeout\" : -1,\n" +
		"      \"protocolMappers\" : [ {\n" +
		"        \"name\" : \"groups\",\n" +
		"        \"protocol\" : \"openid-connect\",\n" +
		"        \"protocolMapper\" : \"oidc-group-membership-mapper\",\n" +
		"        \"consentRequired\" : false,\n" +
		"        \"config\" : {\n" +
		"          \"multivalued\" : \"true\",\n" +
		"          \"userinfo.token.claim\" : \"false\",\n" +
		"          \"id.token.claim\" : \"true\",\n" +
		"          \"access.token.claim\" : \"true\",\n" +
		"          \"claim.name\" : \"groups\",\n" +
		"          \"jsonType.label\" : \"String\"\n" +
		"        }\n" +
		"      }, {\n" +
		"        \"name\": \"realm roles\",\n" +
		"        \"protocol\": \"openid-connect\",\n" +
		"        \"protocolMapper\": \"oidc-usermodel-realm-role-mapper\",\n" +
		"        \"consentRequired\": false,\n" +
		"        \"config\": {\n" +
		"          \"multivalued\": \"true\",\n" +
		"          \"user.attribute\": \"foo\",\n" +
		"          \"id.token.claim\": \"true\",\n" +
		"          \"access.token.claim\": \"true\",\n" +
		"          \"claim.name\": \"realm_access.roles\",\n" +
		"          \"jsonType.label\": \"String\"\n" +
		"        }\n" +
		"      }, {\n" +
		"        \"name\" : \"Client ID\",\n" +
		"        \"protocol\" : \"openid-connect\",\n" +
		"        \"protocolMapper\" : \"oidc-usersessionmodel-note-mapper\",\n" +
		"        \"consentRequired\" : false,\n" +
		"        \"config\" : {\n" +
		"          \"user.session.note\" : \"clientId\",\n" +
		"          \"userinfo.token.claim\" : \"true\",\n" +
		"          \"id.token.claim\" : \"true\",\n" +
		"          \"access.token.claim\" : \"true\",\n" +
		"          \"claim.name\" : \"clientId\",\n" +
		"          \"jsonType.label\" : \"String\"\n" +
		"        }\n" +
		"      }, {\n" +
		"        \"name\" : \"Client IP Address\",\n" +
		"        \"protocol\" : \"openid-connect\",\n" +
		"        \"protocolMapper\" : \"oidc-usersessionmodel-note-mapper\",\n" +
		"        \"consentRequired\" : false,\n" +
		"        \"config\" : {\n" +
		"          \"user.session.note\" : \"clientAddress\",\n" +
		"          \"userinfo.token.claim\" : \"true\",\n" +
		"          \"id.token.claim\" : \"true\",\n" +
		"          \"access.token.claim\" : \"true\",\n" +
		"          \"claim.name\" : \"clientAddress\",\n" +
		"          \"jsonType.label\" : \"String\"\n" +
		"        }\n" +
		"      }, {\n" +
		"        \"name\" : \"Client Host\",\n" +
		"        \"protocol\" : \"openid-connect\",\n" +
		"        \"protocolMapper\" : \"oidc-usersessionmodel-note-mapper\",\n" +
		"        \"consentRequired\" : false,\n" +
		"        \"config\" : {\n" +
		"          \"user.session.note\" : \"clientHost\",\n" +
		"          \"userinfo.token.claim\" : \"true\",\n" +
		"          \"id.token.claim\" : \"true\",\n" +
		"          \"access.token.claim\" : \"true\",\n" +
		"          \"claim.name\" : \"clientHost\",\n" +
		"          \"jsonType.label\" : \"String\"\n" +
		"        }\n" +
		"      } ],\n" +
		"      \"defaultClientScopes\" : [ \"web-origins\", \"role_list\", \"roles\", \"profile\", \"email\" ],\n" +
		"      \"optionalClientScopes\" : [ \"address\", \"phone\", \"offline_access\", \"microprofile-jwt\" ]\n" +
		"}\n" +
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
		ctx.Log().Errorf("setPasswordPolicyForRealm: Error Setting password policy for master: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debugf("setPasswordPolicyForRealm: Set password policy for realm %s", realmName)
	return nil
}

func configureLoginThemeForRealm(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface, realmName string, loginTheme string) error {
	kcPod := keycloakPod()
	setLoginThemeCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/" + realmName + " -s loginTheme=" + loginTheme
	ctx.Log().Debugf("configureLoginThemeForRealm: Configuring login theme Cmd = %s", setLoginThemeCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setLoginThemeCmd))
	if err != nil {
		ctx.Log().Errorf("configureLoginThemeForRealm: Error Configuring login theme for master: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureLoginThemeForRealm: Configured login theme for master Cmd")
	return nil
}

func enableVerrazzanoSystemRealm(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface) error {
	kcPod := keycloakPod()
	setVzEnableRealmCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/" + vzSysRealm + " -s enabled=true"
	ctx.Log().Debugf("enableVerrazzanoSystemRealm: Enabling vzSysRealm realm Cmd = %s", setVzEnableRealmCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setVzEnableRealmCmd))
	if err != nil {
		ctx.Log().Errorf("enableVerrazzanoSystemRealm: Error Enabling vzSysRealm realm: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("enableVerrazzanoSystemRealm: Enabled vzSysRealm realm")
	return nil
}

func removeLoginConfigFile(ctx spi.ComponentContext, cfg *restclient.Config, cli kubernetes.Interface) error {
	kcPod := keycloakPod()
	removeLoginConfigFileCmd := "rm /root/.keycloak/kcadm.config"
	ctx.Log().Debugf("removeLoginConfigFile: Removing login config file Cmd = %s", removeLoginConfigFileCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(removeLoginConfigFileCmd))
	if err != nil {
		ctx.Log().Errorf("removeLoginConfigFile: Error Removing login config file: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("removeLoginConfigFile: Removed login config file")
	return nil
}

// getKeycloakGroups returns a structure of Groups in Realm verrazzano-system
func getKeycloakGroups(ctx spi.ComponentContext) (KeycloakGroups, error) {
	var keycloakGroups KeycloakGroups
	// Get the Client ID JSON array
	cmd := execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "get", "groups", "-r", vzSysRealm)
	out, err := cmd.Output()
	if err != nil {
		ctx.Log().Errorf("getKeycloakGroups: Error retrieving Groups: %s", err)
		return nil, err
	}
	if len(string(out)) == 0 {
		return nil, errors.New("getKeycloakGroups: Error retrieving Groups JSON from Keycloak, zero length")
	}
	err = json.Unmarshal(out, &keycloakGroups)
	if err != nil {
		ctx.Log().Errorf("getKeycloakGroups: Error ummarshalling groups json: %s", err)
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
	cmd := execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "get-roles", "-r", vzSysRealm)
	out, err := cmd.Output()
	if err != nil {
		ctx.Log().Errorf("getKeycloakRoles: Error retrieving Roles: %s", err)
		return nil, err
	}
	if len(string(out)) == 0 {
		return nil, errors.New("getKeycloakRoles: Error retrieving Roles JSON from Keycloak, zero length")
	}
	err = json.Unmarshal(out, &keycloakRoles)
	if err != nil {
		ctx.Log().Errorf("getKeycloakGroups: Error ummarshalling groups json: %s", err)
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
	cmd := execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "get", "users", "-r", vzSysRealm)
	out, err := cmd.Output()
	if err != nil {
		ctx.Log().Errorf("getKeycloakUsers: Error retrieving Users: %s", err)
		return nil, err
	}
	if len(string(out)) == 0 {
		return nil, errors.New("getKeycloakUsers: Error retrieving Users JSON from Keycloak, zero length")
	}
	err = json.Unmarshal(out, &keycloakUsers)
	if err != nil {
		ctx.Log().Errorf("getKeycloakUsers: Error ummarshalling users json: %s", err)
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
	cmd := execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "get", "clients", "-r", "verrazzano-system", "--fields", "id,clientId")
	out, err := cmd.Output()
	if err != nil {
		ctx.Log().Errorf("getKeycloakClients: Error retrieving clients: %s", err)
		return nil, err
	}
	if len(string(out)) == 0 {
		return nil, errors.New("getKeycloakClients: Error retrieving Clients JSON from Keycloak, zero length")
	}
	err = json.Unmarshal(out, &keycloakClients)
	if err != nil {
		ctx.Log().Errorf("getKeycloakClients: Error ummarshalling client json: %s", err)
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
