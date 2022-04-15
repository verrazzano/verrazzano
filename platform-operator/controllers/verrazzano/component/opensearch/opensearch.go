// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"time"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/namespace"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// ComponentName is the name of the component

const (
	esHelmValuePrefixFormat = "elasticSearch.%s"

	workloadName  = "system-es-master"
	containerName = "es-master"
	portName      = "http"
	indexPattern  = "verrazzano-*"

	tmpFilePrefix        = "verrazzano-overrides-"
	tmpSuffix            = "yaml"
	tmpFileCreatePattern = tmpFilePrefix + "*." + tmpSuffix
	tmpFileCleanPattern  = tmpFilePrefix + ".*\\." + tmpSuffix

	fluentDaemonset       = "fluentd"
	nodeExporterDaemonset = "node-exporter"

	esDataDeployment            = "vmi-system-es-data"
	esIngestDeployment          = "vmi-system-es-ingest"
	grafanaDeployment           = "vmi-system-grafana"
	kibanaDeployment            = "vmi-system-kibana"
	prometheusDeployment        = "vmi-system-prometheus-0"
	verrazzanoConsoleDeployment = "verrazzano-console"
	vmoDeployment               = "verrazzano-monitoring-operator"

	esMasterStatefulset = "vmi-system-es-master"
)

var (
	// For Unit test purposes
	execCommand   = exec.Command
	writeFileFunc = ioutil.WriteFile
)

func resetWriteFileFunc() {
	writeFileFunc = ioutil.WriteFile
}

// resolveVerrazzanoNamespace will return the default Verrazzano system namespace unless the namespace is specified
func resolveOpensearchNamespace(ns string) string {
	if len(ns) > 0 && ns != "default" {
		return ns
	}
	return globalconst.VerrazzanoSystemNamespace
}

// isOpensearchInstalled checks if Opensearch has been installed yet
func isOpensearchInstalled(ctx spi.ComponentContext) (bool, error) {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())

	var deployments []types.NamespacedName

	if vzconfig.IsKibanaEnabled(ctx.EffectiveCR()) {
		deployments = append(deployments,
			types.NamespacedName{
				Name:      kibanaDeployment,
				Namespace: ComponentNamespace,
			})
	}
	if vzconfig.IsElasticsearchEnabled(ctx.EffectiveCR()) {
		if ctx.EffectiveCR().Spec.Components.Elasticsearch != nil {
			esInstallArgs := ctx.EffectiveCR().Spec.Components.Elasticsearch.ESInstallArgs
			for _, args := range esInstallArgs {
				if args.Name == "nodes.data.replicas" {
					replicas, _ := strconv.Atoi(args.Value)
					for i := 0; replicas > 0 && i < replicas; i++ {
						deployments = append(deployments,
							types.NamespacedName{
								Name:      fmt.Sprintf("%s-%d", esDataDeployment, i),
								Namespace: ComponentNamespace,
							})
					}
					continue
				}
				if args.Name == "nodes.ingest.replicas" {
					replicas, _ := strconv.Atoi(args.Value)
					if replicas > 0 {
						deployments = append(deployments,
							types.NamespacedName{
								Name:      esIngestDeployment,
								Namespace: ComponentNamespace,
							})
					}
				}
			}
		}
	}

	deploymentsExist, err := status.DoDeploymentsExist(ctx.Log(), ctx.Client(), deployments, prefix)
	if !deploymentsExist {
		return false, err
	}

	// Next, check statefulsets
	if vzconfig.IsElasticsearchEnabled(ctx.EffectiveCR()) {
		if ctx.EffectiveCR().Spec.Components.Elasticsearch != nil {
			esInstallArgs := ctx.EffectiveCR().Spec.Components.Elasticsearch.ESInstallArgs
			for _, args := range esInstallArgs {
				if args.Name == "nodes.master.replicas" {
					var statefulsets []types.NamespacedName
					replicas, _ := strconv.Atoi(args.Value)
					if replicas > 0 {
						statefulsets = append(statefulsets,
							types.NamespacedName{
								Name:      esMasterStatefulset,
								Namespace: ComponentNamespace,
							})
						statefulsetsExist, err := status.DoStatefulSetsExist(ctx.Log(), ctx.Client(), statefulsets, prefix)
						if !statefulsetsExist {
							return false, err
						}
					}
					break
				}
			}
		}
	}

	return isVerrazzanoSecretReady(ctx), nil
}

