// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"io/ioutil"
	rbacv1 "k8s.io/api/rbac/v1"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/namespace"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
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
func resolveVerrazzanoNamespace(ns string) string {
	if len(ns) > 0 && ns != "default" {
		return ns
	}
	return globalconst.VerrazzanoSystemNamespace
}

// isVerrazzanoReady Verrazzano component ready-check
func isVerrazzanoReady(ctx spi.ComponentContext) bool {
	var deployments []types.NamespacedName
	if isVMOEnabled(ctx.EffectiveCR()) {
		deployments = append(deployments, []types.NamespacedName{
			{Name: "verrazzano-monitoring-operator", Namespace: globalconst.VerrazzanoSystemNamespace},
		}...)
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	if !status.DeploymentsReady(ctx.Log(), ctx.Client(), deployments, 1, prefix) {
		return false
	}
	return isVerrazzanoSecretReady(ctx)
}

// VerrazzanoPreUpgrade contains code that is run prior to helm upgrade for the Verrazzano helm chart
func verrazzanoPreUpgrade(log vzlog.VerrazzanoLogger, client clipkg.Client, _ string, namespace string, _ string) error {
	if err := importToHelmChart(client); err != nil {
		return err
	}
	if err := ensureVMISecret(client); err != nil {

	}
	return fixupFluentdDaemonset(log, client, namespace)
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

func isVMOEnabled(vz *vzapi.Verrazzano) bool {
	return vzconfig.IsPrometheusEnabled(vz) || vzconfig.IsKibanaEnabled(vz) || vzconfig.IsElasticsearchEnabled(vz) || vzconfig.IsGrafanaEnabled(vz)
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
	secret := corev1.Secret{}
	err = client.Get(context.TODO(), secretNamespacedName, &secret)
	if err != nil {
		return err
	}

	// The secret must contain a cluster name
	clusterName, ok := secret.Data[constants.ClusterNameData]
	if !ok {
		return log.ErrorfNewErr("Failed, the secret named %s in namespace %s is missing the required field %s", secret.Name, secret.Namespace, constants.ClusterNameData)
	}

	// The secret must contain the Elasticsearch endpoint's URL
	elasticsearchURL, ok := secret.Data[constants.ElasticsearchURLData]
	if !ok {
		return log.ErrorfNewErr("Failed, the secret named %s in namespace %s is missing the required field %s", secret.Name, secret.Namespace, constants.ElasticsearchURLData)
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
	if err := namespace.CreateVerrazzanoSystemNamespace(ctx.Client()); err != nil {
		return err
	}
	if _, err := secret.CheckImagePullSecret(ctx.Client(), globalconst.VerrazzanoSystemNamespace); err != nil {
		return ctx.Log().ErrorfNewErr("Failed checking for image pull secret: %v", err)
	}
	if err := namespace.CreateVerrazzanoMultiClusterNamespace(ctx.Client()); err != nil {
		return err
	}
	if isVMOEnabled(ctx.EffectiveCR()) {
		// If the monitoring operator is enabled, create the monitoring namespace and copy the image pull secret
		if err := namespace.CreateVerrazzanoMonitoringNamespace(ctx.Client()); err != nil {
			return ctx.Log().ErrorfNewErr("Failed creating Verrazzano Monitoring namespace: %v", err)
		}
		if _, err := secret.CheckImagePullSecret(ctx.Client(), globalconst.VerrazzanoMonitoringNamespace); err != nil {
			return ctx.Log().ErrorfNewErr("Failed checking for image pull secret: %v", err)
		}
	}
	if vzconfig.IsKeycloakEnabled(ctx.EffectiveCR()) {
		if err := namespace.CreateKeycloakNamespace(ctx.Client()); err != nil {
			return ctx.Log().ErrorfNewErr("Failed creating Keycloak namespace: %v", err)
		}
	}
	if vzconfig.IsRancherEnabled(ctx.EffectiveCR()) {
		if err := namespace.CreateAndLabelNamespace(ctx.Client(), globalconst.RancherOperatorSystemNamespace,
			true, false); err != nil {
			return ctx.Log().ErrorfNewErr("Failed creating Rancher operator system namespace %s: %v", globalconst.RancherOperatorSystemNamespace, err)
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
			// Copy the external Elasticsearch secret
			if len(fluentdConfig.ElasticsearchURL) > 0 && fluentdConfig.ElasticsearchSecret != globalconst.DefaultElasticsearchSecretName {
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
	ver1_1_0, err := semver.NewSemVersion("v1.1.0")
	if err != nil {
		return err
	}
	sourceVer, err := semver.NewSemVersion(ctx.ActualCR().Status.Version)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed Elasticsearch post-upgrade: Invalid source Verrazzano version: %v", err)
	}
	if sourceVer.IsGreatherThan(ver1_1_0) || sourceVer.IsEqualTo(ver1_1_0) {
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
	objects := []controllerutil.Object{
		&appsv1.DaemonSet{},
		&corev1.ServiceAccount{},
		&corev1.Service{},
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
	}

	for _, obj := range objects {
		if _, err := importHelmObject(cli, obj, namespacedName); err != nil {
			return err
		}
	}
	return nil
}

//importHelmObject annotates an object as being managed by the verrazzano helm chart
func importHelmObject(cli clipkg.Client, obj controllerutil.Object, namespacedName types.NamespacedName) (controllerutil.Object, error) {
	if err := cli.Get(context.TODO(), namespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return obj, nil
		}
		return obj, err
	}
	objMerge := clipkg.MergeFrom(obj.DeepCopyObject())
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["meta.helm.sh/release-name"] = "verrazzano"
	annotations["meta.helm.sh/release-namespace"] = "verrazzano-system"
	obj.SetAnnotations(annotations)
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels["app.kubernetes.io/managed-by"] = "Helm"
	obj.SetLabels(labels)
	return obj, cli.Patch(context.TODO(), obj, objMerge)
}
