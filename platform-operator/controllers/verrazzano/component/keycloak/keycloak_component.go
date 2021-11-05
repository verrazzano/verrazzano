// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package keycloak

import (
	"context"
	"encoding/json"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	corev1 "k8s.io/api/core/v1"
	"log"
	"os/exec"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentName is the name of the component
const ComponentName = "keycloak"

// KeycloakComponent represents an Keycloak component
type KeycloakComponent struct {
	helm.HelmComponent
}

// KeycloakClients represents an array of clients currently configured in Keycloak
type KeycloakClients []struct {
	ID       string `json:"id"`
	ClientID string `json:"clientId"`
}

type bashFuncSig func(inArgs ...string) (string, string, error)

var bashFunc bashFuncSig = vzos.RunBash

// Verify that KeycloakComponent implements Component
var _ spi.Component = KeycloakComponent{}

// NewComponent returns a new Keycloak component
func NewComponent() spi.Component {
	return KeycloakComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:          ComponentName,
			IgnoreNamespaceOverride: true,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "keycloak-values.yaml"),
			Dependencies:            []string{istio.ComponentName},
			AppendOverridesFunc:     AppendKeycloakOverrides,
		},
	}
}

// PostUpgrade Keycloak-post-upgrade processing, create or update the Kiali ingress
func (c KeycloakComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Infof("Keycloak post-upgrade")
	if err := c.HelmComponent.PostUpgrade(ctx); err != nil {
		return err
	}
	return updateKeycloakUris(ctx)
}

func updateKeycloakUris(ctx spi.ComponentContext) error {
	var keycloakClients KeycloakClients

	if ctx.ActualCR().Spec.Profile != vzapi.ManagedCluster {
		// Get the Keycloak admin password
		secret := &corev1.Secret{}
		err := ctx.Client().Get(context.TODO(), client.ObjectKey{
			Namespace: "keycloak",
			Name:      "keycloak-http",
		}, secret)
		if err != nil {
			log.Fatal(err)
		}
		pw := secret.Data["password"]
		keycloakpw := string(pw)
		if keycloakpw == "" {
			log.Fatal("Keycloak Post Upgrade: Error retrieving Keycloak password")
		}
		ctx.Log().Info("Keycloak Post Upgrade: Successfully retrieved Keycloak password")

		// Login to Keycloak
		cmd := exec.Command("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--",
			"/opt/jboss/keycloak/bin/kcadm.sh", "config", "credentials", "--server", "http://localhost:8080/auth", "--realm", "master", "--user", "keycloakadmin", "--password", keycloakpw)
		_, err = cmd.Output()
		if err != nil {
			log.Fatalf("Error = %s", err)
		}
		ctx.Log().Info("Keycloak Post Upgrade: Successfully logged into Keycloak")

		// Get the Client ID JSON array
		cmd = exec.Command("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "get", "clients", "-r", "verrazzano-system", "--fields", "id,clientId")
		out, err := cmd.Output()
		if err != nil {
			log.Fatalf("Error = %s", err)
		}
		if len(string(out)) == 0 {
			log.Fatal("Error retrieving Clients JSON from Keycloak, zero length\n")
		}
		json.Unmarshal([]byte(out), &keycloakClients)

		// Extract the id associated with ClientID verrazzano-pkce
		var id = ""
		for _, client := range keycloakClients {
			if client.ClientID == "verrazzano-pkce" {
				id = client.ID
				ctx.Log().Infof("Keycloak Clients ID found = %s", id)
			}
		}
		if id == "" {
			log.Fatal("Error retrieving ID for verrazzano-pkce, zero length\n")
		}
		ctx.Log().Info("Keycloak Post Upgrade: Successfully retrieved clientID")

		// Get DNS Domain Configuration
		dnsSubDomain, err := nginx.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
		if err != nil {
			return err
		}
		ctx.Log().Infof("DNSDomain returned = %s", dnsSubDomain)

		// Call the Script and Update the URIs
		scriptName := filepath.Join(config.GetInstallDir(), "update-kiali-redirect-uris.sh")
		if _, stderr, err := bashFunc(scriptName, id, dnsSubDomain); err != nil {
			ctx.Log().Errorf("Failed updating KeyCloak URIs %s: %s", err, stderr)
			return err
		}
	}
	ctx.Log().Info("Keycloak Post Upgrade: Successfully Updated Keycloak URIs")
	return nil
}