// isOpensearchReady VMI components ready-check
func isOpensearchReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())

	var deployments []types.NamespacedName

	if vzconfig.IsKibanaEnabled(ctx.EffectiveCR()) {
		deployments = append(deployments,
			types.NamespacedName{
				Name:      kibanaDeployment,
				Namespace: ComponentNamespace,
			})
	}
	if vzconfig.IsElasticsearchEnabled(ctx.EffectiveCR()) {
		if ctx.EffectiveCR().Spec.Components.Elasticsearch != nil {
			esInstallArgs := ctx.EffectiveCR().Spec.Components.Elasticsearch.ESInstallArgs
			for _, args := range esInstallArgs {
				if args.Name == "nodes.data.replicas" {
					replicas, _ := strconv.Atoi(args.Value)
					for i := 0; replicas > 0 && i < replicas; i++ {
						deployments = append(deployments,
							types.NamespacedName{
								Name:      fmt.Sprintf("%s-%d", esDataDeployment, i),
								Namespace: ComponentNamespace,
							})
					}
					continue
				}
				if args.Name == "nodes.ingest.replicas" {
					replicas, _ := strconv.Atoi(args.Value)
					if replicas > 0 {
						deployments = append(deployments,
							types.NamespacedName{
								Name:      esIngestDeployment,
								Namespace: ComponentNamespace,
							})
					}
				}
			}
		}
	}

	if !status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix) {
		return false
	}

	// Next, check statefulsets
	if vzconfig.IsElasticsearchEnabled(ctx.EffectiveCR()) {
		if ctx.EffectiveCR().Spec.Components.Elasticsearch != nil {
			esInstallArgs := ctx.EffectiveCR().Spec.Components.Elasticsearch.ESInstallArgs
			for _, args := range esInstallArgs {
				if args.Name == "nodes.master.replicas" {
					var statefulsets []types.NamespacedName
					replicas, _ := strconv.Atoi(args.Value)
					if replicas > 0 {
						statefulsets = append(statefulsets,
							types.NamespacedName{
								Name:      esMasterStatefulset,
								Namespace: ComponentNamespace,
							})
						if !status.StatefulSetsAreReady(ctx.Log(), ctx.Client(), statefulsets, 1, prefix) {
							return false
						}
					}
					break
				}
			}
		}
	}

	return isVerrazzanoSecretReady(ctx)
}

// VerrazzanoPreUpgrade contains code that is run prior to helm upgrade for the Verrazzano helm chart
func opensearchPreUpgrade(ctx spi.ComponentContext) error {
	if err := common.ApplyCRDYaml(ctx, config.GetHelmVzChartsDir()); err != nil {
		return err
	}
	if err := importToHelmChart(ctx.Client()); err != nil {
		return err
	}
	if err := exportFromHelmChart(ctx.Client()); err != nil {
		return err
	}
	if err := ensureVMISecret(ctx.Client()); err != nil {
		return err
	}
	return ensureGrafanaAdminSecret(ctx.Client())
}

func findStorageOverride(effectiveCR *vzapi.Verrazzano) (*resourceRequestValues, error) {
	if effectiveCR == nil || effectiveCR.Spec.DefaultVolumeSource == nil {
		return nil, nil
	}
	defaultVolumeSource := effectiveCR.Spec.DefaultVolumeSource
	if defaultVolumeSource.EmptyDir != nil {
		return &resourceRequestValues{
			Storage: "",
		}, nil
	}
	if defaultVolumeSource.PersistentVolumeClaim != nil {
		pvcClaim := defaultVolumeSource.PersistentVolumeClaim
		storageSpec, found := vzconfig.FindVolumeTemplate(pvcClaim.ClaimName, effectiveCR.Spec.VolumeClaimSpecTemplates)
		if !found {
			return nil, fmt.Errorf("Failed, did not find matching storage volume template for claim %s", pvcClaim.ClaimName)
		}
		storageString := storageSpec.Resources.Requests.Storage().String()
		return &resourceRequestValues{
			Storage: storageString,
		}, nil
	}
	return nil, fmt.Errorf("Failed, unsupported volume source: %v", defaultVolumeSource)
}

