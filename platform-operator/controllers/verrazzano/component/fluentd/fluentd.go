// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"context"
	"fmt"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Fluentd ConfigMap names
	fluentdInit     = "fluentd-init"
	fluentdConfig   = "fluentd-config"
	fluentdEsConfig = "fluentd-es-config"
)

// checkSecretExists whether verrazzano-es-internal secret exists. Return error if secret does not exist.
func checkSecretExists(ctx spi.ComponentContext) error {
	if vzconfig.IsKeycloakEnabled(ctx.EffectiveCR()) {
		// Check verrazzano-es-internal Secret. return error which will cause requeue
		secret := &corev1.Secret{}
		err := ctx.Client().Get(context.TODO(), clipkg.ObjectKey{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      globalconst.VerrazzanoESInternal,
		}, secret)

		if err != nil {
			if errors.IsNotFound(err) {
				ctx.Log().Progressf("Component Fluentd waiting for the secret %s/%s to exist",
					constants.VerrazzanoSystemNamespace, globalconst.VerrazzanoESInternal)
				return ctrlerrors.RetryableError{Source: ComponentName}
			}
			ctx.Log().Errorf("Component Fluentd failed to get the secret %s/%s: %v",
				constants.VerrazzanoSystemNamespace, globalconst.VerrazzanoESInternal, err)
			return err
		}
	}
	return nil
}

