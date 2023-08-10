// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	opensearchOperatorDeploymentName = "opensearch-operator-controller-manager"
	opensearchHostName               = "opensearch.vmi.system"
	osdHostName                      = "osd.vmi.system"
	osIngressName                    = "opensearch"
	osdIngressName                   = "opensearch-dashboards"
)

var (
	clusterCertificates = []types.NamespacedName{
		{Name: fmt.Sprintf("%s-admin-cert", clusterName), Namespace: ComponentNamespace},
		{Name: fmt.Sprintf("%s-dashboards-cert", clusterName), Namespace: ComponentNamespace},
		{Name: fmt.Sprintf("%s-master-cert", clusterName), Namespace: ComponentNamespace},
		{Name: fmt.Sprintf("%s-node-cert", clusterName), Namespace: ComponentNamespace}}

	dashboardDeployment        = fmt.Sprintf("%s-dashboards", clusterName)
	GetControllerRuntimeClient = GetClient

	clusterGVR = schema.GroupVersionResource{
		Group:    "opensearch.opster.io",
		Resource: "opensearchclusters",
		Version:  "v1",
	}

	roleGVR = schema.GroupVersionResource{
		Group:    "opensearch.opster.io",
		Resource: "opensearchroles",
		Version:  "v1",
	}

	rolesMappingGVR = schema.GroupVersionResource{
		Group:    "opensearch.opster.io",
		Resource: "opensearchuserrolebindings",
		Version:  "v1",
	}

	gvrList = []schema.GroupVersionResource{clusterGVR, roleGVR, rolesMappingGVR}
)

// isReady checks if all the sts and deployments for OpenSearch are ready or not
func (o opensearchOperatorComponent) isReady(ctx spi.ComponentContext) bool {
	nodePools, err := GetMergedNodePools(ctx)
	if err != nil {
		return false
	}

	for _, node := range nodePools {
		if node.Replicas <= 0 {
			continue
		}
		sts := []types.NamespacedName{{
			Namespace: ComponentNamespace,
			Name:      fmt.Sprintf("%s-%s", clusterName, node.Component),
		}}
		if !ready.StatefulSetsAreReady(ctx.Log(), ctx.Client(), sts, node.Replicas, getPrefix(ctx)) {
			return false
		}
	}
	deployments := getEnabledDeployments(ctx)
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, getPrefix(ctx))
}

