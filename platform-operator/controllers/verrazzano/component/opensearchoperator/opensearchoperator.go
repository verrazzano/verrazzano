// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	tmpFilePrefix                    = "opensearch-operator-overrides-"
	tmpSuffix                        = "yaml"
	tmpFileCreatePattern             = tmpFilePrefix + "*." + tmpSuffix
)

var (
	clusterCertificates = []types.NamespacedName{
		{Name: fmt.Sprintf("%s-admin-cert", clusterName), Namespace: ComponentNamespace},
		{Name: fmt.Sprintf("%s-dashboards-cert", clusterName), Namespace: ComponentNamespace},
		{Name: fmt.Sprintf("%s-master-cert", clusterName), Namespace: ComponentNamespace},
		{Name: fmt.Sprintf("%s-node-cert", clusterName), Namespace: ComponentNamespace}}

	GetControllerRuntimeClient = GetClient
	clusterGVR                 = schema.GroupVersionResource{
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

// GetOverrides gets the list of overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.OpenSearchOperator != nil {
			return effectiveCR.Spec.Components.OpenSearchOperator.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.OpenSearchOperator != nil {
			return effectiveCR.Spec.Components.OpenSearchOperator.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

func buildArgsForOpenSearchCR(ctx spi.ComponentContext) (map[string]interface{}, error) {
	args := make(map[string]interface{})

	masterNode := getMasterNode(ctx)
	if len(masterNode) <= 0 {
		return args, fmt.Errorf("invalid cluster topology, no master node defined")
	}
	effectiveCR := ctx.EffectiveCR()

	args["isOpenSearchEnabled"] = vzcr.IsOpenSearchEnabled(effectiveCR)
	args["isOpenSearchDashboardsEnabled"] = vzcr.IsOpenSearchDashboardsEnabled(effectiveCR)

	// Bootstrap pod overrides
	args["bootstrapConfig"] = ""
	if IsUpgrade(ctx) || IsSingleMasterNodeCluster(ctx) {
		args["bootstrapConfig"] = fmt.Sprintf("cluster.initial_master_nodes: %s-%s-0", clusterName, masterNode)
	}

	// Set drainDataNodes only when the cluster has at least 2 data ndoes
	if opensearch.IsSingleDataNodeCluster(ctx) {
		args["drainDataNodes"] = false
	} else {
		args["drainDataNodes"] = true
	}

	// Append OSD replica count from current OSD config
	osd := effectiveCR.Spec.Components.Kibana
	if osd != nil {
		osdReplica := osd.Replicas
		if osdReplica != nil {
			args["osdReplicas"] = osdReplica
		}
	}

	// Append plugins list for Opensearch
	opensearch := effectiveCR.Spec.Components.Elasticsearch
	if opensearch != nil {
		osPlugins := opensearch.Plugins
		if osPlugins.Enabled && len(osPlugins.InstallList) > 0 {
			pluginList, err := yaml.Marshal(osPlugins.InstallList)
			if err != nil {
				return args, nil
			}
			args["osPluginsEnabled"] = true
			args["osPluginsList"] = string(pluginList)
		} else {
			args["osPluginsEnabled"] = false
		}
	}

	// Append plugins list for OSD
	if osd != nil {
		osdPlugins := osd.Plugins
		if osdPlugins.Enabled && len(osdPlugins.InstallList) > 0 {
			pluginList, err := yaml.Marshal(osdPlugins.InstallList)
			if err != nil {
				return args, nil
			}
			args["osdPluginsEnabled"] = true
			args["osdPluginsList"] = string(pluginList)
		} else {
			args["osdPluginsEnabled"] = false
		}
	}

	err := buildNodePoolOverrides(ctx, args, masterNode)
	if err != nil {
		return args, ctx.Log().ErrorfNewErr("Failed to build nodepool overrides: %v", err)
	}

	// Test images
	// TODO:- Get from BOM once BFS images are done
	args["opensearchImage"] = "iad.ocir.io/odsbuilddev/sandboxes/saket.m.mahto/opensearch-security:experimental"
	args["osdImage"] = "iad.ocir.io/odsbuilddev/sandboxes/isha.girdhar/osd:latest"

	return args, nil
}

// appendOverrides appends the additional overrides for install
func appendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {

	kvs, err := buildIngressOverrides(ctx, kvs)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to build ingress overrides: %v", err)
	}

	return kvs, nil
}

type OpenSearch struct {
	OpenSearchCluster `json:"opensearchCluster"`
}

type OpenSearchCluster struct {
	NodePools []NodePool `json:"nodePools" patchStrategy:"merge,retainKeys" patchMergeKey:"component"`
}

type NodePool struct {
	Component        string                      `json:"component"`
	Replicas         int32                       `json:"replicas"`
	DiskSize         string                      `json:"diskSize,omitempty"`
	Resources        corev1.ResourceRequirements `json:"resources,omitempty"`
	Jvm              string                      `json:"jvm,omitempty"`
	Roles            []string                    `json:"roles"`
	Persistence      *PersistenceConfig          `json:"persistence,omitempty"`
	AdditionalConfig map[string]string           `json:"additionalConfig,omitempty"`
}

// PersistenceConfig defines options for data persistence
type PersistenceConfig struct {
	PersistenceSource `json:","`
}

type PersistenceSource struct {
	EmptyDir *corev1.EmptyDirVolumeSource `json:"emptyDir,omitempty"`
}

func buildNodePoolOverrides(ctx spi.ComponentContext, args map[string]interface{}, masterNode string) error {
	convertedNodes, err := convertOSNodesToNodePools(ctx, masterNode)
	if err != nil {
		return err
	}

	nodePoolOverrides, err := yaml.Marshal(convertedNodes)
	if err != nil {
		return err
	}
	args["nodePools"] = string(nodePoolOverrides)
	return nil
}

// convertOSNodesToNodePools converts OpenSearchNode to NodePool type
func convertOSNodesToNodePools(ctx spi.ComponentContext, masterNode string) ([]NodePool, error) {
	var nodePools []NodePool

	effectiveCR := ctx.EffectiveCR()
	effectiveCRNodes := effectiveCR.Spec.Components.Elasticsearch.Nodes
	actualCRNodes := getActualCRNodes(ctx.ActualCR())
	resourceRequest, err := common.FindStorageOverride(effectiveCR)

	if err != nil {
		return nodePools, err
	}

	for _, node := range effectiveCRNodes {
		nodePool := NodePool{
			Component: node.Name,
			Jvm:       node.JavaOpts,
		}
		if node.Replicas != nil {
			nodePool.Replicas = *node.Replicas
		}
		if node.Resources != nil {
			nodePool.Resources = *node.Resources
		}
		if node.Storage != nil {
			nodePool.DiskSize = node.Storage.Size
		} else if resourceRequest != nil && len(resourceRequest.Storage) > 0 {
			nodePool.DiskSize = resourceRequest.Storage
		} else {
			nodePool.DiskSize = ""
			nodePool.Persistence = &PersistenceConfig{
				PersistenceSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}
		}
		// user defined storage has the highest precedence
		if actualCRNode, ok := actualCRNodes[node.Name]; ok {
			if actualCRNode.Storage != nil {
				nodePool.DiskSize = actualCRNode.Storage.Size
			}
		}
		for _, role := range node.Roles {
			nodePool.Roles = append(nodePool.Roles, string(role))
		}
		nodePools = append(nodePools, nodePool)
	}
	if IsSingleMasterNodeCluster(ctx) {
		nodePools[0].AdditionalConfig = map[string]string{
			"cluster.initial_master_nodes": fmt.Sprintf("%s-%s-0", clusterName, masterNode),
		}
	}
	return nodePools, nil
}

// deleteRelatedResource deletes the resources handled by the opensearchOperator
// Like OpenSearchRoles, OpenSearchUserRolesBindings
// Since the operator adds a finalizer to these resources, they need to deleted before the operator is uninstalled
func (o opensearchOperatorComponent) deleteRelatedResource() error {
	client, err := k8sutil.GetDynamicClient()
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
	client, err := k8sutil.GetDynamicClient()
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

// getActualCRNodes returns the nodes from the actual CR
func getActualCRNodes(cr *vzapi.Verrazzano) map[string]vzapi.OpenSearchNode {
	nodeMap := map[string]vzapi.OpenSearchNode{}
	if cr != nil && cr.Spec.Components.Elasticsearch != nil {
		for _, node := range cr.Spec.Components.Elasticsearch.Nodes {
			nodeMap[node.Name] = node
		}
	}
	return nodeMap
}

// getMasterNode returns the first master node from the list of nodes
func getMasterNode(ctx spi.ComponentContext) string {
	for _, node := range ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes {
		for _, role := range node.Roles {
			if node.Replicas == nil || *node.Replicas < 1 {
				continue
			}
			if role == "master" {
				return node.Name
			}
		}
	}
	return ""
}

// buildIngressOverrides builds the overrides required for the OpenSearch and OpenSearchDashboards ingresses
func buildIngressOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	if vzcr.IsNGINXEnabled(ctx.EffectiveCR()) {

		dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
		if err != nil {
			return kvs, ctx.Log().ErrorfNewErr("Failed to build DNS subdomain: %v", err)
		}
		ingressClassName := vzconfig.GetIngressClassName(ctx.EffectiveCR())
		ingressTarget := fmt.Sprintf("verrazzano-ingress.%s", dnsSubDomain)

		ingressAnnotations := make(map[string]string)
		ingressAnnotations[`cert-manager\.io/cluster-issuer`] = constants.VerrazzanoClusterIssuerName
		if vzcr.IsExternalDNSEnabled(ctx.EffectiveCR()) {
			ingressAnnotations[`external-dns\.alpha\.kubernetes\.io/target`] = ingressTarget
			ingressAnnotations[`external-dns\.alpha\.kubernetes\.io/ttl`] = "60"
		}

		tlsSecret := "system-tls-osd"
		path := "ingress.opensearchDashboards"
		kvs = appendIngressOverrides(ingressAnnotations, path, buildOSDHostnameForDomain(dnsSubDomain), tlsSecret, ingressClassName, kvs)

		tlsSecret = "system-tls-os-ingest"
		path = "ingress.opensearch"
		kvs = appendIngressOverrides(ingressAnnotations, path, buildOSHostnameForDomain(dnsSubDomain), tlsSecret, ingressClassName, kvs)

	} else {
		kvs = append(kvs, bom.KeyValue{
			Key:   "ingress.opensearch.enable",
			Value: "false",
		})
		kvs = append(kvs, bom.KeyValue{
			Key:   "ingress.opensearchDashboards.enable",
			Value: "false",
		})
	}

	return kvs, nil
}

// appendIngressOverrides appends the required overrides for the ingresses
func appendIngressOverrides(ingressAnnotations map[string]string, path, hostName, tlsSecret, ingressClassName string, kvs []bom.KeyValue) []bom.KeyValue {
	ingressAnnotations[`cert-manager\.io/common-name`] = hostName

	kvs = append(kvs, bom.KeyValue{
		Key:   fmt.Sprintf("%s.ingressClassName", path),
		Value: ingressClassName,
	})
	kvs = append(kvs, bom.KeyValue{
		Key:   fmt.Sprintf("%s.host", path),
		Value: hostName,
	})
	annotationsKey := fmt.Sprintf("%s.annotations", path)
	for key, value := range ingressAnnotations {
		kvs = append(kvs, bom.KeyValue{
			Key:       fmt.Sprintf("%s.%s", annotationsKey, key),
			Value:     value,
			SetString: true,
		})
	}
	kvs = append(kvs, bom.KeyValue{
		Key:   fmt.Sprintf("%s.tls[0].secretName", path),
		Value: tlsSecret,
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   fmt.Sprintf("%s.tls[0].hosts[0]", path),
		Value: hostName,
	})
	return kvs
}

// isReady checks if all the sts and deployments for OpenSearch are ready or not
func (o opensearchOperatorComponent) isReady(ctx spi.ComponentContext) bool {
	deployments := getEnabledDeployments(ctx)
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, getPrefix(ctx))
}

// IsSingleMasterNodeCluster returns true if the cluster has a single master node
func IsSingleMasterNodeCluster(ctx spi.ComponentContext) bool {
	var replicas int32
	for _, node := range ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes {
		for _, role := range node.Roles {
			if role == "master" && node.Replicas != nil {
				replicas += *node.Replicas
			}
		}
	}
	return replicas <= 1
}

// IsUpgrade returns true if we are upgrading from <=1.6.x to 2.x
func IsUpgrade(ctx spi.ComponentContext) bool {
	for _, node := range ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes {
		// If PVs with this label exists for any node pool, then it's an upgrade
		pvList, err := getPVsBasedOnLabel(ctx, opensearchNodeLabel, node.Name)
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

// getEnabledDeployments returns the enabled deployments for this component
func getEnabledDeployments(ctx spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{
		{
			Name:      opensearchOperatorDeploymentName,
			Namespace: ComponentNamespace,
		},
	}
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
