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
	vzpassword "github.com/verrazzano/verrazzano/pkg/security/password"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
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
	var keycloakClients KeycloakClients
	cfg, cli, err := k8sutil.ClientConfig()
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
			ctx.Log().Debugf("Keycloak Post Upgrade: ID found = %s", id)
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

// configureKeycloakRealms configures the Verrazzano system realm
func configureKeycloakRealms(ctx spi.ComponentContext) error {
	cfg, cli, err := k8sutil.ClientConfig()
	kcPod := keycloakPod()
	if err != nil {
		return err
	}
	// Login to Keycloak
	err = loginKeycloak(ctx, cfg, cli)
	if err != nil {
		return err
	}

	// Create VerrazzanoSystem Realm
	realm := "realm=" + vzSysRealm
	createRealmCmd := "/opt/jboss/keycloak/bin/kcadm.sh create realms -s " + realm + " -s enabled=false"
	ctx.Log().Debugf("configureKeycloakRealm: Create Verrazzano System Realm Cmd = %s", createRealmCmd)
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createRealmCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano System Realm: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Successfully Created Verrazzano System Realm")

	// Create Verrazzano Users Group
	userGroup := "name=" + vzUsersGroup
	cmd := execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "groups", "-r", vzSysRealm, "-s", userGroup)
	ctx.Log().Debugf("configureKeycloakRealm: Create Verrazzano Users Group Cmd = %s", cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Users Group: command output = %s", out)
		return err
	}
	ctx.Log().Debugf("configureKeycloakRealm: Create Verrazzano Users Group Output = %s", out)
	if len(string(out)) == 0 {
		return errors.New("configureKeycloakRealm: Error retrieving User Group ID from Keycloak, zero length")
	}
	arr := strings.Split(string(out), "'")
	if len(arr) != 3 {
		return fmt.Errorf("configureKeycloakRealm: Error parsing output returned from Users Group create stdout returned = %s", out)
	}
	userGroupID := arr[1]
	ctx.Log().Debugf("configureKeycloakRealm: User Group ID = %s", userGroupID)
	ctx.Log().Debug("configureKeycloakRealm: Successfully Created Verrazzano User Group")

	// Create Verrazzano Admin Group
	adminGroup := "groups/" + userGroupID + "/children"
	adminGroupName := "name=" + vzAdminGroup
	cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", adminGroup, "-r", vzSysRealm, "-s", adminGroupName)
	ctx.Log().Debugf("configureKeycloakRealm: Create Verrazzano Admin Group Cmd = %s", cmd.String())
	out, err = cmd.CombinedOutput()
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Admin Group: command output = %s", out)
		return err
	}
	ctx.Log().Debugf("configureKeycloakRealm: Create Verrazzano Admin Group Output = %s", out)
	if len(string(out)) == 0 {
		return errors.New("configureKeycloakRealm: Error retrieving Admin Group ID from Keycloak, zero length")
	}
	arr = strings.Split(string(out), "'")
	if len(arr) != 3 {
		return fmt.Errorf("configureKeycloakRealm: Error parsing output returned from Admin Group create stdout returned = %s", out)
	}
	adminGroupID := arr[1]
	ctx.Log().Debugf("configureKeycloakRealm: Admin Group ID = %s", adminGroupID)
	ctx.Log().Debug("configureKeycloakRealm: Successfully Created Verrazzano Admin Group")

	// Create Verrazzano Project Monitors Group
	monitorGroup := "groups/" + userGroupID + "/children"
	monitorGroupName := "name=" + vzMonitorGroup
	cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", monitorGroup, "-r", vzSysRealm, "-s", monitorGroupName)
	ctx.Log().Debugf("configureKeycloakRealm: Create Verrazzano Monitors Group Cmd = %s", cmd.String())
	out, err = cmd.CombinedOutput()
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Monitor Group: command output = %s", out)
		return err
	}
	ctx.Log().Debugf("configureKeycloakRealm: Create Verrazzano Project Monitors Group Output = %s", out)
	if len(string(out)) == 0 {
		return errors.New("configureKeycloakRealm: Error retrieving Monitor Group ID from Keycloak, zero length")
	}
	arr = strings.Split(string(out), "'")
	if len(arr) != 3 {
		return fmt.Errorf("configureKeycloakRealm: Error parsing output returned from Monitor Group create stdout returned = %s", out)
	}
	monitorGroupID := arr[1]
	ctx.Log().Debugf("configureKeycloakRealm: Monitor Group ID = %s", monitorGroupID)
	ctx.Log().Debug("configureKeycloakRealm: Successfully Created Verrazzano Monitors Group")

	// Create Verrazzano System Group
	systemGroup := "groups/" + userGroupID + "/children"
	systemGroupName := "name=" + vzSystemGroup
	createVzSystemGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh create " + systemGroup + " -r " + vzSysRealm + " -s " + systemGroupName
	ctx.Log().Debugf("configureKeycloakRealm: Create Verrazzano System Group Cmd = %s", createVzSystemGroupCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createVzSystemGroupCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano System Group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Successfully Created Verrazzano System Group")

	// Create Verrazzano API Access Role
	apiAccessRole := "name=" + vzAPIAccessRole
	createAPIAccessRoleCmd := "/opt/jboss/keycloak/bin/kcadm.sh create roles -r " + vzSysRealm + " -s " + apiAccessRole
	ctx.Log().Debugf("configureKeycloakRealm: Create Verrazzano API Access Role Cmd = %s", createAPIAccessRoleCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createAPIAccessRoleCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano API Access Role: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Successfully Created Verrazzano API Access Role")

	// Create Verrazzano Console Users Role
	consoleUserRole := "name=" + vzConsoleUsersRole
	createConsoleUserRoleCmd := "/opt/jboss/keycloak/bin/kcadm.sh create roles -r " + vzSysRealm + " -s " + consoleUserRole
	ctx.Log().Debugf("configureKeycloakRealm: Create Verrazzano Console Users Role Cmd = %s", createConsoleUserRoleCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createConsoleUserRoleCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Console Users Role: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Successfully Created Verrazzano Console User Role")

	// Create Verrazzano Admin Role
	adminRole := "name=" + vzAdminRole
	createVzAdminRoleCmd := "/opt/jboss/keycloak/bin/kcadm.sh create roles -r " + vzSysRealm + " -s " + adminRole
	ctx.Log().Debugf("configureKeycloakRealm: Create Verrazzano Admin Role Cmd = %s", createVzAdminRoleCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createVzAdminRoleCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Admin Role: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Successfully Created Verrazzano Admin Role")

	// Create Verrazzano Viewer Role
	viewerRole := "name=" + vzViewerRole
	createVzViewerRoleCmd := "/opt/jboss/keycloak/bin/kcadm.sh create roles -r " + vzSysRealm + " -s " + viewerRole
	ctx.Log().Debugf("configureKeycloakRealm: Create Verrazzano Viewer Role Cmd = %s", createVzViewerRoleCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createVzViewerRoleCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Viewer Role: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Successfully Created Verrazzano Viewer Role")

	// Granting vz_api_access role to verrazzano users group
	grantAPIAccessToVzUserGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + vzSysRealm + " --gid " + userGroupID + " --rolename " + vzAPIAccessRole
	ctx.Log().Debugf("configureKeycloakRealm: Grant API Access to VZ Users Cmd = %s", grantAPIAccessToVzUserGroupCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(grantAPIAccessToVzUserGroupCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error granting api access role to Verrazzano users group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Granted Access Role to User Group")

	// Granting console_users role to verrazzano users group
	grantConsoleRoleToVzUserGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + vzSysRealm + " --gid " + userGroupID + " --rolename " + vzConsoleUsersRole
	ctx.Log().Debugf("configureKeycloakRealm: Grant Console Role to Vz Users Cmd = %s", grantConsoleRoleToVzUserGroupCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(grantConsoleRoleToVzUserGroupCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error granting console users role to Verrazzano users group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Granted Console Role to User Group")

	// Granting admin role to verrazzano admin group
	grantAdminRoleToVzAdminGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + vzSysRealm + " --gid " + adminGroupID + " --rolename " + vzAdminRole
	ctx.Log().Debugf("configureKeycloakRealm: Grant Admin Role to Vz Admin Cmd = %s", grantAdminRoleToVzAdminGroupCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(grantAdminRoleToVzAdminGroupCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error granting admin role to Verrazzano admin group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Granted Admin Role to Admin Group")

	// Granting viewer role to verrazzano monitor group
	grantViewerRoleToVzMonitorGroupCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + vzSysRealm + " --gid " + monitorGroupID + " --rolename " + vzViewerRole
	ctx.Log().Debugf("configureKeycloakRealm: Grant Viewer Role to Monitor Group Cmd = %s", grantViewerRoleToVzMonitorGroupCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(grantViewerRoleToVzMonitorGroupCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error granting viewer role to Verrazzano monitoring group: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Granted Viewer Role to monitor Group")

	// Creating Verrazzano User
	vzUser := "username=" + vzUserName
	vzUserGroup := "groups[0]=/" + vzUsersGroup + "/" + vzAdminGroup
	createVzUserCmd := "/opt/jboss/keycloak/bin/kcadm.sh create users -r " + vzSysRealm + " -s " + vzUser + " -s " + vzUserGroup + " -s enabled=true"
	ctx.Log().Debugf("configureKeycloakRealm: Create Verrazzano User Cmd = %s", createVzUserCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createVzUserCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano user: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Successfully Created VZ User")

	// Grant realm admin role to Verrazzano user
	grantRealmAdminToVzUserCmd := "/opt/jboss/keycloak/bin/kcadm.sh add-roles -r " + vzSysRealm + " --uusername " + vzUserName + " --cclientid realm-management --rolename realm-admin"
	ctx.Log().Debugf("configureKeycloakRealm: Grant Realm Admin to Verrazzano User Cmd = %s", grantRealmAdminToVzUserCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(grantRealmAdminToVzUserCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error granting realm admin role to Verrazzano user: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Granted realmAdmin Role to VZ user")

	vzpw, err := getSecretPassword(ctx, "verrazzano-system", "verrazzano")
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error retrieving Verrazzano password: %s", err)
		return err
	}
	setVZUserPwCmd := "/opt/jboss/keycloak/bin/kcadm.sh set-password -r " + vzSysRealm + " --username " + vzUserName + " --new-password " + vzpw
	ctx.Log().Debugf("configureKeycloakRealm: Set Verrazzano User PW Cmd = %s", maskPw(setVZUserPwCmd))
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setVZUserPwCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error setting Verrazzano user password: stdout = %s, stderr = %s", stdout, stderr)
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	ctx.Log().Debug("configureKeycloakRealm: Created VZ User PW")

	// Creating Verrazzano Internal Prometheus User
	vzPromUser := "username=" + vzInternalPromUser
	vzPromUserGroup := "groups[0]=/" + vzUsersGroup + "/" + vzSystemGroup
	createVZPromUserCmd := "/opt/jboss/keycloak/bin/kcadm.sh create users -r " + vzSysRealm + " -s " + vzPromUser + " -s " + vzPromUserGroup + " -s enabled=true"
	ctx.Log().Debugf("configureKeycloakRealm: Create Verrazzano Prom User Cmd = %s", createVZPromUserCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createVZPromUserCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano internal Prometheus user: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Successfully Created Prom User")

	// Set verrazzano internal prom user password
	prompw, err := getSecretPassword(ctx, "verrazzano-system", "verrazzano-prom-internal")
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error getting Verrazzano internal Prometheus user password: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	setPromUserPwCmd := "/opt/jboss/keycloak/bin/kcadm.sh set-password -r " + vzSysRealm + " --username " + vzInternalPromUser + " --new-password " + prompw
	ctx.Log().Debugf("configureKeycloakRealm: Set Verrazzano Prom User PW Cmd = %s", maskPw(setPromUserPwCmd))
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setPromUserPwCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error setting Verrazzano internal Prometheus user password: stdout = %s, stderr = %s", stdout, stderr)
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	ctx.Log().Debug("configureKeycloakRealm: Created Prom User PW")

	// Creating Verrazzano Internal ES User
	vzEsUser := "username=" + vzInternalEsUser
	vzEsUserGroup := "groups[0]=/" + vzUsersGroup + "/" + vzSystemGroup
	createVzEsUserCmd := "/opt/jboss/keycloak/bin/kcadm.sh create users -r " + vzSysRealm + " -s " + vzEsUser + " -s " + vzEsUserGroup + " -s enabled=true"
	ctx.Log().Debugf("configureKeycloakRealm: Create VZ ES User Cmd = %s", createVzEsUserCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(createVzEsUserCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano internal Elasticsearch user: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Created ES User")

	// Set verrazzano internal ES user password
	espw, err := getSecretPassword(ctx, "verrazzano-system", "verrazzano-es-internal")
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error getting Verrazzano internal Elasticsearch user password: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	setVzESUserPwCmd := "/opt/jboss/keycloak/bin/kcadm.sh set-password -r " + vzSysRealm + " --username " + vzInternalEsUser + " --new-password " + espw
	ctx.Log().Debugf("configureKeycloakRealm: Set Verrazzano ES User PW Cmd = %s", maskPw(setVzESUserPwCmd))
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setVzESUserPwCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error setting Verrazzano internal Elasticsearch user password: stdout = %s, stderr = %s", stdout, stderr)
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	ctx.Log().Debug("configureKeycloakRealm: Created ES User PW")

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

	ctx.Log().Debugf("configureKeycloakRealm: Create verrazzano-pkce client Cmd = %s", vzPkceCreateCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(vzPkceCreateCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating verrazzano-pkce client: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Created verrazzano-pkce client")

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
	ctx.Log().Debugf("configureKeycloakRealm: Create verrazzano-pg client Cmd = %s", vzPgCreateCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(vzPgCreateCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error creating verrazzano-pg client: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Created verrazzano-pg client")

	// Setting password policy for master
	setPolicyCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/master -s \"passwordPolicy=length(8) and notUsername\""
	ctx.Log().Debugf("configureKeycloakRealm: Setting password policy for master Cmd = %s", setPolicyCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setPolicyCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error Setting password policy for master: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Set password policy for master")

	// Setting password policy for $_VZ_REALM
	setPolicyOnVzRealmCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/" + vzSysRealm + " -s \"passwordPolicy=length(8) and notUsername\""
	ctx.Log().Debugf("configureKeycloakRealm: Setting password policy for VZ_REALM Cmd = %s", setPolicyOnVzRealmCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setPolicyOnVzRealmCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error Setting password policy for VZ Realm: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Set password policy for VZ_REALM")

	// Configuring login theme for master
	setMasterLoginThemeCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/master -s loginTheme=oracle"
	ctx.Log().Debugf("configureKeycloakRealm: Configuring login theme for master Cmd = %s", setMasterLoginThemeCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setMasterLoginThemeCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error Configuring login theme for master: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Configured login theme for master Cmd")

	// Configuring login theme for vzSysRealm
	setVzRealmLoginThemeCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/" + vzSysRealm + " -s loginTheme=oracle"
	ctx.Log().Debugf("configureKeycloakRealm: Configuring login theme for vzSysRealm Cmd = %s", setVzRealmLoginThemeCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setVzRealmLoginThemeCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error Configuring login theme for vzSysRealm: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Configured login theme for vzSysRealm")

	// Enabling vzSysRealm realm
	setVzEnableRealmCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/" + vzSysRealm + " -s enabled=true"
	ctx.Log().Debugf("configureKeycloakRealm: Enabling vzSysRealm realm Cmd = %s", setVzEnableRealmCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(setVzEnableRealmCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error Enabling vzSysRealm realm: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Enabled vzSysRealm realm")

	// Removing login config file
	removeLoginConfigFileCmd := "rm /root/.keycloak/kcadm.config"
	ctx.Log().Debugf("configureKeycloakRealm: Removing login config file Cmd = %s", removeLoginConfigFileCmd)
	stdout, stderr, err = k8sutil.ExecPod(cli, cfg, kcPod, ComponentName, bashCMD(removeLoginConfigFileCmd))
	if err != nil {
		ctx.Log().Errorf("configureKeycloakRealm: Error Removing login config file: stdout = %s, stderr = %s", stdout, stderr)
		return err
	}
	ctx.Log().Debug("configureKeycloakRealm: Removed login config file")
	ctx.Log().Info("configureKeycloakRealm: Keycloak PostUpgrade SUCCESS")
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
		return err
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
			Namespace: ComponentName,
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

// getCertName returns certificate name
func getCertName(vz *vzapi.Verrazzano) string {
	return fmt.Sprintf("%s-secret", getEnvironmentName(vz.Spec.EnvironmentName))
}
