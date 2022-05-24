// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/namespace"
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
	"sigs.k8s.io/yaml"
)

// ComponentName is the name of the component

const (
	tmpFilePrefix        = "verrazzano-overrides-"
	tmpSuffix            = "yaml"
	tmpFileCreatePattern = tmpFilePrefix + "*." + tmpSuffix
	tmpFileCleanPattern  = tmpFilePrefix + ".*\\." + tmpSuffix

	fluentDaemonset       = "fluentd"
	nodeExporterDaemonset = "node-exporter"

	prometheusDeployment        = "vmi-system-prometheus-0"
	verrazzanoConsoleDeployment = "verrazzano-console"
)

var (
	// For Unit test purposes
	writeFileFunc = ioutil.WriteFile
)

func resetWriteFileFunc() {
	writeFileFunc = ioutil.WriteFile
}

// resolveVerrazzanoNamespace will return the default Verrazzano system namespace unless the namespace is specified
func resolveVerrazzanoNamespace(ns string) string {
	if len(ns) > 0 && ns != "default" {
		return ns
	}
	return globalconst.VerrazzanoSystemNamespace
}

// isVerrazzanoReady Verrazzano component ready-check
func isVerrazzanoReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())

	// First, check deployments
	var deployments []types.NamespacedName
	if vzconfig.IsConsoleEnabled(ctx.EffectiveCR()) {
		deployments = append(deployments,
			types.NamespacedName{
				Name:      verrazzanoConsoleDeployment,
				Namespace: ComponentNamespace,
			})
	}

	if vzconfig.IsPrometheusEnabled(ctx.EffectiveCR()) {
		deployments = append(deployments,
			types.NamespacedName{
				Name:      prometheusDeployment,
				Namespace: ComponentNamespace,
			})
	}

	if !status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix) {
		return false
	}

	// Finally, check daemonsets
	var daemonsets []types.NamespacedName
	if vzconfig.IsPrometheusEnabled(ctx.EffectiveCR()) {
		daemonsets = append(daemonsets,
			types.NamespacedName{
				Name:      nodeExporterDaemonset,
				Namespace: globalconst.VerrazzanoMonitoringNamespace,
			})
	}
	if vzconfig.IsFluentdEnabled(ctx.EffectiveCR()) && getProfile(ctx.EffectiveCR()) != vzapi.ManagedCluster {
		daemonsets = append(daemonsets,
			types.NamespacedName{
				Name:      fluentDaemonset,
				Namespace: ComponentNamespace,
			})
	}
	if !status.DaemonSetsAreReady(ctx.Log(), ctx.Client(), daemonsets, 1, prefix) {
		return false
	}
	return common.IsVMISecretReady(ctx)
}

// doesPromExist is the verrazzano IsInstalled check
func doesPromExist(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	deploy := []types.NamespacedName{{
		Name:      prometheusDeployment,
		Namespace: ComponentNamespace,
	}}
	return status.DoDeploymentsExist(ctx.Log(), ctx.Client(), deploy, 1, prefix)
}

// VerrazzanoPreUpgrade contains code that is run prior to helm upgrade for the Verrazzano helm chart
func verrazzanoPreUpgrade(ctx spi.ComponentContext, namespace string) error {
	if err := importToHelmChart(ctx.Client()); err != nil {
		return err
	}
	if err := exportFromHelmChart(ctx.Client()); err != nil {
		return err
	}
	if err := common.EnsureVMISecret(ctx.Client()); err != nil {
		return err
	}
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

func createAndLabelNamespaces(ctx spi.ComponentContext) error {
	if err := LabelKubeSystemNamespace(ctx.Client()); err != nil {
		return err
	}
	if err := common.CreateAndLabelVMINamespaces(ctx); err != nil {
		return err
	}
	if err := namespace.CreateVerrazzanoMultiClusterNamespace(ctx.Client()); err != nil {
		return err
	}
	if vzconfig.IsVMOEnabled(ctx.EffectiveCR()) {
		// If the monitoring operator is enabled, create the monitoring namespace and copy the image pull secret
		if err := namespace.CreateVerrazzanoMonitoringNamespace(ctx.Client()); err != nil {
			return ctx.Log().ErrorfNewErr("Failed creating Verrazzano Monitoring namespace: %v", err)
		}
		if _, err := secret.CheckImagePullSecret(ctx.Client(), globalconst.VerrazzanoMonitoringNamespace); err != nil {
			return ctx.Log().ErrorfNewErr("Failed checking for image pull secret: %v", err)
		}
	}
	if vzconfig.IsKeycloakEnabled(ctx.EffectiveCR()) {
		istio := ctx.EffectiveCR().Spec.Components.Istio
		if err := namespace.CreateKeycloakNamespace(ctx.Client(), istio != nil && istio.IsInjectionEnabled()); err != nil {
			return ctx.Log().ErrorfNewErr("Failed creating Keycloak namespace: %v", err)
		}
	}
	// cattle-system NS must be created since the rancher NetworkPolicy, which is always installed, requires it
	if err := namespace.CreateRancherNamespace(ctx.Client()); err != nil {
		return ctx.Log().ErrorfNewErr("Failed creating Rancher namespace: %v", err)
	}
	return nil
}

// LabelKubeSystemNamespace adds the label needed by network polices to kube-system
func LabelKubeSystemNamespace(client clipkg.Client) error {
	const KubeSystemNamespace = "kube-system"
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: KubeSystemNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), client, &ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["verrazzano.io/namespace"] = KubeSystemNamespace
		return nil
	}); err != nil {
		return err
	}
	return nil
}

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