// deleteRelatedResource deletes the resources handled by the opensearchOperator
// Like OpenSearchRoles, OpenSearchUserRolesBindings
// Since the operator adds a finalizer to these resources, they need to deleted before the operator is uninstalled
func (o opensearchOperatorComponent) deleteRelatedResource() error {
	client, err := pkg.GetDynamicClient()
	if err != nil {
		return err
	}

	for _, gvr := range gvrList {
		resourceClient := client.Resource(gvr)
		objList, err := resourceClient.Namespace(ComponentNamespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, obj := range objList.Items {
			err = resourceClient.Namespace(ComponentNamespace).Delete(context.TODO(), obj.GetName(), metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// areRelatedResourcesDeleted checks if the related resources are deleted or not
func (o opensearchOperatorComponent) areRelatedResourcesDeleted() error {
	client, err := pkg.GetDynamicClient()
	if err != nil {
		return err
	}

	for _, gvr := range gvrList {
		resourceClient := client.Resource(gvr)
		objList, err := resourceClient.Namespace(ComponentNamespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if len(objList.Items) > 0 {
			return fmt.Errorf("waiting for all %s to be deleted", gvr.String())
		}
	}
	return nil
}

// GetMergedNodePools returns the effective list of node pools
func GetMergedNodePools(ctx spi.ComponentContext) ([]NodePool, error) {
	cr := ctx.EffectiveCR()

	resourceRequest, err := common.FindStorageOverride(cr)
	if err != nil {
		return nil, err
	}
	var existingOSConfig []NodePool
	if cr.Spec.Components.Elasticsearch != nil {
		existingOSConfig = convertOSNodesToNodePools(cr.Spec.Components.Elasticsearch.Nodes, resourceRequest)
	}

	mergedNodePoolYaml, err := MergeNodePoolOverrides(cr, ctx.Client(), existingOSConfig, resourceRequest)
	if err != nil {
		return nil, ctx.Log().ErrorfNewErr("failed to get the effective nodepool list: %v", err)
	}

	var openSearch OpenSearch
	err = yaml.Unmarshal([]byte(mergedNodePoolYaml), &openSearch)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal merged nodepool yaml")
	}

	return openSearch.NodePools, err
}

// IsSingleMasterNodeCluster returns true if the cluster has a single mater node
func IsSingleMasterNodeCluster(nodePools []NodePool) bool {
	replicas := int32(0)

	for _, node := range nodePools {
		if vzstring.SliceContainsString(node.Roles, "master") {
			replicas += node.Replicas
		} else if vzstring.SliceContainsString(node.Roles, "cluster_manager") {
			replicas += node.Replicas
		}
	}
	return replicas <= 1
}

// IsSingleDataNodeCluster returns true if the cluster has a single data node
func IsSingleDataNodeCluster(ctx spi.ComponentContext) bool {
	nodePools, err := GetMergedNodePools(ctx)
	if err != nil {
		ctx.Log().Errorf("failed to get the list of nodes for OpenSearch: %v", err)
		return false
	}
	replicas := int32(0)

	for _, node := range nodePools {
		if vzstring.SliceContainsString(node.Roles, "data") {
			replicas += node.Replicas
		}
	}
	return replicas <= 1
}

// IsUpgrade returns true if we are upgrading from <=1.6.x to 2.x
func IsUpgrade(ctx spi.ComponentContext, nodePools []NodePool) bool {
	for _, node := range nodePools {
		// If PVs with this label exists for any node pool, then it's an upgrade
		pvList, err := getPVsBasedOnLabel(ctx, opensearchNodeLabel, node.Component)
		if err != nil {
			return false
		}
		if len(pvList) > 0 {
			return true
		}
	}

	return false
}

// GetClient returns a controller runtime client for the Verrazzano resource
func GetClient() (clipkg.Client, error) {
	runtimeConfig, err := k8sutil.GetConfigFromController()
	if err != nil {
		return nil, err
	}
	return clipkg.New(runtimeConfig, clipkg.Options{Scheme: newScheme()})
}

// newScheme creates a new scheme that includes this package's object for use by client
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = vzapi.AddToScheme(scheme)
	_ = installv1beta1.AddToScheme(scheme)
	_ = clientgoscheme.AddToScheme(scheme)
	return scheme
}

// getMasterNode returns the first master node found from the list of nodes
func getMasterNode(nodes []NodePool) string {
	for _, node := range nodes {
		for _, role := range node.Roles {
			if node.Replicas <= 0 {
				continue
			}
			if role == "master" || role == "cluster_manager" {
				return node.Component
			}
		}
	}
	return ""
}

// getEnabledDeployments returns the enabled deployments for this component
func getEnabledDeployments(ctx spi.ComponentContext) []types.NamespacedName {
	deployments := []types.NamespacedName{
		{
			Name:      opensearchOperatorDeploymentName,
			Namespace: ComponentNamespace,
		},
	}
	if ok, _ := vzcr.IsOpenSearchDashboardsEnabled(ctx.EffectiveCR(), ctx.Client()); ok {
		deployments = append(deployments, types.NamespacedName{
			Namespace: ComponentNamespace,
			Name:      dashboardDeployment,
		})
	}
	return deployments
}

func buildOSHostnameForDomain(dnsDomain string) string {
	return fmt.Sprintf("%s.%s", opensearchHostName, dnsDomain)
}

func buildOSDHostnameForDomain(dnsDomain string) string {
	return fmt.Sprintf("%s.%s", osdHostName, dnsDomain)
}

func getPrefix(ctx spi.ComponentContext) string {
	return fmt.Sprintf("Component %s", ctx.GetComponent())
}
