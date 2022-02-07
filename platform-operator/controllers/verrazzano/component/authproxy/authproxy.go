// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package authproxy

import (
	"fmt"

	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"k8s.io/apimachinery/pkg/types"
)

// IsReady checks if the application operator deployment is ready
func IsReady(ctx spi.ComponentContext, name string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: ComponentName, Namespace: ComponentNamespace},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// AppendOverrides builds the set of verrazzano-authproxy overrides for the helm install
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	effectiveCR := ctx.EffectiveCR()

	// Environment name
	kvs = append(kvs, bom.KeyValue{Key: "config.envName", Value: vzconfig.GetEnvName(effectiveCR)})

	// Full image name
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}
	images, err := bomFile.BuildImageOverrides("verrazzano")
	if err != nil {
		return nil, err
	}
	imageName := ""
	imageVersion := ""
	for _, image := range images {
		if image.Key == "api.imageName" {
			imageName = image.Value
		} else if image.Key == "api.imageVersion" {
			imageVersion = image.Value
		}
	}
	if len(imageName) == 0 {
		return nil, ctx.Log().ErrorNewErr("Failed to find api.imageName in BOM")
	}
	if len(imageVersion) == 0 {
		return nil, ctx.Log().ErrorNewErr("Failed to find api.imageVersion in BOM")
	}
	kvs = append(kvs, bom.KeyValue{Key: "imageName", Value: imageName})
	kvs = append(kvs, bom.KeyValue{Key: "imageVersion", Value: imageVersion})

	// DNS Suffix
	dnsSuffix, err := vzconfig.GetDNSSuffix(ctx.Client(), effectiveCR)
	if err != nil {
		return kvs, err
	}
	kvs = append(kvs, bom.KeyValue{Key: "config.dnsSuffix", Value: dnsSuffix})

	return kvs, nil
}
