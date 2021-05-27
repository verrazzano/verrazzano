// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	"os"
)

const istioGlobalHubKey = "global.hub"

// appendIstioOverrides appends the Keycloak theme for the key keycloak.extraInitContainers.
// A go template is used to replace the image in the init container spec.
func appendIstioOverrides(_ *zap.SugaredLogger, _ string, _ string, _ string, kvs []keyValue) ([]keyValue, error) {

	// Get the registryOverride ENV override, if it doesn't exist use the default
	registryOverride := os.Getenv(constants.RegistryOverrideEnvVar)
	if len(registryOverride) > 0 {
		// Return a new key:value pair with the rendered value
		kvs = append(kvs, keyValue{
			key:   istioGlobalHubKey,
			value: registryOverride,
		})
	}

	return kvs, nil
}
