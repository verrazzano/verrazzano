// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"os"
	"os/exec"
	"path/filepath"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"text/template"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	dnsTarget          = "dnsTarget"
	rulesHost          = "rulesHost"
	tlsHosts           = "tlsHosts"
	tlsSecret          = "tlsSecret"
	vzSysRealm         = "verrazzano-system"
	vzUsersGroup       = "verrazzano-users"
	vzAdminGroup       = "verrazzano-admins"
	vzMonitorGroup     = "verrazzano-project-monitors"
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

var realmCreated bool = false

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
		return nil, fmt.Errorf("Expected 1 image for Keycloak theme, found %v", len(images))
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
		compContext.Log().Errorf("configureKeycloakRealms: Error retrieving DNS sub domain: %s", err)
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

func updateKeycloakUris(ctx spi.ComponentContext) error {
	var keycloakClients KeycloakClients
	cfg, cli, err := k8sutil.RESTClientConfig()
	if err != nil {
		return err
	}

	err = loginKeycloak(ctx, cfg, cli)
	if err != nil {
		return err
	}

	// Get the Client ID JSON array
	cmd := execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "get", "clients", "-r", "verrazzano-system", "--fields", "id,clientId")
	out, err := cmd.Output()
	if err != nil {
		ctx.Log().Errorf("Keycloak Post Upgrade: Error retrieving ID for client ID, zero length: %s", err)
		return err
	}
	if len(string(out)) == 0 {
		return errors.New("Keycloak Post Upgrade: Error retrieving Clients JSON from Keycloak, zero length")
	}
	err = json.Unmarshal(out, &keycloakClients)
	if err != nil {
		ctx.Log().Errorf("Keycloak Post Upgrade: Error ummarshalling client json: %s", err)
		return err
	}

	// Extract the id associated with ClientID verrazzano-pkce
	var id = ""
	for _, keycloakClient := range keycloakClients {
		if keycloakClient.ClientID == "verrazzano-pkce" {
			id = keycloakClient.ID
			ctx.Log().Debugf("Keycloak Clients ID found = %s", id)
		}
	}
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