// loggingPreInstall copies logging secrets from the verrazzano-install namespace to the verrazzano-system namespace
func loggingPreInstall(ctx spi.ComponentContext) error {
	if vzconfig.IsFluentdEnabled(ctx.EffectiveCR()) {
		// If fluentd is enabled, copy any custom secrets
		fluentdConfig := ctx.EffectiveCR().Spec.Components.Fluentd
		if fluentdConfig != nil {
			// Copy the internal Opensearch secret
			if len(fluentdConfig.ElasticsearchURL) > 0 && fluentdConfig.ElasticsearchSecret != globalconst.VerrazzanoESInternal {
				if err := common.CopySecret(ctx, fluentdConfig.ElasticsearchSecret, constants.VerrazzanoSystemNamespace, "custom Elasticsearch"); err != nil {
					return err
				}
			}
			// Copy the OCI API secret
			if fluentdConfig.OCI != nil && len(fluentdConfig.OCI.APISecret) > 0 {
				if err := common.CopySecret(ctx, fluentdConfig.OCI.APISecret, constants.VerrazzanoSystemNamespace, "OCI API"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// isFluentdReady Fluentd component ready-check
func isFluentdReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())

	// Check daemonsets
	var daemonsets []types.NamespacedName
	if vzconfig.IsFluentdEnabled(ctx.EffectiveCR()) {
		daemonsets = append(daemonsets,
			types.NamespacedName{
				Name:      ComponentName,
				Namespace: ComponentNamespace,
			})
		return ready.DaemonSetsAreReady(ctx.Log(), ctx.Client(), daemonsets, 1, prefix)
	}
	return false
}

// fluentdPreUpgrade contains code that is run prior to helm upgrade for the Verrazzano Fluentd helm chart
func fluentdPreUpgrade(ctx spi.ComponentContext, namespace string) error {
	return fixupFluentdDaemonset(ctx.Log(), ctx.Client(), namespace)
}

// This function is used to fixup the fluentd daemonset on a managed cluster so that helm upgrade of Verrazzano does
// not fail.  Prior to Verrazzano v1.0.1, the mcagent would change the environment variables CLUSTER_NAME and
// OPENSEARCH_URL on a managed cluster to use valueFrom (from a secret) instead of using a Value. The helm chart
// template for the fluentd daemonset expects a Value.
func fixupFluentdDaemonset(log vzlog.VerrazzanoLogger, client clipkg.Client, namespace string) error {
	// Get the fluentd daemonset resource
	fluentdNamespacedName := types.NamespacedName{Name: globalconst.FluentdDaemonSetName, Namespace: namespace}
	daemonSet := appsv1.DaemonSet{}
	err := client.Get(context.TODO(), fluentdNamespacedName, &daemonSet)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return log.ErrorfNewErr("Failed to find the fluentd DaemonSet %s, %v", daemonSet.Name, err)
	}

	// Find the fluent container and save it's container index
	fluentdIndex := -1
	for i, container := range daemonSet.Spec.Template.Spec.Containers {
		if container.Name == "fluentd" {
			fluentdIndex = i
			break
		}
	}
	if fluentdIndex == -1 {
		return log.ErrorfNewErr("Failed, fluentd container not found in fluentd daemonset: %s", daemonSet.Name)
	}

	// Check if env variables CLUSTER_NAME and OPENSEARCH_URL are using valueFrom.
	clusterNameIndex := -1
	openURLIndex := -1
	for i, env := range daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env {
		if env.Name == constants.ClusterNameEnvVar && env.ValueFrom != nil {
			clusterNameIndex = i
			continue
		}
		if env.Name == constants.OpensearchURLEnvVar && env.ValueFrom != nil {
			openURLIndex = i
		}
	}

	// If valueFrom is not being used then we do not need to fix the env variables
	if clusterNameIndex == -1 && openURLIndex == -1 {
		return nil
	}

	// Get the secret containing managed cluster name and Opensearch URL
	secretNamespacedName := types.NamespacedName{Name: constants.MCRegistrationSecret, Namespace: namespace}
	sec := corev1.Secret{}
	err = client.Get(context.TODO(), secretNamespacedName, &sec)
	if err != nil {
		return err
	}

	// The secret must contain a cluster name
	clusterName, ok := sec.Data[constants.ClusterNameData]
	if !ok {
		return log.ErrorfNewErr("Failed, the secret named %s in namespace %s is missing the required field %s", sec.Name, sec.Namespace, constants.ClusterNameData)
	}

	// The secret must contain the Opensearch endpoint's URL
	opensearchURL, ok := sec.Data[constants.OpensearchURLData]
	if !ok {
		return log.ErrorfNewErr("Failed, the secret named %s in namespace %s is missing the required field %s", sec.Name, sec.Namespace, constants.OpensearchURLData)
	}

	// Update the daemonset to use a Value instead of the valueFrom
	if clusterNameIndex != -1 {
		daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env[clusterNameIndex].Value = string(clusterName)
		daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env[clusterNameIndex].ValueFrom = nil
	}
	if openURLIndex != -1 {
		daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env[openURLIndex].Value = string(opensearchURL)
		daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env[openURLIndex].ValueFrom = nil
	}
	log.Debug("Updating fluentd daemonset to use valueFrom instead of Value for CLUSTER_NAME and OPENSEARCH_URL environment variables")
	err = client.Update(context.TODO(), &daemonSet)
	return err
}

// ReassociateResources updates the resources to ensure they are managed by this release/component.  The resource policy
// annotation is removed to ensure that helm manages the lifecycle of the resources (the resource policy annotation is
// added to ensure the resources are disassociated from the VZ chart which used to manage these resources)
func ReassociateResources(cli clipkg.Client) error {
	namespacedName := types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}
	name := types.NamespacedName{Name: ComponentName}
	objects := []clipkg.Object{
		&corev1.ServiceAccount{},
		&appsv1.DaemonSet{},
	}

	noNamespaceObjects := []clipkg.Object{
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
	}

	// namespaced resources
	for _, obj := range objects {
		if _, err := common.RemoveResourcePolicyAnnotation(cli, obj, namespacedName); err != nil {
			return err
		}
	}

	// additional namespaced resources managed by this helm chart
	helmManagedResources := GetHelmManagedResources()
	for _, managedResoure := range helmManagedResources {
		if _, err := common.RemoveResourcePolicyAnnotation(cli, managedResoure.Obj, managedResoure.NamespacedName); err != nil {
			return err
		}
	}

	// cluster resources
	for _, obj := range noNamespaceObjects {
		if _, err := common.RemoveResourcePolicyAnnotation(cli, obj, name); err != nil {
			return err
		}
	}
	return nil
}

// GetHelmManagedResources returns a list of extra resource types and their namespaced names that are managed by the
// fluentd helm chart
func GetHelmManagedResources() []common.HelmManagedResource {
	return []common.HelmManagedResource{
		{Obj: &corev1.ConfigMap{}, NamespacedName: types.NamespacedName{Name: fluentdInit, Namespace: ComponentNamespace}},
		{Obj: &corev1.ConfigMap{}, NamespacedName: types.NamespacedName{Name: fluentdConfig, Namespace: ComponentNamespace}},
		{Obj: &corev1.ConfigMap{}, NamespacedName: types.NamespacedName{Name: fluentdEsConfig, Namespace: ComponentNamespace}},
	}
}

// getFluentdManagedResources returns a list of resource types and their namespaced names that are managed by the
// Fluent helm chart
func getFluentdManagedResources() []common.HelmManagedResource {
	return []common.HelmManagedResource{
		{Obj: &rbacv1.ClusterRole{}, NamespacedName: types.NamespacedName{Name: ComponentName}},
		{Obj: &rbacv1.ClusterRoleBinding{}, NamespacedName: types.NamespacedName{Name: ComponentName}},
		{Obj: &corev1.ConfigMap{}, NamespacedName: types.NamespacedName{Name: "fluentd-config", Namespace: ComponentNamespace}},
		{Obj: &corev1.ConfigMap{}, NamespacedName: types.NamespacedName{Name: "fluentd-es-config", Namespace: ComponentNamespace}},
		{Obj: &corev1.ConfigMap{}, NamespacedName: types.NamespacedName{Name: "fluentd-init", Namespace: ComponentNamespace}},
		{Obj: &appsv1.DaemonSet{}, NamespacedName: types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}},
		{Obj: &corev1.Service{}, NamespacedName: types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}},
		{Obj: &corev1.ServiceAccount{}, NamespacedName: types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}},
	}
}
