// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	"os"
)

// appendCertManagerOverrides appends the Keycloak theme for the key keycloak.extraInitContainers.
// A go template is used to replace the image in the init container spec.
func appendCertManagerOverrides(_ *zap.SugaredLogger, _ string, _ string, _ string, kvs []keyValue) ([]keyValue, error) {

	// Only override the Cert-Manager additional images if the registry override is set, otherwise
	// defer to the charts.  We can investigate uniformly overriding things later.
	registry := os.Getenv(constants.RegistryOverrideEnvVar)
	if len(registry) == 0 {
		return []keyValue{}, nil
	}

	// Create a Bom and get the key value overrides
	bom, err := NewBom(DefaultBomFilePath())
	if err != nil {
		return nil, err
	}

	// Get additional cert-manager images
	images, err := bom.buildImageOverrides("additional-cert-manager")
	if err != nil {
		return nil, err
	}
	// Return a new key:value pair with the rendered value
	kvs = append(kvs, images...)

	return kvs, nil
}
