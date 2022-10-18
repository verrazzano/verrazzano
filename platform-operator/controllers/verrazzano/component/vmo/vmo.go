// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmo

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	corev1 "k8s.io/api/core/v1"
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
	return ready.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, prefix)
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

// retainPrometheusPersistentVolume locates the persistent volume associated with the Prometheus persistent volume claim
// and sets the reclaim policy to "retain" so that it can be migrated to the new Prometheus Operator-managed Prometheus.
// When the VMO is upgraded, it will remove the existing Prometheus deployment and persistent volume claim, so we need
// to retain the volume so it can be migrated.
func retainPrometheusPersistentVolume(ctx spi.ComponentContext) error {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := ctx.Client().Get(context.TODO(), types.NamespacedName{Name: constants.VMISystemPrometheusVolumeClaim, Namespace: ComponentNamespace}, pvc); err != nil {
		// no pvc so just log it and there's nothing left to do
		ctx.Log().Debugf("Did not find pvc %s, skipping volume migration: %v", constants.VMISystemPrometheusVolumeClaim, err)
		return nil
	}

	ctx.Log().Infof("Updating persistent volume associated with pvc %s so that the volume can be migrated", constants.VMISystemPrometheusVolumeClaim)

	pvName := pvc.Spec.VolumeName
	pv := &corev1.PersistentVolume{}
	if err := ctx.Client().Get(context.TODO(), types.NamespacedName{Name: pvName}, pv); err != nil {
		return ctx.Log().ErrorfNewErr("Failed fetching persistent volume associated with pvc %s: %v", constants.VMISystemPrometheusVolumeClaim, err)
	}

	// set the reclaim policy on the pv to retain so that it does not get deleted when the VMO-managed Prometheus is removed
	oldReclaimPolicy := pv.Spec.PersistentVolumeReclaimPolicy
	pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain

	// add labels to the pv - one that allows the new Prometheus to select the volume and another that captures
	// the old reclaim policy so we can set it back once the volume is bound
	if pv.Labels == nil {
		pv.Labels = make(map[string]string)
	}
	pv.Labels[constants.StorageForLabel] = constants.PrometheusStorageLabelValue
	pv.Labels[constants.OldReclaimPolicyLabel] = string(oldReclaimPolicy)

	if err := ctx.Client().Update(context.TODO(), pv); err != nil {
		return ctx.Log().ErrorfNewErr("Failed updating persistent volume associated with pvc %s: %v", constants.VMISystemPrometheusVolumeClaim, err)
	}
	return nil
}
