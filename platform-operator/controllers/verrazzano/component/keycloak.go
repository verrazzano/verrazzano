// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"bytes"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"text/template"

	"go.uber.org/zap"
)

// Define the keylcloak Key:Value pair for init container.
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

// appendKeycloakOverrides appends the Keycloak theme for the Key keycloak.extraInitContainers.
// A go template is used to replace the image in the init container spec.
func appendKeycloakOverrides(_ *zap.SugaredLogger, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
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

	return kvs, nil
}
