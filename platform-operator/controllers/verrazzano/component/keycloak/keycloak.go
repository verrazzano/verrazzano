// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"bytes"
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"text/template"
)

// ComponentName is the name of the component
const ComponentName = "keycloak"

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
          mountPath: /cacerts"
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

	// Return a new Key:Value pair with the rendered Value
	kvs = append(kvs, bom.KeyValue{
		Key:   kcInitContainerKey,
		Value: b.String(),
	})

	var ingress = &networkingv1.Ingress{}
	err = compContext.Client().Get(context.TODO(), types.NamespacedName{Name: "verrazzano-ingress", Namespace: "verrazzano-system"}, ingress)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch ingress %s/%s, %v", "verrazzano-ingress", "verrazzano-system", err)
	}

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress\\.annotations\\.external-dns\\.alpha\\.kubernetes\\.io/target",
		Value: ingress.Spec.TLS[0].Hosts[0],
	})

	kvs = append(kvs, bom.KeyValue{
		Key:       "ingress\\.annotations\\.nginx\\.ingress\\.kubernetes\\.io/service-upstream",
		Value:     "true",
		SetString: true,
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress\\.annotations\\.nginx\\.ingress\\.kubernetes\\.io/upstream-vhost",
		Value: "keycloak-http.keycloak.svc.cluster.local",
	})

	var keycloakIngress = &networkingv1.Ingress{}
	err = compContext.Client().Get(context.TODO(), types.NamespacedName{Name: "keycloak", Namespace: "keycloak"}, keycloakIngress)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch ingress %s/%s, %v", "keycloak", "keycloak", err)
	}

	ingressHosts := fmt.Sprintf("{%s}", keycloakIngress.Spec.TLS[0].Hosts[0])
	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress\\.hosts",
		Value: ingressHosts,
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress\\.rules[0]\\.host",
		Value: keycloakIngress.Spec.TLS[0].Hosts[0],
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress\\.rules[0]\\.paths[0]\\.path",
		Value: "/",
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress\\.rules[0]\\.paths[0]\\.pathType",
		Value: "ImplementationSpecific",
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress\\.tls[0]\\.hosts",
		Value: ingressHosts,
	})

	tlsSecret := fmt.Sprintf("%s-secret", compContext.EffectiveCR().Spec.EnvironmentName)
	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress\\.tls[0]\\.secretName",
		Value: tlsSecret,
	})
	return kvs, nil
}
