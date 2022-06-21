// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/console"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentd"
	"io/ioutil"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
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

	nodeExporterDaemonset = "node-exporter"

	prometheusDeployment = "vmi-system-prometheus-0"
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
	return nil
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
// the resources associated with authproxy, fluentd and console which historically were associated with the Verrazzano chart.
func exportFromHelmChart(cli clipkg.Client) error {
	err := associateAuthProxyResources(cli)
	if err != nil {
		return err
	}

	err = associateFluentdResources(cli)
	if err != nil {
		return err
	}

	err = associateConsoleResources(cli)
	if err != nil {
		return err
	}

	return nil
}

func associateAuthProxyResources(cli clipkg.Client) error {
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

func associateFluentdResources(cli clipkg.Client) error {
	fluentdReleaseName := types.NamespacedName{Name: fluentd.ComponentName, Namespace: fluentd.ComponentNamespace}
	namespacedName := fluentdReleaseName
	name := types.NamespacedName{Name: fluentd.ComponentName}
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
		if _, err := common.AssociateHelmObject(cli, obj, fluentdReleaseName, namespacedName, true); err != nil {
			return err
		}
	}

	helmManagedResources := fluentd.GetHelmManagedResources()
	for _, managedResource := range helmManagedResources {
		if _, err := common.AssociateHelmObject(cli, managedResource.Obj, fluentdReleaseName, managedResource.NamespacedName, true); err != nil {
			return err
		}
	}

	// cluster resources
	for _, obj := range noNamespaceObjects {
		if _, err := common.AssociateHelmObject(cli, obj, fluentdReleaseName, name, true); err != nil {
			return err
		}
	}

	return nil
}

func associateConsoleResources(cli clipkg.Client) error {
	// Associate console objects
	consoleReleaseName := types.NamespacedName{Name: console.ComponentName, Namespace: console.ComponentNamespace}
	consoleObjects := []clipkg.Object{
		&corev1.ServiceAccount{},
		&corev1.Service{},
		&appsv1.Deployment{},
	}
	for _, obj := range consoleObjects {
		if _, err := common.AssociateHelmObject(cli, obj, consoleReleaseName, consoleReleaseName, true); err != nil {
			return err
		}
	}
	return nil
}

//associateHelmObjectToThisRelease annotates an object as being managed by the verrazzano helm chart
func associateHelmObjectToThisRelease(cli clipkg.Client, obj clipkg.Object, namespacedName types.NamespacedName) (clipkg.Object, error) {
	return common.AssociateHelmObject(cli, obj, types.NamespacedName{Name: ComponentName, Namespace: globalconst.VerrazzanoSystemNamespace}, namespacedName, false)
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