func configureKeycloakRealms(ctx spi.ComponentContext, prompw string, espw string) error {
	cfg, cli, err := k8sutil.RESTClientConfig()
	if err != nil {
		return err
	}
	// Login to Keycloak
	err = loginKeycloak(ctx, cfg, cli)
	if err != nil {
		return err
	}

	ctx.Log().Info("CDD Successfully Logged Into Keycloak")

	// Create VerrazzanoSystem Realm
	if !realmCreated {
		realm := "realm=" + vzSysRealm
		createRealmCmd := "/opt/jboss/keycloak/bin/kcadm.sh create realms -s " + realm + " -s enabled=false"
		ctx.Log().Infof("CDD Create Verrazzano System Realm Cmd = %s", createRealmCmd)
		stdout, stderr, err := ExecCmd(cli, cfg, "keycloak-0", createRealmCmd)
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano System Realm: stdout = %s, stderr = %s", stdout, stderr)
			return err
		}
		realmCreated = true
		ctx.Log().Info("CDD Successfully Created Verrazzano System Realm")
	}

	// Create Verrazzano Users Group
	userGroup := "name=" + vzUsersGroup
	createVzUserGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh create groups -r " + vzSysRealm + " -s " + userGroup
	ctx.Log().Infof("CDD Create Verrazzano Users Group Cmd = %s", createVzUserGroupCmd)
	stdout, stderr, err := ExecCmd(cli, cfg, "keycloak-0", createVzUserGroupCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Users Group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Infof("CDD Create Verrazzano Users Group Output: stdout = %s, stderr = %s", stdout, stderr)
	if len(stdout) == 0 {
		return errors.New("configureKeycloakRealm: Error retrieving User Group ID from Keycloak, zero length")
	}
	arr := strings.Split(stdout, "'")
	userGroupID := arr[1]
	ctx.Log().Infof("configureKeycloakRealm: User Group ID = %s", userGroupID)
	ctx.Log().Info("CDD Successfully Created Verrazzano User Group")

	// Create Verrazzano Admin Group
	adminGroup := "groups/" + userGroupID + "/children"
	adminGroupName := "name=" + vzAdminGroup
	createVzAdminGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh create " + adminGroup + " -r " + vzSysRealm + " -s " + adminGroupName
	ctx.Log().Infof("CDD Create Verrazzano Admin Group Cmd = %s", createVzAdminGroupCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", createVzAdminGroupCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Admin Group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Infof("CDD Create Verrazzano Admin Group Output: stdout = %s, stderr = %s", stdout, stderr)
	if len(stdout) == 0 {
		return errors.New("configureKeycloakRealm: Error retrieving Admin Group ID from Keycloak, zero length")
	}
	arr = strings.Split(stdout, "'")
	adminGroupID := arr[1]
	ctx.Log().Infof("configureKeycloakRealm: Admin Group ID = %s", adminGroupID)
	ctx.Log().Info("CDD Successfully Created Verrazzano Admin Group")

	// Create Verrazzano Project Monitors Group
	monitorGroup := "groups/" + userGroupID + "/children"
	monitorGroupName := "name=" + vzMonitorGroup
	createVzMonitorGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh create " + monitorGroup + " -r " + vzSysRealm + " -s " + monitorGroupName
	ctx.Log().Infof("CDD Create Verrazzano Monitor Group Cmd = %s", createVzMonitorGroupCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", createVzMonitorGroupCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Monitor Group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Infof("CDD Create Verrazzano Project Monitors Group Output: stdout = %s, stderr = %s", stdout, stderr)
	if len(stdout) == 0 {
		return errors.New("configureKeycloakRealm: Error retrieving Monitor Group ID from Keycloak, zero length")
	}
	arr = strings.Split(stdout, "'")
	monitorGroupID := arr[1]
	ctx.Log().Infof("configureKeycloakRealm: Monitor Group ID = %s", monitorGroupID)
	ctx.Log().Info("CDD Successfully Created Verrazzano Monitors Group")

	// Create Verrazzano System Group
	systemGroup := "groups/" + userGroupID + "/children"
	systemGroupName := "name=" + vzSystemGroup
	createVzSystemGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh create " + systemGroup + " -r " + vzSysRealm + " -s " + systemGroupName
	ctx.Log().Infof("CDD Create Verrazzano System Group Cmd = %s", createVzSystemGroupCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", createVzSystemGroupCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano System Group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Successfully Created Verrazzano System Group")

	// Create Verrazzano API Access Role
	apiAccessRole := "name=" + vzAPIAccessRole
	createAPIAccessRoleCmd := "/opt/jboss/keycloak/bin/kcadm.sh create roles -r " + vzSysRealm + " -s " + apiAccessRole
	ctx.Log().Infof("CDD Create Verrazzano API Access Role Cmd = %s", createAPIAccessRoleCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", createAPIAccessRoleCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano API Access Role: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Successfully Created Verrazzano API Access Role")

	// Create Verrazzano Console Users Role
	consoleUserRole := "name=" + vzConsoleUsersRole
	createConsoleUserRoleCmd := "/opt/jboss/keycloak/bin/kcadm.sh create roles -r " + vzSysRealm + " -s " + consoleUserRole
	ctx.Log().Infof("CDD Create Verrazzano Console Users Role Cmd = %s", createConsoleUserRoleCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", createConsoleUserRoleCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Console Users Role: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Successfully Created Verrazzano Console User Role")

	// Create Verrazzano Admin Role
	adminRole := "name=" + vzAdminRole
	createVzAdminRoleCmd := "/opt/jboss/keycloak/bin/kcadm.sh create roles -r " + vzSysRealm + " -s " + adminRole
	ctx.Log().Infof("CDD Create Verrazzano Admin Role Cmd = %s", createVzAdminRoleCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", createVzAdminRoleCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Admin Role: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Successfully Created Verrazzano Admin Role")

	// Create Verrazzano Viewer Role
	viewerRole := "name=" + vzViewerRole
	createVzViewerRoleCmd := "/opt/jboss/keycloak/bin/kcadm.sh create roles -r " + vzSysRealm + " -s " + viewerRole
	ctx.Log().Infof("CDD Create Verrazzano Viewer Role Cmd = %s", createVzViewerRoleCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", createVzViewerRoleCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Viewer Role: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Successfully Created Verrazzano Viewer Role")

	// Granting vz_api_access role to verrazzano users group
	grantAPIAccessToVzUserGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + vzSysRealm + " --gid " + userGroupID + " --rolename " + vzAPIAccessRole
	ctx.Log().Infof("CDD Grant API Access to VZ Users Cmd = %s", grantAPIAccessToVzUserGroupCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", grantAPIAccessToVzUserGroupCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error granting api access role to Verrazzano users group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Granted Access Role to User Group")

	// Granting console_users role to verrazzano users group
	grantConsoleRoleToVzUserGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + vzSysRealm + " --gid " + userGroupID + " --rolename " + vzConsoleUsersRole
	ctx.Log().Infof("CDD Grant Console Role to Vz Users Cmd = %s", grantConsoleRoleToVzUserGroupCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", grantConsoleRoleToVzUserGroupCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error granting console users role to Verrazzano users group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Granted Console Role to User Group")

	// Granting admin role to verrazzano admin group
	grantAdminRoleToVzAdminGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + vzSysRealm + " --gid " + adminGroupID + " --rolename " + vzAdminRole
	ctx.Log().Infof("CDD Grant Admin Role to Vz Admin Cmd = %s", grantAdminRoleToVzAdminGroupCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", grantAdminRoleToVzAdminGroupCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error granting admin role to Verrazzano admin group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Granted Admin Role to Admin Group")

	// Granting viewer role to verrazzano monitor group
	grantViewerRoleToVzMonitorGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + vzSysRealm + " --gid " + monitorGroupID + " --rolename " + vzViewerRole
	ctx.Log().Infof("CDD Grant Viewer Role to Monitor Group Cmd = %s", grantViewerRoleToVzMonitorGroupCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", grantViewerRoleToVzMonitorGroupCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error granting viewer role to Verrazzano monitoring group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Granted Viewer Role to monitor Group")

	// Creating Verrazzano User
	vzUser := "username=" + vzUserName
	vzUserGroup := "groups[0]=/" + vzUsersGroup + "/" + vzAdminGroup
	createVzUserCmd := "/opt/jboss/keycloak/bin/kcadm.sh create users -r " + vzSysRealm + " -s " + vzUser + " -s " + vzUserGroup + " -s enabled=true"
	ctx.Log().Infof("CDD Create Verrazzano User Cmd = %s", createVzUserCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", createVzUserCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano user: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Successfully Created VZ User")

	// Grant realm admin role to Verrazzano user
	grantRealmAdminToVzUserCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + vzSysRealm + " --uusername " + vzUserName + " --cclientid realm-management --rolename realm-admin"
	ctx.Log().Infof("CDD Grant Realm Admin to Verrazzano User Cmd = %s", grantRealmAdminToVzUserCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", grantRealmAdminToVzUserCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error granting realm admin role to Verrazzano user: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Granted realmAdmin Role to VZ user")

	// Set verrazzano user password
	secret := &corev1.Secret{}
	err = ctx.Client().Get(context.TODO(), client.ObjectKey{
		Namespace: "verrazzano-system",
		Name:      "verrazzano",
	}, secret)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error retrieving Verrazzano password: %s", err)
		return err
	}
	pw := secret.Data["password"]
	vzpw := string(pw)
	if vzpw == "" {
		return errors.New("configureKeycloakRealm: Error retrieving verrazzano password")
	}

	setVZUserPwCmd := "/opt/jboss/keycloak/bin/kcadm.sh set-password -r " + vzSysRealm + " --username " + vzUserName + " --new-password " + vzpw
	ctx.Log().Infof("CDD Set Verrazzano User PW Cmd = %s", setVZUserPwCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", setVZUserPwCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error setting Verrazzano user password: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Created VZ User PW")

	// Creating Verrazzano Internal Prometheus User
	vzPromUser := "username=" + vzInternalPromUser
	vzPromUserGroup := "groups[0]=/" + vzUsersGroup + "/" + vzSystemGroup
	createVZPromUserCmd := "/opt/jboss/keycloak/bin/kcadm.sh create users -r " + vzSysRealm + " -s " + vzPromUser + " -s " + vzPromUserGroup + " -s enabled=true"
	ctx.Log().Infof("CDD Create Verrazzano Prom User Cmd = %s", createVZPromUserCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", createVZPromUserCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano internal Prometheus user: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Successfully Created Prom User")

	// Set verrazzano internal prom user password
	setPromUserPwCmd := "/opt/jboss/keycloak/bin/kcadm.sh set-password -r " + vzSysRealm + " --username " + vzInternalPromUser + " --new-password " + prompw
	ctx.Log().Infof("CDD Set Verrazzano Prom User PW Cmd = %s", setPromUserPwCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", setPromUserPwCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error setting Verrazzano internal Prometheus user password: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Created Prom User PW")

	// Creating Verrazzano Internal ES User
	vzEsUser := "username=" + vzInternalEsUser
	vzEsUserGroup := "groups[0]=/" + vzUsersGroup + "/" + vzSystemGroup
	createVzEsUserCmd := "/opt/jboss/keycloak/bin/kcadm.sh create users -r " + vzSysRealm + " -s " + vzEsUser + " -s " + vzEsUserGroup + " -s enabled=true"
	ctx.Log().Infof("CDD Create VZ ES User Cmd = %s", createVzEsUserCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", createVzEsUserCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano internal Elasticsearch user: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Created ES User")

	// Set verrazzano internal ES user password
	setVzESUserPwCmd := "/opt/jboss/keycloak/bin/kcadm.sh set-password -r " + vzSysRealm + " --username " + vzInternalEsUser + " --new-password " + espw
	ctx.Log().Infof("CDD Set Verrazzano ES User PW Cmd = %s", setVzESUserPwCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", setVzESUserPwCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error setting Verrazzano internal Elasticsearch user password: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Created ES User PW")

	// Get DNS Domain Configuration
	dnsSubDomain, err := getDNSDomain(ctx.Client(), ctx.EffectiveCR())
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealms: Error retrieving DNS sub domain: %s", err)
		return err
	}
	ctx.Log().Infof("configureKeycloakRealms: DNSDomain returned %s", dnsSubDomain)

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

	ctx.Log().Infof("CDD Create verrazzano-pkce client Cmd = %s", vzPkceCreateCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", vzPkceCreateCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating verrazzano-pkce client: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Created verrazzano-pkce client")

	// Creating verrazzano-pg client
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
	ctx.Log().Infof("CDD Create verrazzano-pg client Cmd = %s", vzPgCreateCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", vzPgCreateCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating verrazzano-pg client: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Created verrazzano-pg client")

	// Setting password policy for master
	setPolicyCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/master -s \"passwordPolicy=length(8) and notUsername\""
	ctx.Log().Infof("CDD Setting password policy for master Cmd = %s", setPolicyCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", setPolicyCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error Setting password policy for master: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Set password policy for master")

	// Setting password policy for $_VZ_REALM
	setPolicyOnVzRealmCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/" + vzSysRealm + " -s \"passwordPolicy=length(8) and notUsername\""
	ctx.Log().Infof("CDD Setting password policy for VZ_REALM Cmd = %s", setPolicyOnVzRealmCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", setPolicyOnVzRealmCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error Setting password policy for VZ Realm: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Set password policy for VZ_REALM")

	// Configuring login theme for master
	setMasterLoginThemeCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/master -s loginTheme=oracle"
	ctx.Log().Infof("CDD Configuring login theme for master Cmd = %s", setMasterLoginThemeCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", setMasterLoginThemeCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error Configuring login theme for master: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Configured login theme for master Cmd")

	// Configuring login theme for vzSysRealm
	setVzRealmLoginThemeCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/" + vzSysRealm + " -s loginTheme=oracle"
	ctx.Log().Infof("CDD Configuring login theme for vzSysRealm Cmd = %s", setVzRealmLoginThemeCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", setVzRealmLoginThemeCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error Configuring login theme for vzSysRealm: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Configured login theme for vzSysRealm")

	// Enabling vzSysRealm realm
	setVzEnableRealmCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/" + vzSysRealm + " -s enabled=true"
	ctx.Log().Infof("CDD Enabling vzSysRealm realm Cmd = %s", setVzEnableRealmCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", setVzEnableRealmCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error Enabling vzSysRealm realm: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Enabled vzSysRealm realm")

	// Removing login config file
	removeLoginConfigFileCmd := "rm /root/.keycloak/kcadm.config"
	ctx.Log().Infof("CDD Removing login config file Cmd = %s", removeLoginConfigFileCmd)
	stdout, stderr, err = ExecCmd(cli, cfg, "keycloak-0", removeLoginConfigFileCmd)
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error Removing login config file: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Info("CDD Removed login config file")
	ctx.Log().Info("CDD Keycloak PostUpgrade SUCCESS")
	return nil
}

