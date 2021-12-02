// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/namespace"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// componentName is the name of the component

const (
	componentName           = "verrazzano"
	keycloakInClusterURL    = "keycloak-http.keycloak.svc.cluster.local"
	esHelmValuePrefixFormat = "elasticSearch.%s"

	workloadName  = "system-es-master"
	containerName = "es-master"
	portName      = "http"
	indexPattern  = "verrazzano-*"
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

// VerrazzanoPreUpgrade contains code that is run prior to helm upgrade for the Verrazzano helm chart
func verrazzanoPreUpgrade(log *zap.SugaredLogger, client clipkg.Client, _ string, namespace string, _ string) error {
	return fixupFluentdDaemonset(log, client, namespace)
}

// appendVerrazzanoOverrides appends the image overrides for the monitoring-init-images subcomponent
func appendVerrazzanoOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {

	// Append some custom image overrides
	// - use local KeyValues array to ensure we append those after the file override; typically won't matter with the
	//   way we implement Helm calls, but don't depend on that
	vzkvs, err := appendCustomImageOverrides(kvs)
	if err != nil {
		return kvs, err
	}

	effectiveCR := ctx.EffectiveCR()
	// Find any storage overrides for the VMI, and
	resourceRequestOverrides, err := findStorageOverride(effectiveCR)
	if err != nil {
		return kvs, err
	}

	// Overrides object to store any user overrides
	overrides := verrazzanoValues{}

	// Append the simple overrides
	if err := appendVerrazzanoValues(ctx, &overrides); err != nil {
		return kvs, ctrlerrors.RetryableError{Source: componentName, Cause: err}
	}
	// Append any VMI overrides to the override values object, and any installArgs overrides to the kvs list
	vzkvs = appendVMIOverrides(effectiveCR, &overrides, resourceRequestOverrides, vzkvs)

	// append any fluentd overrides
	appendFluentdOverrides(effectiveCR, &overrides)
	// append the security role overrides
	if err := appendSecurityOverrides(effectiveCR, &overrides); err != nil {
		return kvs, ctrlerrors.RetryableError{Source: componentName, Cause: err}
	}

	// Write the overrides file to a temp dir and add a helm file override argument
	overridesFileName, err := generateOverridesFile(ctx, &overrides)
	if err != nil {
		return kvs, ctrlerrors.RetryableError{Source: componentName, Cause: err}
	}

	// Append any installArgs overrides in vzkvs after the file overrides to ensure precedence of those
	kvs = append(kvs, bom.KeyValue{Value: overridesFileName, IsFile: true})
	kvs = append(kvs, vzkvs...)
	return kvs, nil
}

func appendCustomImageOverrides(kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, ctrlerrors.RetryableError{Source: componentName, Cause: err}
	}

	imageOverrides, err := bomFile.BuildImageOverrides("monitoring-init-images")
	if err != nil {
		return kvs, ctrlerrors.RetryableError{Source: componentName, Cause: err}
	}

	kvs = append(kvs, imageOverrides...)
	return kvs, nil
}

const tmpFilePrefix = "verrazzano-overrides-"

func generateOverridesFile(ctx spi.ComponentContext, overrides *verrazzanoValues) (string, error) {
	bytes, err := yaml.Marshal(overrides)
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("%s*.yaml", tmpFilePrefix))
	if err != nil {
		return "", err
	}

	overridesFileName := file.Name()
	if err := writeFileFunc(overridesFileName, bytes, fs.ModeAppend); err != nil {
		return "", err
	}
	ctx.Log().Infof("Verrazzano install overrides file %s contents: %s", overridesFileName, string(bytes))
	return overridesFileName, nil
}