func createAndLabelNamespaces(ctx spi.ComponentContext) error {
	if err := namespace.CreateVerrazzanoSystemNamespace(ctx.Client()); err != nil {
		return err
	}
	if _, err := secret.CheckImagePullSecret(ctx.Client(), globalconst.VerrazzanoSystemNamespace); err != nil {
		return ctx.Log().ErrorfNewErr("Failed checking for image pull secret: %v", err)
	}
	return nil
}

// isVerrazzanoSecretReady returns true if the Verrazzano secret is present in the system namespace
func isVerrazzanoSecretReady(ctx spi.ComponentContext) bool {
	if err := ctx.Client().Get(context.TODO(),
		types.NamespacedName{Name: "verrazzano", Namespace: globalconst.VerrazzanoSystemNamespace},
		&corev1.Secret{}); err != nil {
		if !errors.IsNotFound(err) {
			ctx.Log().Errorf("Failed, unexpected error getting verrazzano secret: %v", err)
			return false
		}
		ctx.Log().Debugf("Verrazzano secret not found")
		return false
	}
	return true
}

//cleanTempFiles - Clean up the override temp files in the temp dir
func cleanTempFiles(ctx spi.ComponentContext) {
	if err := vzos.RemoveTempFiles(ctx.Log().GetZapLogger(), tmpFileCleanPattern); err != nil {
		ctx.Log().Errorf("Failed deleting temp files: %v", err)
	}
}

// fixupElasticSearchReplicaCount fixes the replica count set for single node Elasticsearch cluster
func fixupElasticSearchReplicaCount(ctx spi.ComponentContext, namespace string) error {
	// Only apply this fix to clusters with Elasticsearch enabled.
	if !vzconfig.IsElasticsearchEnabled(ctx.EffectiveCR()) {
		ctx.Log().Debug("Elasticsearch Post Upgrade: Replica count update unnecessary on managed cluster.")
		return nil
	}

	// Only apply this fix to clusters being upgraded from a source version before 1.1.0.
	ver110, err := semver.NewSemVersion("v1.1.0")
	if err != nil {
		return err
	}
	sourceVer, err := semver.NewSemVersion(ctx.ActualCR().Status.Version)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed Elasticsearch post-upgrade: Invalid source Verrazzano version: %v", err)
	}
	if sourceVer.IsGreatherThan(ver110) || sourceVer.IsEqualTo(ver110) {
		ctx.Log().Debug("Elasticsearch Post Upgrade: Replica count update unnecessary for source Verrazzano version %v.", sourceVer.ToString())
		return nil
	}

	// Wait for an Elasticsearch (i.e., label app=system-es-master) pod with container (i.e. es-master) to be ready.
	pods, err := waitForPodsWithReadyContainer(ctx.Client(), 15*time.Second, 5*time.Minute, containerName, clipkg.MatchingLabels{"app": workloadName}, clipkg.InNamespace(namespace))
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed getting the Elasticsearch pods during post-upgrade: %v", err)
	}
	if len(pods) == 0 {
		return ctx.Log().ErrorfNewErr("Failed to find Elasticsearch pods during post-upgrade: %v", err)
	}
	pod := pods[0]

	// Find the Elasticsearch HTTP control container port.
	httpPort, err := getNamedContainerPortOfContainer(pod, containerName, portName)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to find HTTP port of Elasticsearch container during post-upgrade: %v", err)
	}
	if httpPort <= 0 {
		return ctx.Log().ErrorfNewErr("Failed to find Elasticsearch port during post-upgrade: %v", err)
	}

	// Set the the number of replicas for the Verrazzano indices
	// to something valid in single node Elasticsearch cluster
	ctx.Log().Debug("Elasticsearch Post Upgrade: Getting the health of the Elasticsearch cluster")
	getCmd := execCommand("kubectl", "exec", pod.Name, "-n", namespace, "-c", containerName, "--", "sh", "-c",
		fmt.Sprintf("curl -v -XGET -s -k --fail http://localhost:%d/_cluster/health", httpPort))
	output, err := getCmd.Output()
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed in Elasticsearch post upgrade: error getting the Elasticsearch cluster health: %v", err)
	}
	ctx.Log().Debugf("Elasticsearch Post Upgrade: Output of the health of the Elasticsearch cluster %s", string(output))
	if ctx.EffectiveCR().Spec.DefaultVolumeSource != nil && ctx.EffectiveCR().Spec.DefaultVolumeSource.EmptyDir != nil {
		ctx.Log().Infof("Skipping Elasticsearch health check due to lack of configured persistence")
	} else {
		// If the data node count is seen as 1 then the node is considered as single node cluster
		if strings.Contains(string(output), `"number_of_data_nodes":1,`) {
			// Login to Elasticsearch and update index settings for single data node elasticsearch cluster
			putCmd := execCommand("kubectl", "exec", pod.Name, "-n", namespace, "-c", containerName, "--", "sh", "-c",
				fmt.Sprintf(`curl -v -XPUT -d '{"index":{"auto_expand_replicas":"0-1"}}' --header 'Content-Type: application/json' -s -k --fail http://localhost:%d/%s/_settings`, httpPort, indexPattern))
			_, err = putCmd.Output()
			if err != nil {
				return ctx.Log().ErrorfNewErr("Failed in Elasticsearch post-upgrade: Error logging into Elasticsearch: %v", err)
			}
			ctx.Log().Debug("Elasticsearch Post Upgrade: Successfully updated Elasticsearch index settings")
		}
	}
	ctx.Log().Debug("Elasticsearch Post Upgrade: Completed successfully")
	return nil
}