func loginKeycloak(ctx spi.ComponentContext, cfg *restclient.Config, cli restclient.Interface) error {
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
		return errors.New("loginKeycloak: Error retrieving Keycloak password")
	}
	ctx.Log().Info("loginKeycloak: Successfully retrieved Keycloak password")

	// Login to Keycloak

	loginCmd := "/opt/jboss/keycloak/bin/kcadm.sh config credentials --server http://localhost:8080/auth --realm master --user keycloakadmin --password " + keycloakpw
	cmd := []string{
		"bash",
		"-c",
		loginCmd,
	}
	ctx.Log().Infof("CDD Login Cmd = %s", loginCmd)
	ctx.Log().Infof("CDD Total Cmd = %v", cmd)
	//	stdOut, stdErr, err := k8sutil.ExecPod(cfg, cli.RESTClient(), &keycloakPod, "keycloak", cmd)
	// err = ExecCmd(cli, cfg, "keycloak-0", loginCmd, os.Stdin, os.Stdout, os.Stderr)
	stdOut, stdErr, err := ExecCmd(cli, cfg, "keycloak-0", loginCmd)
	if err != nil {
		ctx.Log().Errorf("loginKeycloak: Error retrieving logging into Keycloak: stdout = %s: stderr = %s", stdOut, stdErr)
		return err
	}
	ctx.Log().Info("loginKeycloak: Successfully logged into Keycloak")

	return nil
}

