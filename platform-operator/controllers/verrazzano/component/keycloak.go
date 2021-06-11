// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"bytes"
	"fmt"
	"go.uber.org/zap"
	"text/template"
)

// Define the keylcloak key:value pair for init container.
// We need to replace image using the real image in the bom
const kcInitContainerKey = "keycloak.extraInitContainers"
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

// appendKeycloakOverrides appends the Keycloak theme for the key keycloak.extraInitContainers.
// A go template is used to replace the image in the init container spec.
func appendKeycloakOverrides(_ *zap.SugaredLogger, _ string, _ string, _ string, kvs []keyValue) ([]keyValue, error) {
	// Create a Bom and get the key value overrides
	bom, err := NewBom(DefaultBomFilePath())
	if err != nil {
		return nil, err
	}

	// Get Keycloak theme images
	images, err := bom.buildImageOverrides("keycloak-oracle-theme")
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
	data := imageData{Image: images[0].value}
	err = t.Execute(&b, data)
	if err != nil {
		return nil, err
	}

	// Return a new key:value pair with the rendered value
	kvs = append(kvs, keyValue{
		key:   kcInitContainerKey,
		value: b.String(),
	})

	return kvs, nil
}
