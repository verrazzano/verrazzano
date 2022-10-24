// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/types"
	"os/exec"
)

const (
	esDataDeployment   = "vmi-system-es-data"
	esIngestDeployment = "vmi-system-es-ingest"

	esMasterStatefulset = "vmi-system-es-master"
	nodeNamePrefix      = "vmi-system-%s"
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
	return ready.DoStatefulSetsExist(ctx.Log(), ctx.Client(), sts, 1, prefix)
}

// IsSingleDataNodeCluster returns true if there is exactly 1 or 0 data nodes
func IsSingleDataNodeCluster(ctx spi.ComponentContext) bool {
	return findESReplicas(ctx, "data") <= 1
}

// isOSReady checks if the OpenSearch resources are ready
func isOSReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	if vzconfig.IsOpenSearchEnabled(ctx.EffectiveCR()) && ctx.EffectiveCR().Spec.Components.Elasticsearch != nil {
		for _, node := range ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes {
			if !isOSNodeReady(ctx, node, prefix) {
				return false
			}
		}
	}

	return common.IsVMISecretReady(ctx)
}

func isOSNodeReady(ctx spi.ComponentContext, node vzapi.OpenSearchNode, prefix string) bool {
	if node.Replicas < 1 {
		return true
	}
	nodeControllerName := fmt.Sprintf(nodeNamePrefix, node.Name)

	// If a node has the master role, it is a statefulset
	if hasRole(node.Roles, vmov1.MasterRole) {
		return ready.StatefulSetsAreReady(ctx.Log(), ctx.Client(), []types.NamespacedName{{
			Name:      nodeControllerName,
			Namespace: ComponentNamespace,
		}}, node.Replicas, prefix)
	}

	// Data nodes have N = node.Replicas number of deployment objects.
	if hasRole(node.Roles, vmov1.DataRole) {
		var dataDeployments []types.NamespacedName
		var i int32
		for i = 0; i < node.Replicas; i++ {
			dataDeploymentName := fmt.Sprintf("%s-%d", nodeControllerName, i)
			dataDeployments = append(dataDeployments, types.NamespacedName{
				Name:      dataDeploymentName,
				Namespace: ComponentNamespace,
			})
		}
		return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), dataDeployments, 1, prefix)
	}

	// Ingest nodes can be handled like normal deployments
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), []types.NamespacedName{{
		Name:      nodeControllerName,
		Namespace: ComponentNamespace,
	}}, node.Replicas, prefix)
}

func hasRole(roles []vmov1.NodeRole, roleToHave vmov1.NodeRole) bool {
	for _, role := range roles {
		if role == roleToHave {
			return true
		}
	}
	return false
}

// findESReplicas searches the ES install args to find the correct resources to search for in isReady
func findESReplicas(ctx spi.ComponentContext, nodeType vmov1.NodeRole) int32 {
	var replicas int32
	if vzconfig.IsOpenSearchEnabled(ctx.EffectiveCR()) && ctx.EffectiveCR().Spec.Components.Elasticsearch != nil {
		for _, node := range ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes {
			for _, role := range node.Roles {
				if role == nodeType {
					replicas += node.Replicas
				}
			}
		}
	}
	return replicas
}
