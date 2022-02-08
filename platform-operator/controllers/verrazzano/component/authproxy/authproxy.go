// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"sigs.k8s.io/yaml"
)

const (
	keycloakInClusterURL = "keycloak-http.keycloak.svc.cluster.local"
	tmpFilePrefix        = "verrazzano-overrides-"
	tmpSuffix            = "yaml"
	tmpFileCreatePattern = tmpFilePrefix + "*." + tmpSuffix
	tmpFileCleanPattern  = tmpFilePrefix + ".*\\." + tmpSuffix
)

var (
	// For Unit test purposes
	writeFileFunc = ioutil.WriteFile
)

func resetWriteFileFunc() {
	writeFileFunc = ioutil.WriteFile
}

// AppendOverrides builds the set of verrazzano-authproxy overrides for the helm install
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	effectiveCR := ctx.EffectiveCR()

	// Overrides object to store any user overrides
	overrides := authProxyValues{}

	// Environment name
	overrides.Config = &configSettings{
		EnvName: vzconfig.GetEnvName(effectiveCR),
	}

	// DNS Suffix
	dnsSuffix, err := vzconfig.GetDNSSuffix(ctx.Client(), effectiveCR)
	if err != nil {
		return nil, err
	}
	overrides.Config.DNSSuffix = dnsSuffix

	overrides.Proxy = &proxySettings{
		OidcProviderHost:          fmt.Sprintf("keycloak.%s.%s", overrides.Config.EnvName, dnsSuffix),
		OidcProviderHostInCluster: keycloakInClusterURL,
	}

	// Image name and version
	err = getImageSettings(ctx, &overrides)
	if err != nil {
		return nil, err
	}

	// Write the overrides file to a temp dir and add a helm file override argument
	overridesFileName, err := generateOverridesFile(ctx, &overrides)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed generating AuthProxy overrides file: %v", err)
	}

	// Append any installArgs overrides in vzkvs after the file overrides to ensure precedence of those
	kvs = append(kvs, bom.KeyValue{Value: overridesFileName, IsFile: true})

	return kvs, nil
}

// getImageSettings sets the override values for the image name and version
func getImageSettings(ctx spi.ComponentContext, overrides *authProxyValues) error {
	// Full image name
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return err
	}
	images, err := bomFile.BuildImageOverrides("verrazzano")
	if err != nil {
		return err
	}

	for _, image := range images {
		if image.Key == "api.imageName" {
			overrides.ImageName = image.Value
		} else if image.Key == "api.imageVersion" {
			overrides.ImageVersion = image.Value
		}
	}
	if len(overrides.ImageName) == 0 {
		return ctx.Log().ErrorNewErr("Failed to find api.imageName in BOM")
	}
	if len(overrides.ImageVersion) == 0 {
		return ctx.Log().ErrorNewErr("Failed to find api.imageVersion in BOM")
	}

	return nil
}

// TODO: move this to a common package?
func generateOverridesFile(ctx spi.ComponentContext, overrides interface{}) (string, error) {
	bytes, err := yaml.Marshal(overrides)
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp(os.TempDir(), tmpFileCreatePattern)
	if err != nil {
		return "", err
	}

	overridesFileName := file.Name()
	if err := writeFileFunc(overridesFileName, bytes, fs.ModeAppend); err != nil {
		return "", err
	}
	ctx.Log().Debugf("Verrazzano install overrides file %s contents: %s", overridesFileName, string(bytes))
	return overridesFileName, nil
}