func appendVerrazzanoValues(ctx spi.ComponentContext, overrides *verrazzanoValues) error {
	effectiveCR := ctx.EffectiveCR()
	if isWildcardDNS, domain := getWildcardDNS(&effectiveCR.Spec); isWildcardDNS {
		overrides.DNS = &dnsValues{
			Wildcard: &wildcardDNSSettings{
				Domain: domain,
			},
		}
	}

	dnsSuffix, err := vzconfig.GetDNSSuffix(ctx.Client(), effectiveCR)
	if err != nil {
		return ctrlerrors.RetryableError{
			Source: componentName,
			Cause:  err,
		}
	}

	if externalDNSEnabled := vzconfig.IsExternalDNSEnabled(effectiveCR); externalDNSEnabled {
		overrides.Externaldns = &externalDNSValues{
			Enabled: externalDNSEnabled,
		}
	}

	envName := vzconfig.GetEnvName(effectiveCR)
	overrides.Config = &configValues{
		EnvName:   envName,
		DNSSuffix: dnsSuffix,
	}

	overrides.Keycloak = &keycloakValues{Enabled: vzconfig.IsKeycloakEnabled(effectiveCR)}
	overrides.Rancher = &rancherValues{Enabled: vzconfig.IsRancherEnabled(effectiveCR)}
	overrides.Console = &consoleValues{Enabled: vzconfig.IsConsoleEnabled(effectiveCR)}
	overrides.VerrazzanoOperator = &voValues{Enabled: isVMOEnabled(effectiveCR)}
	overrides.MonitoringOperator = &vmoValues{Enabled: isVMOEnabled(effectiveCR)}
	overrides.API = &apiValues{
		Proxy: &proxySettings{
			OidcProviderHost:          fmt.Sprintf("keycloak.%s.%s", envName, dnsSuffix),
			OidcProviderHostInCluster: keycloakInClusterURL,
		},
	}
	return nil
}

func appendSecurityOverrides(effectiveCR *vzapi.Verrazzano, overrides *verrazzanoValues) error {
	vzSpec := effectiveCR.Spec

	numAdminSubjects := len(vzSpec.Security.AdminSubjects)
	numMonSubjects := len(vzSpec.Security.MonitorSubjects)
	if numMonSubjects == 0 && numAdminSubjects == 0 {
		return nil
	}

	overrides.Security = &securityRoleBindingValues{}

	if numAdminSubjects > 0 {
		adminSubjectsMap := make(map[string]subject, numAdminSubjects)
		for i, adminSubj := range vzSpec.Security.AdminSubjects {
			subjectName := fmt.Sprintf("subject-%d", i)
			if err := vzconfig.ValidateRoleBindingSubject(adminSubj, subjectName); err != nil {
				return err
			}
			adminSubjectsMap[subjectName] = subject{
				Name:      adminSubj.Name,
				Kind:      adminSubj.Kind,
				Namespace: adminSubj.Namespace,
				APIGroup:  adminSubj.APIGroup,
			}
		}
		overrides.Security.AdminSubjects = adminSubjectsMap
	}
	if numMonSubjects > 0 {
		monSubjectMap := make(map[string]subject, numMonSubjects)
		for i, monSubj := range vzSpec.Security.MonitorSubjects {
			subjectName := fmt.Sprintf("subject-%d", i)
			if err := vzconfig.ValidateRoleBindingSubject(monSubj, fmt.Sprintf("monitorSubjects[%d]", i)); err != nil {
				return err
			}
			monSubjectMap[subjectName] = subject{
				Name:      monSubj.Name,
				Kind:      monSubj.Kind,
				Namespace: monSubj.Namespace,
				APIGroup:  monSubj.APIGroup,
			}
		}
		overrides.Security.MonitorSubjects = monSubjectMap
	}
	return nil
}

func appendVMIOverrides(effectiveCR *vzapi.Verrazzano, overrides *verrazzanoValues, storageOverrides *resourceRequestValues, kvs []bom.KeyValue) []bom.KeyValue {
	overrides.Kibana = &kibanaValues{Enabled: vzconfig.IsKibanaEnabled(effectiveCR)}

	overrides.ElasticSearch = &elasticsearchValues{
		Enabled: vzconfig.IsElasticsearchEnabled(effectiveCR),
	}
	if storageOverrides != nil {
		overrides.ElasticSearch.Nodes = &esNodes{
			// Only have to override the data node storage
			Data: &esNodeValues{
				Requests: storageOverrides,
			},
		}
	}
	if effectiveCR.Spec.Components.Elasticsearch != nil {
		for _, arg := range effectiveCR.Spec.Components.Elasticsearch.ESInstallArgs {
			kvs = append(kvs, bom.KeyValue{
				Key:   fmt.Sprintf(esHelmValuePrefixFormat, arg.Name),
				Value: arg.Value,
			})
		}
	}

	overrides.Prometheus = &prometheusValues{
		Enabled:  vzconfig.IsPrometheusEnabled(effectiveCR),
		Requests: storageOverrides,
	}

	overrides.Grafana = &grafanaValues{
		Enabled:  vzconfig.IsGrafanaEnabled(effectiveCR),
		Requests: storageOverrides,
	}
	return kvs
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
			return nil, fmt.Errorf("Did not find matching storage volume template for claim %s", pvcClaim.ClaimName)
		}
		storageString := storageSpec.Resources.Requests.Storage().String()
		return &resourceRequestValues{
			Storage: storageString,
		}, nil
	}
	return nil, fmt.Errorf("Unsupported volume source: %v", defaultVolumeSource)
}

