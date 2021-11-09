// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	corev1 "k8s.io/api/core/v1"
	"os/exec"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"text/template"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	dnsTarget = "dnsTarget"
	rulesHost = "rulesHost"
	tlsHosts  = "tlsHosts"
	tlsSecret = "tlsSecret"
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
	var keycloakIngress = &networkingv1.Ingress{}
	err = compContext.Client().Get(context.TODO(), types.NamespacedName{Name: constants.KeycloakIngress, Namespace: constants.KeycloakNamespace}, keycloakIngress)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch ingress %s/%s, %v", constants.KeycloakIngress, constants.KeycloakNamespace, err)
	}

	if len(keycloakIngress.Spec.TLS) == 0 || len(keycloakIngress.Spec.TLS[0].Hosts) == 0 {
		return nil, fmt.Errorf("no ingress hosts found for %s/%s, %v", constants.KeycloakIngress, constants.KeycloakNamespace, err)
	}

	host := keycloakIngress.Spec.TLS[0].Hosts[0]

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

// // getEnvironmentName returns the name of the Verrazzano install environment
func getEnvironmentName(envName string) string {
	if envName == "" {
		return constants.DefaultEnvironmentName
	}

	return envName
}

func updateKeycloakUris(ctx spi.ComponentContext) error {
	var keycloakClients KeycloakClients

	if ctx.EffectiveCR().Spec.Profile != vzapi.ManagedCluster {
		// Get the Keycloak admin password
		secret := &corev1.Secret{}
		err := ctx.Client().Get(context.TODO(), client.ObjectKey{
			Namespace: "keycloak",
			Name:      "keycloak-http",
		}, secret)
		if err != nil {
			ctx.Log().Errorf("Keycloak Post Upgrade: Error retrieving Keycloak password: %s", err)
			return err
		}
		pw := secret.Data["password"]
		keycloakpw := string(pw)
		if keycloakpw == "" {
			return errors.New("Keycloak Post Upgrade: Error retrieving Keycloak password")
		}
		ctx.Log().Info("Keycloak Post Upgrade: Successfully retrieved Keycloak password")

		// Login to Keycloak
		cmd := execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--",
			"/opt/jboss/keycloak/bin/kcadm.sh", "config", "credentials", "--server", "http://localhost:8080/auth", "--realm", "master", "--user", "keycloakadmin", "--password", keycloakpw)
		_, err = cmd.Output()
		if err != nil {
			ctx.Log().Errorf("Keycloak Post Upgrade: Error logging into Keycloak: %s", err)
			return err
		}
		ctx.Log().Info("Keycloak Post Upgrade: Successfully logged into Keycloak")

		// Get the Client ID JSON array
		cmd = execCommand("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "get", "clients", "-r", "verrazzano-system", "--fields", "id,clientId")
		out, err := cmd.Output()
		if err != nil {
			ctx.Log().Errorf("Keycloak Post Upgrade: Error retrieving ID for client ID, zero length: %s", err)
			return err
		}
		if len(string(out)) == 0 {
			return errors.New("Keycloak Post Upgrade: error retrieving Clients JSON from Keycloak, zero length")
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
		dnsSubDomain, err := nginx.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
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
