// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"context"
	"fmt"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// loggingPreInstall copies logging secrets from the verrazzano-install namespace to the verrazzano-system namespace
func loggingPreInstall(ctx spi.ComponentContext) error {
	if vzconfig.IsFluentdEnabled(ctx.EffectiveCR()) {
		// If fluentd is enabled, copy any custom secrets
		fluentdConfig := ctx.EffectiveCR().Spec.Components.Fluentd
		if fluentdConfig != nil {
			// Copy the internal Elasticsearch secret
			if len(fluentdConfig.ElasticsearchURL) > 0 && fluentdConfig.ElasticsearchSecret != globalconst.VerrazzanoESInternal {
				if err := copySecret(ctx, fluentdConfig.ElasticsearchSecret, "custom Elasticsearch"); err != nil {
					return err
				}
			}
			// Copy the OCI API secret
			if fluentdConfig.OCI != nil && len(fluentdConfig.OCI.APISecret) > 0 {
				if err := copySecret(ctx, fluentdConfig.OCI.APISecret, "OCI API"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// copySecret copies a secret from the verrazzano-install namespace to the verrazzano-system namespace. If
// the target secret already exists, then it will be updated if necessary.
func copySecret(ctx spi.ComponentContext, secretName string, logMsg string) error {
	vzLog := ctx.Log()
	vzLog.Debugf("Copying %s secret %s to %s namespace", logMsg, secretName, globalconst.VerrazzanoSystemNamespace)

	targetSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
	}
	opResult, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &targetSecret, func() error {
		sourceSecret := corev1.Secret{}
		nsn := types.NamespacedName{Name: secretName, Namespace: constants.VerrazzanoInstallNamespace}
		if err := ctx.Client().Get(context.TODO(), nsn, &sourceSecret); err != nil {
			return err
		}
		targetSecret.Type = sourceSecret.Type
		targetSecret.Immutable = sourceSecret.Immutable
		targetSecret.StringData = sourceSecret.StringData
		targetSecret.Data = sourceSecret.Data
		return nil
	})

	vzLog.Debugf("Copy %s secret result: %s", logMsg, opResult)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctx.Log().ErrorfNewErr("Failed in create/update for copysecret: %v", err)
		}
		return vzLog.ErrorfNewErr("Failed, the %s secret %s not found in namespace %s", logMsg, secretName, constants.VerrazzanoInstallNamespace)
	}

	return nil
}

// isFluentdReady Fluentd component ready-check
func isFluentdReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())

	// Check daemonsets
	var daemonsets []types.NamespacedName
	if vzconfig.IsFluentdEnabled(ctx.EffectiveCR()) && getProfile(ctx.EffectiveCR()) != vzapi.ManagedCluster {
		daemonsets = append(daemonsets,
			types.NamespacedName{
				Name:      DaemonsetName,
				Namespace: ComponentNamespace,
			})
		if status.DaemonSetsAreReady(ctx.Log(), ctx.Client(), daemonsets, 1, prefix) {
			return true
		}
	}
	return false
}

// fluentdPreUpgrade contains code that is run prior to helm upgrade for the Verrazzano Fluentd helm chart
func fluentdPreUpgrade(ctx spi.ComponentContext, namespace string) error {
	return fixupFluentdDaemonset(ctx.Log(), ctx.Client(), namespace)
}

// This function is used to fixup the fluentd daemonset on a managed cluster so that helm upgrade of Verrazzano does
// not fail.  Prior to Verrazzano v1.0.1, the mcagent would change the environment variables CLUSTER_NAME and
// ELASTICSEARCH_URL on a managed cluster to use valueFrom (from a secret) instead of using a Value. The helm chart
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

	// Check if env variables CLUSTER_NAME and ELASTICSEARCH_URL are using valueFrom.
	clusterNameIndex := -1
	elasticURLIndex := -1
	for i, env := range daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env {
		if env.Name == constants.ClusterNameEnvVar && env.ValueFrom != nil {
			clusterNameIndex = i
			continue
		}
		if env.Name == constants.ElasticsearchURLEnvVar && env.ValueFrom != nil {
			elasticURLIndex = i
		}
	}

	// If valueFrom is not being used then we do not need to fix the env variables
	if clusterNameIndex == -1 && elasticURLIndex == -1 {
		return nil
	}

	// Get the secret containing managed cluster name and Elasticsearch URL
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

	// The secret must contain the Elasticsearch endpoint's URL
	elasticsearchURL, ok := sec.Data[constants.ElasticsearchURLData]
	if !ok {
		return log.ErrorfNewErr("Failed, the secret named %s in namespace %s is missing the required field %s", sec.Name, sec.Namespace, constants.ElasticsearchURLData)
	}

	// Update the daemonset to use a Value instead of the valueFrom
	if clusterNameIndex != -1 {
		daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env[clusterNameIndex].Value = string(clusterName)
		daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env[clusterNameIndex].ValueFrom = nil
	}
	if elasticURLIndex != -1 {
		daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env[elasticURLIndex].Value = string(elasticsearchURL)
		daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env[elasticURLIndex].ValueFrom = nil
	}
	log.Debug("Updating fluentd daemonset to use valueFrom instead of Value for CLUSTER_NAME and ELASTICSEARCH_URL environment variables")
	err = client.Update(context.TODO(), &daemonSet)
	return err
}

// fluentdPreHelmOps ensures the fluentd associated resources are managed its helm install/upgrade executions by
// ensuring the resource policy of "keep" is removed (if it remains then helm is unable to delete these resources and
// they will become orphaned)
func fluentdPreHelmOps(ctx spi.ComponentContext) error {
	return reassociateResources(ctx.Client())
}

// reassociateResources updates the resources to ensure they are managed by this release/component.  The resource policy
// annotation is removed to ensure that helm manages the lifecycle of the resources (the resource policy annotation is
// added to ensure the resources are disassociated from the VZ chart which used to manage these resources)
func reassociateResources(cli clipkg.Client) error {
	namespacedName := types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}
	name := types.NamespacedName{Name: ComponentName}
	objects := []controllerutil.Object{
		&corev1.ServiceAccount{},
		&corev1.Service{},
		&appsv1.DaemonSet{},
	}

	noNamespaceObjects := []controllerutil.Object{
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
	}

	// namespaced resources
	for _, obj := range objects {
		if _, err := common.RemoveResourcePolicyAnnotation(cli, obj, namespacedName); err != nil {
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

// GetProfile Returns the configured profile name, or "prod" if not specified in the configuration
func getProfile(vz *vzapi.Verrazzano) vzapi.ProfileType {
	profile := vz.Spec.Profile
	if len(profile) == 0 {
		profile = vzapi.Prod
	}
	return profile
}