func appendFluentdOverrides(effectiveCR *vzapi.Verrazzano, overrides *verrazzanoValues) {
	overrides.Fluentd = &fluentdValues{
		Enabled: vzconfig.IsFluentdEnabled(effectiveCR),
	}

	fluentd := effectiveCR.Spec.Components.Fluentd
	if fluentd != nil {
		overrides.Logging = &loggingValues{}
		if len(fluentd.ElasticsearchURL) > 0 {
			overrides.Logging.ElasticsearchURL = fluentd.ElasticsearchURL
		}
		if len(fluentd.ElasticsearchSecret) > 0 {
			overrides.Logging.ElasticsearchSecret = fluentd.ElasticsearchSecret
		}
		if len(fluentd.ExtraVolumeMounts) > 0 {
			for _, vm := range fluentd.ExtraVolumeMounts {
				dest := vm.Source
				if vm.Destination != "" {
					dest = vm.Destination
				}
				readOnly := true
				if vm.ReadOnly != nil {
					readOnly = *vm.ReadOnly
				}
				overrides.Fluentd.ExtraVolumeMounts = append(overrides.Fluentd.ExtraVolumeMounts,
					volumeMount{Source: vm.Source, Destination: dest, ReadOnly: readOnly})
			}
		}
	}
}

func isVMOEnabled(vz *vzapi.Verrazzano) bool {
	return vzconfig.IsPrometheusEnabled(vz) || vzconfig.IsKibanaEnabled(vz) || vzconfig.IsElasticsearchEnabled(vz) || vzconfig.IsGrafanaEnabled(vz)
}

func getWildcardDNS(vz *vzapi.VerrazzanoSpec) (bool, string) {
	if vz.Components.DNS != nil && vz.Components.DNS.Wildcard != nil {
		return true, vz.Components.DNS.Wildcard.Domain
	}
	return false, ""
}

// This function is used to fixup the fluentd daemonset on a managed cluster so that helm upgrade of Verrazzano does
// not fail.  Prior to Verrazzano v1.0.1, the mcagent would change the environment variables CLUSTER_NAME and
// ELASTICSEARCH_URL on a managed cluster to use valueFrom (from a secret) instead of using a Value. The helm chart
// template for the fluentd daemonset expects a Value.
func fixupFluentdDaemonset(log *zap.SugaredLogger, client clipkg.Client, namespace string) error {
	// Get the fluentd daemonset resource
	fluentdNamespacedName := types.NamespacedName{Name: "fluentd", Namespace: namespace}
	daemonSet := appsv1.DaemonSet{}
	err := client.Get(context.TODO(), fluentdNamespacedName, &daemonSet)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		log.Errorf("Failed to find the fluentd DaemonSet %s, %v", daemonSet.Name, err)
		return err
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
		return fmt.Errorf("fluentd container not found in fluentd daemonset: %s", daemonSet.Name)
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
		return fmt.Errorf("the secret named %s in namespace %s is missing the required field %s", secret.Name, secret.Namespace, constants.ClusterNameData)
	}

	// The secret must contain the Elasticsearch endpoint's URL
	elasticsearchURL, ok := secret.Data[constants.ElasticsearchURLData]
	if !ok {
		return fmt.Errorf("the secret named %s in namespace %s is missing the required field %s", secret.Name, secret.Namespace, constants.ElasticsearchURLData)
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
	log.Infof("Updating fluentd daemonset to use valueFrom instead of Value for CLUSTER_NAME and ELASTICSEARCH_URL environment variables")
	err = client.Update(context.TODO(), &daemonSet)
	return err
}

