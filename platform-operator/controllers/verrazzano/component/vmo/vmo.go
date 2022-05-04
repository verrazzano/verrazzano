// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmo

import (
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	"k8s.io/apimachinery/pkg/types"
)

// isVMOReady checks to see if the VMO component is in ready state
func isVMOReady(context spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, prefix)
}

// appendVMOOverrides appends overrides for the VMO component
func appendVMOOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	vzkvs, err := appendInitImageOverrides(kvs)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to append monitoring init image overrides: %v", err)
	}

	effectiveCR := ctx.EffectiveCR()

	// If NGINX is enabled, then get the values used to build up the defaultIngressTargetDNSName
	// value in the VMO config map.  Otherwise, the value is not set in the VMO config map.
	if vzconfig.IsNGINXEnabled(effectiveCR) {
		// Get the dnsSuffix override
		dnsSuffix, err := vzconfig.GetDNSSuffix(ctx.Client(), effectiveCR)
		if err != nil {
			return kvs, ctx.Log().ErrorfNewErr("Failed getting DNS suffix: %v", err)
		}
		kvs = append(kvs, bom.KeyValue{Key: "config.dnsSuffix", Value: dnsSuffix})

		// Get the env name
		envName := vzconfig.GetEnvName(effectiveCR)

		kvs = append(kvs, bom.KeyValue{Key: "config.envName", Value: envName})
	}

	// Override the OIDC auth enabled value if Auth Proxy is disabled
	if !vzconfig.IsAuthProxyEnabled(effectiveCR) {
		kvs = append(kvs, bom.KeyValue{Key: "monitoringOperator.oidcAuthEnabled", Value: "false"})
	}

	kvs = append(kvs, vzkvs...)
	return kvs, nil
}

// append the monitoring-init-images overrides
func appendInitImageOverrides(kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, err
	}

	imageOverrides, err := bomFile.BuildImageOverrides("monitoring-init-images")
	if err != nil {
		return kvs, err
	}

	kvs = append(kvs, imageOverrides...)
	return kvs, nil
}
