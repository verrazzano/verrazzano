// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmo

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
)

// isVmoReady checks to see if the VMO component is in ready state
func isVmoReady(context spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, prefix)
}

// appendVmoOverrides appends overrides for the VMO component
func appendVmoOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	vzkvs, err := appendInitImageOverrides(kvs)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to append monitoring init image overrides: %v", err)
	}

	effectiveCR := ctx.EffectiveCR()

	// Get the dnsSuffix override
	dnsSuffix, err := vzconfig.GetDNSSuffix(ctx.Client(), effectiveCR)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed getting DNS suffix: %v", err)
	}
	kvs = append(kvs, bom.KeyValue{Key: "config.dnsSuffix", Value: dnsSuffix})

	// Get the env name
	envName := vzconfig.GetEnvName(effectiveCR)

	kvs = append(kvs, bom.KeyValue{Key: "config.envName", Value: envName})
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

// ExportVmoHelmChart adds necessary annotations to verrazzano-monitoring-operator objects which allows them to be
// managed by the verrazzano-monitoring-operator helm chart.  This is needed for the case when VMO was
// previously installed by the verrazzano helm charrt.
func ExportVMOHelmChart(ctx spi.ComponentContext) error {
	releaseName := types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}
	managedResources := getHelmManagedResources()
	for _, managedResource := range managedResources {
		if _, err := common.AssociateHelmObject(ctx.Client(), managedResource.Obj, releaseName, managedResource.NamespacedName, true); err != nil {
			return err
		}
	}

	return nil
}

// reassociateResources updates the resources to ensure they are managed by the VMO release/component.  The resource policy
// annotation is removed to ensure that helm manages the lifecycle of the resources (the resource policy annotation is
// added to ensure the resources are disassociated from the VZ chart which used to manage these resources)
func reassociateResources(ctx spi.ComponentContext) error {
	managedResources := getHelmManagedResources()
	for _, managedResource := range managedResources {
		if _, err := common.RemoveResourcePolicyAnnotation(ctx.Client(), managedResource.Obj, managedResource.NamespacedName); err != nil {
			return err
		}
	}

	return nil
}

// getHelmManagedResources returns a list of resource types and their namespaced names that are managed by the
// VMO helm chart
func getHelmManagedResources() []common.HelmManagedResource {
	return []common.HelmManagedResource{
		{Obj: &corev1.ConfigMap{}, NamespacedName: types.NamespacedName{Name: "verrazzano-monitoring-operator-config", Namespace: ComponentNamespace}},
		{Obj: &corev1.Service{}, NamespacedName: types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}},
		{Obj: &corev1.ServiceAccount{}, NamespacedName: types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}},
		{Obj: &rbacv1.ClusterRole{}, NamespacedName: types.NamespacedName{Name: "verrazzano-monitoring-operator-cluster-role"}},
		{Obj: &rbacv1.ClusterRole{}, NamespacedName: types.NamespacedName{Name: "vmi-cluster-role-default"}},
		{Obj: &rbacv1.ClusterRole{}, NamespacedName: types.NamespacedName{Name: "verrazzano-monitoring-operator-get-nodes"}},
		{Obj: &rbacv1.ClusterRoleBinding{}, NamespacedName: types.NamespacedName{Name: "verrazzano-monitoring-operator-cluster-role-binding"}},
		{Obj: &rbacv1.ClusterRoleBinding{}, NamespacedName: types.NamespacedName{Name: "verrazzano-monitoring-operator-cluster-role-default-binding"}},
		{Obj: &rbacv1.ClusterRoleBinding{}, NamespacedName: types.NamespacedName{Name: "verrazzano-monitoring-operator-get-nodes"}},
	}
}