func createAndLabelNamespaces(ctx spi.ComponentContext) error {
	if err := namespace.CreateVerrazzanoSystemNamespace(ctx.Client()); err != nil {
		return err
	}
	if _, err := secret.CheckImagePullSecret(ctx.Client(), globalconst.VerrazzanoSystemNamespace); err != nil {
		return ctrlerrors.RetryableError{Source: componentName, Cause: err}
	}
	if err := namespace.CreateVerrazzanoMultiClusterNamespace(ctx.Client()); err != nil {
		return err
	}
	if isVMOEnabled(ctx.EffectiveCR()) {
		// If the monitoring operator is enabled, create the monitoring namespace and copy the image pull secret
		if err := namespace.CreateVerrazzanoMonitoringNamespace(ctx.Client()); err != nil {
			return err
		}
		if _, err := secret.CheckImagePullSecret(ctx.Client(), globalconst.VerrazzanoMonitoringNamespace); err != nil {
			return ctrlerrors.RetryableError{Source: componentName, Cause: err}
		}
	}
	if vzconfig.IsKeycloakEnabled(ctx.EffectiveCR()) {
		if err := namespace.CreateKeycloakNamespace(ctx.Client()); err != nil {
			return ctrlerrors.RetryableError{Source: componentName, Cause: err}
		}
	}
	if vzconfig.IsRancherEnabled(ctx.EffectiveCR()) {
		// Create and/or label the Rancher system namespaces if necessary
		if err := namespace.CreateRancherNamespace(ctx.Client()); err != nil {
			return ctrlerrors.RetryableError{Source: componentName, Cause: err}
		}
		if err := namespace.CreateAndLabelNamespace(ctx.Client(), globalconst.RancherOperatorSystemNamespace,
			true, false); err != nil {
			return ctrlerrors.RetryableError{Source: componentName, Cause: err}
		}
	}
	return nil
}

func loggingPreInstall(ctx spi.ComponentContext) error {
	if vzconfig.IsFluentdEnabled(ctx.EffectiveCR()) {
		// If fluentd is enabled, copy any custom Elasticsearch secret if found
		fluentdConfig := ctx.EffectiveCR().Spec.Components.Fluentd
		if fluentdConfig != nil &&
			len(fluentdConfig.ElasticsearchURL) > 0 &&
			fluentdConfig.ElasticsearchSecret != globalconst.DefaultElasticsearchSecretName {

			esSecret := fluentdConfig.ElasticsearchSecret
			vzLog := ctx.Log()
			vzLog.Debugf("Copying custom/external Elasticsearch secret %s to %s namespace",
				esSecret, globalconst.VerrazzanoSystemNamespace)
			targetSecret := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: esSecret, Namespace: globalconst.VerrazzanoSystemNamespace},
			}
			opResult, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &targetSecret, func() error {
				sourceSecret := corev1.Secret{}
				if err := ctx.Client().Get(context.TODO(),
					types.NamespacedName{Name: esSecret, Namespace: constants.VerrazzanoInstallNamespace},
					&sourceSecret); err != nil {
					return err
				}
				targetSecret.Type = sourceSecret.Type
				targetSecret.Immutable = sourceSecret.Immutable
				targetSecret.StringData = sourceSecret.StringData
				targetSecret.Data = sourceSecret.Data
				return nil
			})
			vzLog.Debugf("Copy custom Elasticsearch secret result: %s", opResult)
			if err != nil {
				if !errors.IsNotFound(err) {
					return ctrlerrors.RetryableError{
						Source: componentName,
						Cause:  err,
					}
				}
				vzLog.Errorf("Custom Elasticsearch secret %s not found in namespace %s",
					esSecret, constants.VerrazzanoInstallNamespace)
				return ctrlerrors.RetryableError{Source: componentName}
			}
		}
	}
	return nil
}

// isVerrazzanoSecretReady returns true if the Verrazzano secret is present in the system namespace
func isVerrazzanoSecretReady(ctx spi.ComponentContext) bool {
	if err := ctx.Client().Get(context.TODO(),
		types.NamespacedName{Name: "verrazzano", Namespace: globalconst.VerrazzanoSystemNamespace},
		&corev1.Secret{}); err != nil {
		if !errors.IsNotFound(err) {
			ctx.Log().Error("Unexpected error getting verrazzano secret: %s", err)
			return false
		}
		ctx.Log().Debugf("Verrazzano secret not found")
		return false
	}
	return true
}

//cleanTempFiles - Clean up the override temp files in the temp dir
func cleanTempFiles(ctx spi.ComponentContext) {
	log := ctx.Log()
	files, err := ioutil.ReadDir(os.TempDir())
	if err != nil {
		log.Errorf("Unable to read temp directory: %s", err.Error())
	}
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), tmpFilePrefix) && strings.HasSuffix(file.Name(), ".yaml") {
			fullPath := filepath.Join(os.TempDir(), file.Name())
			log.Debugf("Deleting temp file %s", fullPath)
			if err := os.Remove(fullPath); err != nil {
				log.Errorf("Error deleting temp file %s", fullPath)
			}
		}
	}
}