func getNamedContainerPortOfContainer(pod corev1.Pod, containerName string, portName string) (int32, error) {
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			for _, port := range container.Ports {
				if port.Name == portName {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return -1, fmt.Errorf("Failed, no port named %s found in container %s of pod %s", portName, containerName, pod.Name)
}

func getPodsWithReadyContainer(client clipkg.Client, containerName string, podSelectors ...clipkg.ListOption) ([]corev1.Pod, error) {
	pods := []corev1.Pod{}
	list := &corev1.PodList{}
	err := client.List(context.TODO(), list, podSelectors...)
	if err != nil {
		return pods, err
	}
	for _, pod := range list.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name == containerName && containerStatus.Ready {
				pods = append(pods, pod)
			}
		}
	}
	return pods, err
}

func waitForPodsWithReadyContainer(client clipkg.Client, retryDelay time.Duration, timeout time.Duration, containerName string, podSelectors ...clipkg.ListOption) ([]corev1.Pod, error) {
	start := time.Now()
	for {
		pods, err := getPodsWithReadyContainer(client, containerName, podSelectors...)
		if err == nil && len(pods) > 0 {
			return pods, err
		}
		if time.Since(start) >= timeout {
			return pods, err
		}
		time.Sleep(retryDelay)
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
		if _, err := associateHelmObject(cli, obj, authproxyReleaseName, namespacedName, true); err != nil {
			return err
		}
	}

	authproxyManagedResources := authproxy.GetHelmManagedResources()
	for _, managedResource := range authproxyManagedResources {
		if _, err := associateHelmObject(cli, managedResource.Obj, authproxyReleaseName, managedResource.NamespacedName, true); err != nil {
			return err
		}
	}

	// cluster resources
	for _, obj := range noNamespaceObjects {
		if _, err := associateHelmObject(cli, obj, authproxyReleaseName, name, true); err != nil {
			return err
		}
	}
	return nil
}

//associateHelmObjectToThisRelease annotates an object as being managed by the verrazzano helm chart
func associateHelmObjectToThisRelease(cli clipkg.Client, obj clipkg.Object, namespacedName types.NamespacedName) (clipkg.Object, error) {
	return associateHelmObject(cli, obj, types.NamespacedName{Name: ComponentName, Namespace: globalconst.VerrazzanoSystemNamespace}, namespacedName, false)
}

//associateHelmObject annotates an object as being managed by the specified release helm chart
func associateHelmObject(cli clipkg.Client, obj clipkg.Object, releaseName types.NamespacedName, namespacedName types.NamespacedName, keepResource bool) (clipkg.Object, error) {
	if err := cli.Get(context.TODO(), namespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return obj, nil
		}
		return obj, err
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["meta.helm.sh/release-name"] = releaseName.Name
	annotations["meta.helm.sh/release-namespace"] = releaseName.Namespace
	if keepResource {
		annotations["helm.sh/resource-policy"] = "keep"
	}
	obj.SetAnnotations(annotations)
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels["app.kubernetes.io/managed-by"] = "Helm"
	obj.SetLabels(labels)
	err := cli.Update(context.TODO(), obj)
	return obj, err
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
