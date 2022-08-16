// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/pkg/semver"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	workloadName  = "system-es-master"
	containerName = "es-master"
	portName      = "http"
	indexPattern  = "verrazzano-*"

	esDataDeployment   = "vmi-system-es-data"
	esIngestDeployment = "vmi-system-es-ingest"

	esMasterStatefulset = "vmi-system-es-master"
)

var (
	// For Unit test purposes
	execCommand = exec.Command
)

// doesOSExist is the IsInstalled check
func doesOSExist(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	sts := []types.NamespacedName{{
		Name:      esMasterStatefulset,
		Namespace: ComponentNamespace,
	}}
	return status.DoStatefulSetsExist(ctx.Log(), ctx.Client(), sts, 1, prefix)
}

// IsSingleDataNodeCluster returns true if there is exactly 1 or 0 data nodes
func IsSingleDataNodeCluster(ctx spi.ComponentContext) bool {
	return findESReplicas(ctx, "data") <= 1
}

// isOSReady checks if the OpenSearch resources are ready
func isOSReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())

	// check data nodes
	var deployments []types.NamespacedName
	dataReplicas := findESReplicas(ctx, "data")
	for i := int32(0); dataReplicas > 0 && i < dataReplicas; i++ {
		deployments = append(deployments,
			types.NamespacedName{
				Name:      fmt.Sprintf("%s-%d", esDataDeployment, i),
				Namespace: ComponentNamespace,
			})
	}
	if !status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix) {
		return false
	}

	// check ingest nodes
	ingestReplicas := findESReplicas(ctx, "ingest")
	if ingestReplicas > 0 &&
		!status.DeploymentsAreReady(ctx.Log(), ctx.Client(), []types.NamespacedName{{
			Name:      esIngestDeployment,
			Namespace: ComponentNamespace,
		}}, ingestReplicas, prefix) {
		return false
	}

	// check master nodes
	masterReplicas := findESReplicas(ctx, "master")
	if masterReplicas > 0 &&
		!status.StatefulSetsAreReady(ctx.Log(), ctx.Client(), []types.NamespacedName{{
			Name:      esMasterStatefulset,
			Namespace: ComponentNamespace,
		}}, masterReplicas, prefix) {
		return false
	}

	return common.IsVMISecretReady(ctx)
}

// findESReplicas searches the ES install args to find the correct resources to search for in isReady
func findESReplicas(ctx spi.ComponentContext, nodeType string) int32 {
	if vzconfig.IsElasticsearchEnabled(ctx.EffectiveCR()) && ctx.EffectiveCR().Spec.Components.Elasticsearch != nil {
		esInstallArgs := ctx.EffectiveCR().Spec.Components.Elasticsearch.ESInstallArgs
		for _, args := range esInstallArgs {
			if args.Name == fmt.Sprintf("nodes.%s.replicas", nodeType) {
				replicas, _ := strconv.Atoi(args.Value) //nolint:gosec //#gosec G109
				return int32(replicas)
			}
		}
	}
	return 0
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
