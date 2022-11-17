// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"context"
	"fmt"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/types"
)

const (
	nodeNamePrefix = "vmi-system-%s"
	componentName  = "opensearch"

	// OpenSearchTier is the environment variable helm value override that specifies the OpenSearch tier to workers
	OpenSearchTier = "OPENSEARCH_TIER"

	MasterTier = "master"
	DataTier   = "data"
	IngestTier = "ingest"
)

// IsOSReady checks if the OpenSearch resources are ready
func IsOSReady(ctrlRuntimeClient client.Client, cr *vzv1alpha1.Verrazzano) bool {
	prefix := fmt.Sprintf("Component %s", componentName)
	for _, node := range cr.Spec.Components.Elasticsearch.Nodes {
		if !IsOSNodeReady(ctrlRuntimeClient, node, prefix) {
			return false
		}
	}
	//	return common.IsVMISecretReady(ctx)
	return true
}

// IsOSNodeReady returns true if the OpenSearch tier is ready
func IsOSNodeReady(client client.Client, node vzv1alpha1.OpenSearchNode, prefix string) bool {
	if node.Replicas < 1 {
		return true
	}
	nodeControllerName := getNodeControllerName(node)

	// If a node has the master role, it is a statefulset
	if hasRole(node.Roles, vmov1.MasterRole) {
		return ready.StatefulSetsAreReady(vzlog.DefaultLogger(), client, []types.NamespacedName{{
			Name:      nodeControllerName,
			Namespace: constants.VerrazzanoSystemNamespace,
		}}, node.Replicas, prefix)
	}

	// Data nodes have N = node.Replicas number of deployment objects.
	if hasRole(node.Roles, vmov1.DataRole) {
		return ready.DeploymentsAreReady(vzlog.DefaultLogger(), client, dataDeploymentObjectKeys(node, nodeControllerName), 1, prefix)
	}

	// Ingest nodes can be handled like normal deployments
	return ready.DeploymentsAreReady(vzlog.DefaultLogger(), client, []types.NamespacedName{{
		Name:      nodeControllerName,
		Namespace: constants.VerrazzanoSystemNamespace,
	}}, node.Replicas, prefix)
}

func getNodeControllerName(node vzv1alpha1.OpenSearchNode) string {
	return fmt.Sprintf(nodeNamePrefix, node.Name)
}

func hasRole(roles []vmov1.NodeRole, roleToHave vmov1.NodeRole) bool {
	for _, role := range roles {
		if role == roleToHave {
			return true
		}
	}
	return false
}

func dataDeploymentObjectKeys(node vzv1alpha1.OpenSearchNode, nodeControllerName string) []types.NamespacedName {
	var dataDeployments []types.NamespacedName
	var i int32
	for i = 0; i < node.Replicas; i++ {
		dataDeploymentName := fmt.Sprintf("%s-%d", nodeControllerName, i)
		dataDeployments = append(dataDeployments, types.NamespacedName{
			Name:      dataDeploymentName,
			Namespace: constants.VerrazzanoSystemNamespace,
		})
	}
	return dataDeployments
}

// GetPodsForTier returns pods for a given tier
func GetPodsForTier(ctrlRuntimeClient client.Client, tier string) ([]corev1.Pod, error) {
	var label string
	switch tier {
	case MasterTier:
		label = "opensearch.verrazzano.io/role-master"
	case DataTier:
		label = "opensearch.verrazzano.io/role-data"
	case IngestTier:
		label = "opensearch.verrazzano.io/role-ingest"
	}
	req, _ := labels.NewRequirement(label, selection.Equals, []string{"true"})
	//	req, _ := labels.NewRequirement(label, selection.Exists, []string{})

	pods, err := getPodsByLabels(ctrlRuntimeClient, constants.VerrazzanoSystemNamespace, *req)
	if err != nil {
		return nil, err
	}
	return pods, nil
}

// getConfigMapsByLabels gets the configmaps by label
func getPodsByLabels(ctrlRuntimeClient client.Client, namespace string, requirements ...labels.Requirement) ([]corev1.Pod, error) {
	// Find the scenario configmaps in the cluster
	selector := labels.NewSelector()
	for _, req := range requirements {
		selector = selector.Add(req)
	}
	podList := corev1.PodList{}
	err := ctrlRuntimeClient.List(context.TODO(), &podList, &client.ListOptions{Namespace: namespace, LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

// ValidateOpenSeachTier validates the envvar is a correct opensearch tier
func ValidateOpenSeachTier() (string, error) {
	tier := config.PsrEnv.GetEnv(OpenSearchTier)
	if tier != MasterTier && tier != DataTier && tier != IngestTier {
		return "", fmt.Errorf("error, %s not a valid OpenSearch tier to restart", tier)
	}
	return tier, nil
}
