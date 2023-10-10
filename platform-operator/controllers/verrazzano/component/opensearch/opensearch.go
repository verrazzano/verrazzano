// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"context"
	"fmt"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"k8s.io/apimachinery/pkg/types"
)

const (
	esDataDeployment   = "vmi-system-es-data"
	esIngestDeployment = "vmi-system-os-ingest"

	esMasterStatefulset = "vmi-system-es-master"
	nodeNamePrefix      = "vmi-system-%s"
)

// doesOSExist is the IsInstalled check
func doesOSExist(ctx spi.ComponentContext) bool {
	sts := types.NamespacedName{
		Name:      esMasterStatefulset,
		Namespace: ComponentNamespace,
	}
	exists, err := ready.DoesStatefulsetExist(ctx.Client(), sts)
	if err != nil {
		ctx.Log().Errorf("Component %s failed getting statefulset %v: %v", ctx.GetComponent(), sts, err)
	}
	return exists
}

// IsSingleDataNodeCluster returns true if there is exactly 1 or 0 data nodes
func IsSingleDataNodeCluster(ctx spi.ComponentContext) bool {
	return findESReplicas(ctx, "data") <= 1
}

// isOSReady checks if the OpenSearch resources are ready
func isOSReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	if vzcr.IsOpenSearchEnabled(ctx.EffectiveCR()) && ctx.EffectiveCR().Spec.Components.Elasticsearch != nil {
		for _, node := range ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes {
			if !isOSNodeReady(ctx, node, prefix) {
				return false
			}
		}
	}

	return common.IsVMISecretReady(ctx)
}

func nodesToObjectKeys(ctx spi.ComponentContext) *ready.AvailabilityObjects {
	objects := &ready.AvailabilityObjects{}
	vz := ctx.EffectiveCR()
	if vzcr.IsOpenSearchEnabled(vz) && vz.Spec.Components.Elasticsearch != nil {
		isLegacyOS, err := common.IsLegacyOS(ctx)
		if err != nil {
			ctx.Log().ErrorfThrottled("Failed to get VMI, considering legacy OS to be disabled: %v", err)
		}
		ns := ComponentNamespace
		if !isLegacyOS {
			ns = constants.VerrazzanoLoggingNamespace
		}
		for _, node := range vz.Spec.Components.Elasticsearch.Nodes {
			if node.Replicas == nil || *node.Replicas < 1 {
				continue
			}
			nodeControllerName := getNodeControllerName(node, isLegacyOS)
			if !isLegacyOS || hasRole(node.Roles, vmov1.MasterRole) {
				objects.StatefulsetNames = append(objects.StatefulsetNames, types.NamespacedName{
					Name:      nodeControllerName,
					Namespace: ns,
				})
				continue
			}
			if hasRole(node.Roles, vmov1.DataRole) {
				objects.DeploymentNames = append(objects.DeploymentNames, dataDeploymentObjectKeys(node, nodeControllerName)...)
				continue
			}
			objects.DeploymentNames = append(objects.DeploymentNames, types.NamespacedName{
				Name:      nodeControllerName,
				Namespace: ComponentNamespace,
			})
		}
	}
	return objects
}

func isOSNodeReady(ctx spi.ComponentContext, node vzapi.OpenSearchNode, prefix string) bool {
	if node.Replicas == nil || *node.Replicas < 1 {
		return true
	}
	isLegacyOS, err := common.IsLegacyOS(ctx)
	if err != nil {
		ctx.Log().ErrorfThrottled("Failed to get VMI, considering legacy OS to be disabled: %v", err)
	}
	ns := ComponentNamespace
	if !isLegacyOS {
		ns = constants.VerrazzanoLoggingNamespace
	}
	nodeControllerName := getNodeControllerName(node, isLegacyOS)

	// If a node has the master role, it is a statefulset
	// If the opster operator is managing OpenSearch, then all nodes are statefulset
	if !isLegacyOS || hasRole(node.Roles, vmov1.MasterRole) {
		return AreOpensearchStsReady(ctx.Log(), ctx.Client(), []types.NamespacedName{{
			Name:      nodeControllerName,
			Namespace: ns,
		}}, *node.Replicas, prefix)
	}

	// Data nodes have N = node.Replicas number of deployment objects.
	if hasRole(node.Roles, vmov1.DataRole) {
		return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), dataDeploymentObjectKeys(node, nodeControllerName), 1, prefix)
	}

	// Ingest nodes can be handled like normal deployments
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), []types.NamespacedName{{
		Name:      nodeControllerName,
		Namespace: ComponentNamespace,
	}}, *node.Replicas, prefix)
}