// fixupElasticSearchReplicaCount fixes the replica count set for single node Elasticsearch cluster
func fixupElasticSearchReplicaCount(ctx spi.ComponentContext, namespace string) error {
	// Only apply this fix to clusters with Elasticsearch enabled.
	if !vzconfig.IsElasticsearchEnabled(ctx.EffectiveCR()) {
		ctx.Log().Info("Elasticsearch Post Upgrade: Replica count update unnecessary on managed cluster.")
		return nil
	}

	// Only apply this fix to clusters being upgraded from a source version before 1.1.0.
	ver1_1_0, err := semver.NewSemVersion("v1.1.0")
	if err != nil {
		return err
	}
	sourceVer, err := semver.NewSemVersion(ctx.ActualCR().Status.Version)
	if err != nil {
		ctx.Log().Errorf("Elasticsearch Post Upgrade: Invalid source Verrazzano version: %s", err)
		return err
	}
	if sourceVer.IsGreatherThan(ver1_1_0) || sourceVer.IsEqualTo(ver1_1_0) {
		ctx.Log().Info("Elasticsearch Post Upgrade: Replica count update unnecessary for source Verrazzano version %v.", sourceVer.ToString())
		return nil
	}

	// Wait for an Elasticsearch (i.e., label app=system-es-master) pod with container (i.e. es-master) to be ready.
	pods, err := waitForPodsWithReadyContainer(ctx.Client(), 15*time.Second, 5*time.Minute, containerName, clipkg.MatchingLabels{"app": workloadName}, clipkg.InNamespace(namespace))
	if err != nil {
		ctx.Log().Errorf("Elasticsearch Post Upgrade: Error getting the Elasticsearch pods: %s", err)
		return err
	}
	if len(pods) == 0 {
		err := fmt.Errorf("no pods found")
		ctx.Log().Errorf("Elasticsearch Post Upgrade: Failed to find Elasticsearch pods: %s", err)
		return err
	}
	pod := pods[0]

	// Find the Elasticsearch HTTP control container port.
	httpPort, err := getNamedContainerPortOfContainer(pod, containerName, portName)
	if err != nil {
		ctx.Log().Errorf("Elasticsearch Post Upgrade: Failed to find HTTP port of Elasticsearch container: %s", err)
		return err
	}
	if httpPort <= 0 {
		err := fmt.Errorf("no port found")
		ctx.Log().Errorf("Elasticsearch Post Upgrade: Failed to find Elasticsearch port: %s", err)
		return err
	}

	// Set the the number of replicas for the Verrazzano indices
	// to something valid in single node Elasticsearch cluster
	ctx.Log().Info("Elasticsearch Post Upgrade: Getting the health of the Elasticsearch cluster")
	getCmd := execCommand("kubectl", "exec", pod.Name, "-n", namespace, "-c", containerName, "--", "sh", "-c",
		fmt.Sprintf("curl -v -XGET -s -k --fail http://localhost:%d/_cluster/health", httpPort))
	output, err := getCmd.Output()
	if err != nil {
		ctx.Log().Errorf("Elasticsearch Post Upgrade: Error getting the Elasticsearch cluster health: %s", err)
		return err
	}
	ctx.Log().Info("Elasticsearch Post Upgrade: Output of the health of the Elasticsearch cluster %v", string(output))
	// If the data node count is seen as 1 then the node is considered as single node cluster
	if strings.Contains(string(output), `"number_of_data_nodes":1,`) {
		// Login to Elasticsearch and update index settings for single data node elasticsearch cluster
		putCmd := execCommand("kubectl", "exec", pod.Name, "-n", namespace, "-c", containerName, "--", "sh", "-c",
			fmt.Sprintf(`curl -v -XPUT -d '{"index":{"auto_expand_replicas":"0-1"}}' --header 'Content-Type: application/json' -s -k --fail http://localhost:%d/%s/_settings`, httpPort, indexPattern))
		_, err = putCmd.Output()
		if err != nil {
			ctx.Log().Errorf("Elasticsearch Post Upgrade: Error logging into Elasticsearch: %s", err)
			return err
		}
		ctx.Log().Info("Elasticsearch Post Upgrade: Successfully updated Elasticsearch index settings")
	}
	ctx.Log().Info("Elasticsearch Post Upgrade: Completed successfully")
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
	return -1, fmt.Errorf("no port named %s found in container %s of pod %s", portName, containerName, pod.Name)
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
