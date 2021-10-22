// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"bytes"
	"context"
	"fmt"
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

const kcExtraEnvKey = "extraEnv"
const kcExtraEnvKeyValue = `
  - name: DB_VENDOR
    value: mysql
  - name: DB_ADDR
    value: mysql
  - name: DB_PORT
    value: "3306"
  - name: DB_DATABASE
    value: keycloak
  - name: DB_USER
    value: keycloak
  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: mysql
        key: mysql-password
  - name: PROXY_ADDRESS_FORWARDING
    value: "true"
`

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

	// Override extraEnv helm value
	kvs = append(kvs, bom.KeyValue{
		Key:   kcExtraEnvKey,
		Value: kcExtraEnvKeyValue,
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