//cleanTempFiles - Clean up the override temp files in the temp dir
func cleanTempFiles(ctx spi.ComponentContext) {
	if err := vzos.RemoveTempFiles(ctx.Log().GetZapLogger(), tmpFileCleanPattern); err != nil {
		ctx.Log().Errorf("Failed deleting temp files: %v", err)
	}
}

//importToHelmChart annotates any existing objects that should be managed by helm
func importToHelmChart(cli clipkg.Client) error {
	namespacedName := types.NamespacedName{Name: nodeExporter, Namespace: globalconst.VerrazzanoMonitoringNamespace}
	name := types.NamespacedName{Name: nodeExporter}
	objects := []clipkg.Object{
		&appsv1.DaemonSet{},
		&corev1.ServiceAccount{},
		&corev1.Service{},
	}

	noNamespaceObjects := []clipkg.Object{
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
	}

	for _, obj := range objects {
		if _, err := associateHelmObjectToThisRelease(cli, obj, namespacedName); err != nil {
			return err
		}
	}

	for _, obj := range noNamespaceObjects {
		if _, err := associateHelmObjectToThisRelease(cli, obj, name); err != nil {
			return err
		}
	}
	return nil
}

//exportFromHelmChart annotates any existing objects that should be managed by another helm component, e.g.
// the resources associated with the authproxy which historically were associated with the Verrazzano chart.
func exportFromHelmChart(cli clipkg.Client) error {
	// The authproxy resources can not be managed by the authproxy component since the upgrade path may be from a
	// version that does not define the authproxy as a top level component and therefore PreUpgrade is not invoked
	// on the authproxy component (in that case the authproxy upgrade is skipped)
	authproxyReleaseName := types.NamespacedName{Name: authproxy.ComponentName, Namespace: authproxy.ComponentNamespace}
	namespacedName := authproxyReleaseName
	name := types.NamespacedName{Name: authproxy.ComponentName}
	objects := []clipkg.Object{
		&corev1.ServiceAccount{},
		&corev1.Service{},
		&appsv1.Deployment{},
	}

	noNamespaceObjects := []clipkg.Object{
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
	}

	// namespaced resources
	for _, obj := range objects {
		if _, err := common.AssociateHelmObject(cli, obj, authproxyReleaseName, namespacedName, true); err != nil {
			return err
		}
	}

	authproxyManagedResources := authproxy.GetHelmManagedResources()
	for _, managedResource := range authproxyManagedResources {
		if _, err := common.AssociateHelmObject(cli, managedResource.Obj, authproxyReleaseName, managedResource.NamespacedName, true); err != nil {
			return err
		}
	}

	// cluster resources
	for _, obj := range noNamespaceObjects {
		if _, err := common.AssociateHelmObject(cli, obj, authproxyReleaseName, name, true); err != nil {
			return err
		}
	}
	return nil
}

//associateHelmObjectToThisRelease annotates an object as being managed by the verrazzano helm chart
func associateHelmObjectToThisRelease(cli clipkg.Client, obj clipkg.Object, namespacedName types.NamespacedName) (clipkg.Object, error) {
	return common.AssociateHelmObject(cli, obj, types.NamespacedName{Name: ComponentName, Namespace: globalconst.VerrazzanoSystemNamespace}, namespacedName, false)
}

// GetProfile Returns the configured profile name, or "prod" if not specified in the configuration
func getProfile(vz *vzapi.Verrazzano) vzapi.ProfileType {
	profile := vz.Spec.Profile
	if len(profile) == 0 {
		profile = vzapi.Prod
	}
	return profile
}

// HashSum returns the hash sum of the config object
func HashSum(config interface{}) string {
	sha := sha256.New()
	if data, err := yaml.Marshal(config); err == nil {
		sha.Write(data)
		return fmt.Sprintf("%x", sha.Sum(nil))
	}
	return ""
}

// GetOverrides returns install overrides for a component
func GetOverrides(effectiveCR *vzapi.Verrazzano) []vzapi.Overrides {
	if effectiveCR.Spec.Components.Verrazzano != nil {
		return effectiveCR.Spec.Components.Verrazzano.ValueOverrides
	}
	return []vzapi.Overrides{}
}
