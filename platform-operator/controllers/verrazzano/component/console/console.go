// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package console

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	effectiveCR := ctx.EffectiveCR()
	// Environment name
	envName := vzconfig.GetEnvName(effectiveCR)
	// DNS Suffix
	dnsSuffix, err := vzconfig.GetDNSSuffix(ctx.Client(), effectiveCR)
	if err != nil {
		return nil, err
	}

	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, err
	}
	images, err := bomFile.BuildImageOverrides("verrazzano")
	if err != nil {
		return kvs, err
	}
	var imageName, imageTag string
	for _, image := range images {
		if image.Key == "console.imageName" {
			imageName = image.Value
		} else if image.Key == "console.imageVersion" {
			imageTag = image.Value
		}
	}
	if imageTag == "" || imageName == "" {
		return kvs, ctx.Log().ErrorfNewErr("Failed to construct console image from BOM")
	}

	return append(kvs,
		bom.KeyValue{
			Key:   "config.dnsSuffix",
			Value: dnsSuffix,
		},
		bom.KeyValue{
			Key:   "config.envName",
			Value: envName,
		},
		bom.KeyValue{
			Key:   "imageName",
			Value: imageName,
		},
		bom.KeyValue{
			Key:   "imageTag",
			Value: imageTag,
		}), nil
}

func isConsoleReady(ctx spi.ComponentContext) bool {
	return ready.DeploymentsAreReady(
		ctx.Log(),
		ctx.Client(),
		[]types.NamespacedName{
			{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
			},
		},
		1,
		fmt.Sprintf("Component %s", ctx.GetComponent()))
}

func preHook(ctx spi.ComponentContext) error {
	namespacedName := types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}
	objects := []client.Object{
		&corev1.ServiceAccount{},
		&corev1.Service{},
		&appsv1.Deployment{},
	}

	// namespaced resources
	for _, obj := range objects {
		if _, err := common.RemoveResourcePolicyAnnotation(ctx.Client(), obj, namespacedName); err != nil {
			return err
		}
	}
	return nil
}