func getNodeControllerName(node vzapi.OpenSearchNode, isLegacyOS bool) string {
	if isLegacyOS {
		return fmt.Sprintf(nodeNamePrefix, node.Name)
	}
	return fmt.Sprintf("opensearch-%s", node.Name)
}

func dataDeploymentObjectKeys(node vzapi.OpenSearchNode, nodeControllerName string) []types.NamespacedName {
	var dataDeployments []types.NamespacedName
	if node.Replicas == nil {
		return dataDeployments
	}
	var i int32
	for i = 0; i < *node.Replicas; i++ {
		dataDeploymentName := fmt.Sprintf("%s-%d", nodeControllerName, i)
		dataDeployments = append(dataDeployments, types.NamespacedName{
			Name:      dataDeploymentName,
			Namespace: ComponentNamespace,
		})
	}
	return dataDeployments
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
	if vzcr.IsOpenSearchEnabled(ctx.EffectiveCR()) && ctx.EffectiveCR().Spec.Components.Elasticsearch != nil {
		for _, node := range ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes {
			for _, role := range node.Roles {
				if role == nodeType && node.Replicas != nil {
					replicas += *node.Replicas
				}
			}
		}
	}
	return replicas
}

// AreOpensearchStsReady Check that the OS statefulsets have the minimum number of specified replicas ready and available. It ignores the updated replicas check if updated replicas are zero or cluster is not healthy.
func AreOpensearchStsReady(log vzlog.VerrazzanoLogger, client client.Client, namespacedNames []types.NamespacedName, expectedReplicas int32, prefix string) bool {
	for _, namespacedName := range namespacedNames {
		statefulset := appsv1.StatefulSet{}
		if err := client.Get(context.TODO(), namespacedName, &statefulset); err != nil {
			if errors.IsNotFound(err) {
				log.Progressf("%s is waiting for statefulset %v to exist", prefix, namespacedName)
				// StatefulSet not found
				return false
			}
			log.Errorf("Failed getting statefulset %v: %v", namespacedName, err)
			return false
		}
		if !areOSReplicasUpdated(log, statefulset, expectedReplicas, client, prefix, namespacedName) {
			return false
		}
		if statefulset.Status.ReadyReplicas < expectedReplicas {
			log.Progressf("%s is waiting for statefulset %s replicas to be %v. Current ready replicas is %v", prefix, namespacedName,
				expectedReplicas, statefulset.Status.ReadyReplicas)
			return false
		}
		log.Oncef("%s has enough ready replicas for statefulsets %v", prefix, namespacedName)
	}
	return true
}

// areOSReplicasUpdated check whether all replicas of opensearch are updated or not. In case of yellow cluster status, we skip this check and consider replicas are updated.
func areOSReplicasUpdated(log vzlog.VerrazzanoLogger, statefulset appsv1.StatefulSet, expectedReplicas int32, client client.Client, prefix string, namespacedName types.NamespacedName) bool {
	if statefulset.Status.UpdatedReplicas > 0 && statefulset.Status.UpdatedReplicas < expectedReplicas {
		pas, err := GetVerrazzanoPassword(client)
		if err != nil {
			log.Errorf("Failed getting OS secret to check OS cluster health: %v", err)
			return false
		}
		osClient := NewOSClient(pas)
		healthy, err := osClient.IsClusterHealthy(client)
		if err != nil {
			log.Errorf("Failed getting OpenSearch cluster health: %v", err)
			return false
		}
		if !healthy {
			log.Progressf("Skipping updated replicas check for OpenSearch because cluster health is not green")
			return true
		}
		log.Progressf("%s is waiting for statefulset %s replicas to be %v. Current updated replicas is %v", prefix, namespacedName,
			expectedReplicas, statefulset.Status.UpdatedReplicas)
		return false
	}
	return true
}