// ExecCmd exec command on specific pod and wait the command's output.
func ExecCmd(client restclient.Interface, config *restclient.Config, podName string, command string) (string, string, error) {

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := []string{
		"bash",
		"-c",
		command,
	}
	req := client.Post().Resource("pods").Name(podName).
		Namespace("keycloak").SubResource("exec")
	option := &v1.PodExecOptions{
		Command:   cmd,
		Container: "keycloak",
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}
	if os.Stdin == nil {
		option.Stdin = false
	}
	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)
	executor, err := k8sutil.NewPodExecutor(config, "POST", req.URL())
	if err != nil {
		return stdout.String(), stderr.String(), err
	}
	err = executor.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("error running command %s on Keycloak Pod: %v", command, err)
	}

	return stdout.String(), stderr.String(), nil
}

func createOrUpdateAuthSecret(ctx spi.ComponentContext, namespace string, secretname string, username string, password string) error {
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretname, Namespace: namespace},
	}

	opResult, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &secret, func() error {
		// Build the secret data
		secret.Data = map[string][]byte{
			"username": []byte(username),
			"password": []byte(password),
		}
		return nil
	})
	ctx.Log().Infof("Keycloak secret operation result: %s", opResult)

	if err != nil {
		return err
	}
	return nil
}

func getDNSDomain(c client.Client, vz *vzapi.Verrazzano) (string, error) {
	dnsSuffix, err := vzconfig.GetDNSSuffix(c, vz)
	if err != nil {
		return "", err
	}
	dnsDomain := fmt.Sprintf("%s.%s", vz.Spec.EnvironmentName, dnsSuffix)
	return dnsDomain, nil
}
