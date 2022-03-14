// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Operator Lifecycle Manager
package olm

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

var (
	// For Unit test purposes
	writeFileFunc        = ioutil.WriteFile
	tmpFilePrefix        = "operator-lifecycle-manager-overrides-"
	tmpSuffix            = "yaml"
	tmpFileCreatePattern = tmpFilePrefix + "*." + tmpSuffix
	// tmpFileCleanPattern  = tmpFilePrefix + ".*\\." + tmpSuffix
)

// func resetWriteFileFunc() {
// 	writeFileFunc = ioutil.WriteFile
// }

// isOLMReady checks if the OLM deployments are in the ready state
func isOLMReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{Name: ComponentName, Namespace: ComponentNamespace},
		{Name: CatalogComponentName, Namespace: ComponentNamespace},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// AppendOverrides builds the set of verrazzano-authproxy overrides for the helm install
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Overrides object to store any user overrides
	overrides := olmValues{}

	// Image name and version
	err := loadImageSettings(ctx, &overrides)
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

// loadImageSettings loads the override values for the image name and version
func loadImageSettings(ctx spi.ComponentContext, overrides *olmValues) error {
	// Full image name
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return err
	}
	images, err := bomFile.BuildImageOverrides("operator-lifecycle-manager")
	if err != nil {
		return err
	}

	for _, image := range images {
		if image.Key == "olm.imageName" {
			overrides.ImageName = image.Value
		} else if image.Key == "olm.imageVersion" {
			overrides.ImageVersion = image.Value
		}
	}
	if len(overrides.ImageName) == 0 {
		return ctx.Log().ErrorNewErr("Failed to find olm.imageName in BOM")
	}
	if len(overrides.ImageVersion) == 0 {
		return ctx.Log().ErrorNewErr("Failed to find olm.imageVersion in BOM")
	}

	return nil
}

// Function to generate overrides file
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

// createNamespace function creates a namespace with the given name
func (c olmComponent) createOrUpdateNamespace(compContext spi.ComponentContext, namespaceName string) error {
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		return nil
	}); err != nil {
		return compContext.Log().ErrorfNewErr("Failed to create or update the operator-lifecycle-manager namespace: %v", err)
	}
	return nil
}

// applyManifest applies the CRDs required for operator-lifecycle-manager
func (c olmComponent) applyManifest(compContext spi.ComponentContext) error {
	crdManifestDir := filepath.Join(config.GetThirdPartyManifestsDir(), crdDirectory)
	// Exclude the input file, since it will be parsed into individual documents
	// File containing OLM CRDs
	crdFilePath := filepath.Join(crdManifestDir, crdFile)

	// Apply the CRD Manifest for OLM
	if err := k8sutil.NewYAMLApplier(compContext.Client(), "").ApplyF(crdFilePath); err != nil {
		return compContext.Log().ErrorfNewErr("Failed applying CRD Manifests for OLM: %v", err)

	}
	return nil
}
