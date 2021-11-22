// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"io"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

	// Additional overrides for Keycloak 15.0.2 charts.
	//	var keycloakIngress = &networkingv1.Ingress{}
	//	err = compContext.Client().Get(context.TODO(), types.NamespacedName{Name: constants.KeycloakIngress, Namespace: constants.KeycloakNamespace}, keycloakIngress)
	//	if err != nil {
	//		return nil, fmt.Errorf("unable to fetch ingress %s/%s, %v", constants.KeycloakIngress, constants.KeycloakNamespace, err)
	//	}
	//
	//	if len(keycloakIngress.Spec.TLS) == 0 || len(keycloakIngress.Spec.TLS[0].Hosts) == 0 {
	//		return nil, fmt.Errorf("no ingress hosts found for %s/%s, %v", constants.KeycloakIngress, constants.KeycloakNamespace, err)
	//	}

	//	host := keycloakIngress.Spec.TLS[0].Hosts[0]

	// Get DNS Domain Configuration
	dnsSubDomain, err := nginx.BuildDNSDomain(compContext.Client(), compContext.EffectiveCR())
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

	if isKeycloakEnabled(ctx) {
		err := loginKeycloak(ctx)
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
		json.Unmarshal([]byte(out), &keycloakClients)

		// Extract the id associated with ClientID verrazzano-pkce
		var id = ""
		for _, client := range keycloakClients {
			if client.ClientID == "verrazzano-pkce" {
				id = client.ID
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
	}
	ctx.Log().Info("Keycloak Post Upgrade: Successfully Updated Keycloak URIs")
	return nil
}

func configureKeycloakRealms(ctx spi.ComponentContext, prompw string, espw string) error {

	if isKeycloakEnabled(ctx) {
		// Login to Keycloak
		err := loginKeycloak(ctx)
		if err != nil {
			return err
		}

		ctx.Log().Info("CDD Successfully Logged Into Keycloak")

		// Create VerrazzanoSystem Realm
		realm := "realm=" + vzSysRealm
		cmd := execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "realms", "-s", realm, "-s", "enabled=false")
		ctx.Log().Info("CDD Create Verrazzano System Realm Cmd = %s", cmd.String())
		out, err := cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano System Realm: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Successfully Created Verrazzano System Realm")

		// Create Verrazzano Users Group
		userGroup := "name=" + vzUsersGroup
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "groups", "-r", vzSysRealm, "-s", userGroup)
		ctx.Log().Info("CDD Create Verrazzano Users Group Cmd = %s", cmd.String())
		out, err = cmd.CombinedOutput()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Users Group: command output = %s", out)
			return err
		}
		ctx.Log().Infof("CDD Create Verrazzano Users Group Output = %s", out)
		if len(string(out)) == 0 {
			return errors.New("configureKeycloakRealm: Error retrieving User Group ID from Keycloak, zero length")
		}
		arr := strings.Split(string(out), "'")
		userGroupID := arr[1]
		ctx.Log().Infof("configureKeycloakRealm: User Group ID = %s", userGroupID)
		ctx.Log().Info("CDD Successfully Created Verrazzano User Group")

		// Create Verrazzano Admin Group
		adminGroup := "name=" + vzAdminGroup
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "groups", "-r", vzSysRealm, "-s", adminGroup)
		ctx.Log().Infof("CDD Create Verrazzano Admin Group Cmd = %s", cmd.String())
		out, err = cmd.CombinedOutput()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Admin Group: command output = %s", out)
			return err
		}
		ctx.Log().Infof("CDD Create Verrazzano Admin Group Output = %s", out)
		if len(string(out)) == 0 {
			return errors.New("configureKeycloakRealm: Error retrieving Admin Group ID from Keycloak, zero length")
		}
		arr = strings.Split(string(out), "'")
		adminGroupID := arr[1]
		ctx.Log().Infof("configureKeycloakRealm: Admin Group ID = %s", adminGroupID)
		ctx.Log().Info("CDD Successfully Created Verrazzano Admin Group")

		// Create Verrazzano Project Monitors Group
		monitorGroup := "name=" + vzMonitorGroup
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "groups", "-r", vzSysRealm, "-s", monitorGroup)
		ctx.Log().Infof("CDD Create Verrazzano Monitors Group Cmd = %s", cmd.String())
		out, err = cmd.CombinedOutput()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Monitor Group: command output = %s", out)
			return err
		}
		ctx.Log().Infof("CDD Create Verrazzano Project Monitors Group Output = %s", out)
		if len(string(out)) == 0 {
			return errors.New("configureKeycloakRealm: Error retrieving Monitor Group ID from Keycloak, zero length")
		}
		arr = strings.Split(string(out), "'")
		monitorGroupID := arr[1]
		ctx.Log().Infof("configureKeycloakRealm: Monitro Group ID = %s", monitorGroupID)
		ctx.Log().Info("CDD Successfully Created Verrazzano Monitors Group")

		// Create Verrazzano System Group
		systemGroup := "name=" + vzSystemGroup
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "groups", "-r", vzSysRealm, "-s", systemGroup)
		ctx.Log().Infof("CDD Create Verrazzano System Group Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano System Group: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Successfully Created Verrazzano System Group")

		// Create Verrazzano API Access Role
		apiAccessRole := "name=" + vzAPIAccessRole
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "roles", "-r", vzSysRealm, "-s", apiAccessRole)
		ctx.Log().Infof("CDD Create Verrazzano API Access Role Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano API Access Role: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Successfully Created Verrazzano API Access Role")

		// Create Verrazzano Console Users Role
		consoleUserRole := "name=" + vzConsoleUsersRole
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "roles", "-r", vzSysRealm, "-s", consoleUserRole)
		ctx.Log().Infof("CDD Create Verrazzano Console User Role Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Console Users Role: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Successfully Created Verrazzano Console User Role")

		// Create Verrazzano Admin Role
		adminRole := "name=" + vzAdminRole
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "roles", "-r", vzSysRealm, "-s", adminRole)
		ctx.Log().Infof("CDD Create Verrazzano Admin Role Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Admin Role: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Successfully Created Verrazzano Admin Role")

		// Create Verrazzano Viewer Role
		viewerRole := "name=" + vzViewerRole
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "roles", "-r", vzSysRealm, "-s", viewerRole)
		ctx.Log().Infof("CDD Create Verrazzano Viewer Role Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano Viewer Role: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Successfully Created Verrazzano Viewer Role")

		// Granting vz_api_access role to verrazzano users group
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "add-roles", "-r", vzSysRealm, "--gid", userGroupID, "--rolename", vzAPIAccessRole)
		ctx.Log().Infof("CDD Grant Access Role to User Group Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error granting api access role to Verrazzano users group: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Granted Access Role to User Group")

		// Granting console_users role to verrazzano users group
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "add-roles", "-r", vzSysRealm, "--gid", userGroupID, "--rolename", vzConsoleUsersRole)
		ctx.Log().Infof("CDD Grant Console Role to User Group Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error granting console users role to Verrazzano users group: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Granted Console Role to User Group")

		// Granting admin role to verrazzano admin group
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "add-roles", "-r", vzSysRealm, "--gid", adminGroupID, "--rolename", vzAdminRole)
		ctx.Log().Infof("CDD Grant Admin Role to Admin Group Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error granting admin role to Verrazzano admin group: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Granted Admin Role to Admin Group")

		// Granting viewer role to verrazzano monitor group
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "add-roles", "-r", vzSysRealm, "--gid", monitorGroupID, "--rolename", vzViewerRole)
		ctx.Log().Infof("CDD Grant Viewer Role to monitor Group Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error granting viewer role to Verrazzano monitoring group: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Granted Viewer Role to monitor Group")

		// Creating Verrazzano User
		vzUser := "username=" + vzUserName
		vzUserGroup := "groups[0]=" + vzUsersGroup
		vzAdminGroup := "groups[1]=" + vzAdminGroup
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "users", "-r", vzSysRealm, "-s", vzUser, "-s", vzUserGroup, "-s", vzAdminGroup, "-s", "enabled=true")
		ctx.Log().Infof("CDD Create VZ User Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano user: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Successfully Created VZ User")

		// Grant realm admin role to Verrazzano user
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "add-roles", "-r", vzSysRealm, "--uusername", vzUserName, "--cclientid", "realm-management", "--rolename", "realm-admin")
		ctx.Log().Infof("CDD Grant realmAdmin Role to VZ user Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error granting realm admin role to Verrazzano user: command output = %s", out)
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

		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "set-password", "-r", vzSysRealm, "--username", vzUserName, "--new-password", vzpw)
		ctx.Log().Infof("CDD Create VZ User PW Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error setting Verrazzano user password: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Created VZ User PW")

		// Creating Verrazzano Internal Prometheus User
		vzPromUser := "username=" + vzInternalPromUser
		vzPromUserGroup := "groups[0]=" + vzUsersGroup
		vzPromSystemGroup := "groups[1]=" + vzSystemGroup
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "users", "-r", vzSysRealm, "-s", vzPromUser, "-s", vzPromUserGroup, "-s", vzPromSystemGroup, "-s", "enabled=true")
		ctx.Log().Infof("CDD Create Prom User Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano internal Prometheus user: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Successfully Created Prom User")

		// Set verrazzano internal prom user password
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "set-password", "-r", vzSysRealm, "--username", vzInternalPromUser, "--new-password", prompw)
		ctx.Log().Infof("CDD Create Prom User PW Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error setting Verrazzano internal Prometheus user password: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Created Prom User PW")

		// Creating Verrazzano Internal ES User
		vzEsUser := "username=" + vzInternalEsUser
		vzEsUserGroup := "groups[0]=/" + vzUsersGroup
		vzEsSystemGroup := "groups[1]=" + vzSystemGroup
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "create", "users", "-r", vzSysRealm, "-s", vzEsUser, "-s", vzEsUserGroup, "-s", vzEsSystemGroup, "-s", "enabled=true")
		ctx.Log().Infof("CDD Create ES User Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error creating Verrazzano internal Elasticsearch user: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Created ES User")

		// Set verrazzano internal ES user password
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "set-password", "-r", vzSysRealm, "--username", vzInternalEsUser, "--new-password", espw)
		ctx.Log().Infof("CDD Create ES User PW Cmd = %s", cmd.String())
		out, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("configureKeycloakRealm: Error setting Verrazzano internal Elasticsearch user password: command output = %s", out)
			return err
		}
		ctx.Log().Info("CDD Created ES User PW")

		// Get DNS Domain Configuration
		dnsSubDomain, err := nginx.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
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

		//		config, err := k8sutil.GetKubeConfig()
		config, err := controllerruntime.GetConfig()
		if err != nil {
			return err
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return err
		}
		ctx.Log().Infof("CDD Create verrazzano-pkce client Cmd = %s", vzPkceCreateCmd)
		err = ExecCmd(clientset, config, "keycloak-0", vzPkceCreateCmd, os.Stdin, os.Stdout, os.Stderr)
		if err != nil {
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
		err = ExecCmd(clientset, config, "keycloak-0", vzPgCreateCmd, os.Stdin, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
		ctx.Log().Info("CDD Created verrazzano-pg client")

		// Setting password policy for master
		setPolicyCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/master -s \"passwordPolicy=length(8) and notUsername\""
		ctx.Log().Infof("CDD Setting password policy for master Cmd = %s", setPolicyCmd)
		err = ExecCmd(clientset, config, "keycloak-0", setPolicyCmd, os.Stdin, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
		ctx.Log().Info("CDD Set password policy for master")

		// Setting password policy for $_VZ_REALM
		setPolicyOnVzRealmCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/" + vzSysRealm + " -s \"passwordPolicy=length(8) and notUsername\""
		ctx.Log().Infof("CDD Setting password policy for VZ_REALM Cmd = %s", setPolicyOnVzRealmCmd)
		err = ExecCmd(clientset, config, "keycloak-0", setPolicyOnVzRealmCmd, os.Stdin, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
		ctx.Log().Info("CDD Set password policy for VZ_REALM", cmd.String())

		// Configuring login theme for master
		setMasterLoginThemeCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/master -s loginTheme=oracle"
		ctx.Log().Infof("CDD Configuring login theme for master Cmd = %s", setMasterLoginThemeCmd)
		err = ExecCmd(clientset, config, "keycloak-0", setMasterLoginThemeCmd, os.Stdin, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
		ctx.Log().Info("CDD Configured login theme for master Cmd")

		// Configuring login theme for vzSysRealm
		setVzRealmLoginThemeCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/" + vzSysRealm + " -s loginTheme=oracle"
		ctx.Log().Infof("CDD Configuring login theme for vzSysRealm Cmd = %s", setVzRealmLoginThemeCmd)
		err = ExecCmd(clientset, config, "keycloak-0", setVzRealmLoginThemeCmd, os.Stdin, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
		ctx.Log().Info("CDD Configured login theme for vzSysRealm")

		// Enabling vzSysRealm realm
		setVzEnableRealmCmd := "/opt/jboss/keycloak/bin/kcadm.sh update realms/" + vzSysRealm + " -s enabled=true"
		ctx.Log().Infof("CDD Enabling vzSysRealm realm Cmd = %s", setVzEnableRealmCmd)
		err = ExecCmd(clientset, config, "keycloak-0", setVzEnableRealmCmd, os.Stdin, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
		ctx.Log().Info("CDD Enabled vzSysRealm realm")

		// Removing login config file
		removeLoginConfigFileCmd := "rm /root/.keycloak/kcadm.config"
		ctx.Log().Infof("CDD Removing login config file Cmd = %s", removeLoginConfigFileCmd)
		err = ExecCmd(clientset, config, "keycloak-0", removeLoginConfigFileCmd, os.Stdin, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
		ctx.Log().Info("CDD Removed login config file")
		ctx.Log().Info("CDD Keycloak PostUpgrade SUCCESS")
	}

	return nil
}

func loginKeycloak(ctx spi.ComponentContext) error {
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
	cmd := execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--",
		"/opt/jboss/keycloak/bin/kcadm.sh", "config", "credentials", "--server", "http://localhost:8080/auth", "--realm", "master", "--user", "keycloakadmin", "--password", keycloakpw)
	_, err = cmd.Output()
	if err != nil {
		ctx.Log().Errorf("loginKeycloak: Error logging into Keycloak: %s", err)
		return err
	}
	ctx.Log().Info("loginKeycloak: Successfully logged into Keycloak")

	return nil
}

// ExecCmd exec command on specific pod and wait the command's output.
func ExecCmd(client kubernetes.Interface, config *restclient.Config, podName string,
	command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := []string{
		"bash",
		"-c",
		command,
	}
	req := client.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
		Namespace("keycloak").SubResource("exec")
	option := &v1.PodExecOptions{
		Command:   cmd,
		Container: "keycloak",
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}
	if stdin == nil {
		option.Stdin = false
	}
	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return err
	}

	return nil
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
