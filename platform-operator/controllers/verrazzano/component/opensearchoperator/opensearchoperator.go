// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"io/fs"
	corev1 "k8s.io/api/core/v1"
	"os"
	"sigs.k8s.io/yaml"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
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

	dashboardDeployment        = fmt.Sprintf("%s-dashboards", clusterName)
	GetControllerRuntimeClient = GetClient
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

// AppendOverrides appends the additional overrides for install
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// TODO: Image overrides once the BFS images are done

	// Bootstrap pod overrides
	if IsUpgrade(ctx) || IsSingleMasterNodeCluster(ctx) {
		kvs = append(kvs, bom.KeyValue{
			Key:   `opensearchCluster.bootstrap.additionalConfig.cluster\.initial_master_nodes`,
			Value: fmt.Sprintf("%s-%s-0", clusterName, getMasterNode(ctx)),
		})
	}

	kvs, err := buildIngressOverrides(ctx, kvs)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to build ingress overrides: %v", err)
	}

	// Append OSD replica count from current OSD config
	// This will later go as part CR conversion
	osd := ctx.EffectiveCR().Spec.Components.Kibana
	if osd != nil {
		osdReplica := osd.Replicas
		if osdReplica != nil {
			kvs = append(kvs, bom.KeyValue{
				Key:   "opensearchCluster.dashboards.replicas",
				Value: fmt.Sprint(*osdReplica),
			})
		}
	}

	// Append plugins list
	// This will later go as part of CR conversion
	os := ctx.EffectiveCR().Spec.Components.Elasticsearch
	if os != nil {
		osPlugins := os.Plugins
		if osPlugins.Enabled && len(osPlugins.InstallList) > 0 {
			for i, plugin := range osPlugins.InstallList {
				kvs = append(kvs, bom.KeyValue{
					Key:   fmt.Sprintf("opensearchCluster.general.pluginsList[%d]", i),
					Value: plugin,
				})
			}
		}
	}

	if osd != nil {
		osdPlugins := osd.Plugins
		if osdPlugins.Enabled && len(osdPlugins.InstallList) > 0 {
			for i, plugin := range osdPlugins.InstallList {
				kvs = append(kvs, bom.KeyValue{
					Key:   fmt.Sprintf("opensearchCluster.dashboards.pluginsList[%d]", i),
					Value: plugin,
				})
			}
		}
	}

	kvs, err = buildNodePoolOverrides(ctx, kvs)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to build nodepool overrides: %v", err)
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

func buildNodePoolOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	convertedNodes, err := convertOSNodesToNodePools(ctx)
	if err != nil {
		return kvs, err
	}

	nodePoolOverrides, err := yaml.Marshal(OpenSearch{
		OpenSearchCluster{NodePools: convertedNodes},
	})
	if err != nil {
		return kvs, err
	}

	file, err := os.CreateTemp(os.TempDir(), tmpFileCreatePattern)
	if err != nil {
		return kvs, err
	}

	overridesFileName := file.Name()
	if err := os.WriteFile(overridesFileName, nodePoolOverrides, fs.ModeAppend); err != nil {
		return kvs, err
	}

	kvs = append(kvs, bom.KeyValue{Value: overridesFileName, IsFile: true})
	return kvs, nil
}

// convertOSNodesToNodePools converts OpenSearchNode to NodePool type
func convertOSNodesToNodePools(ctx spi.ComponentContext) ([]NodePool, error) {
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
			"cluster.initial_master_nodes": fmt.Sprintf("%s-%s-0", clusterName, nodePools[0].Component),
		}
	}
	return nodePools, nil
}

func getActualCRNodes(cr *vzapi.Verrazzano) map[string]vzapi.OpenSearchNode {
	nodeMap := map[string]vzapi.OpenSearchNode{}
	if cr != nil && cr.Spec.Components.Elasticsearch != nil {
		for _, node := range cr.Spec.Components.Elasticsearch.Nodes {
			nodeMap[node.Name] = node
		}
	}
	return nodeMap
}

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

		kvs, _ = appendOSIngressOverrides(ingressAnnotations, dnsSubDomain, ingressClassName, kvs)
		kvs, _ = appendOSDIngressOverrides(ingressAnnotations, dnsSubDomain, ingressClassName, kvs)

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

// appendOSDIngressOverrides appends the additional overrides for OpenSearchDashboards ingress
func appendOSDIngressOverrides(ingressAnnotations map[string]string, dnsSubDomain, ingressClassName string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	osdHostName := buildOSDHostnameForDomain(dnsSubDomain)
	ingressAnnotations[`cert-manager\.io/common-name`] = osdHostName

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.opensearchDashboards.ingressClassName",
		Value: ingressClassName,
	})
	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.opensearchDashboards.host",
		Value: osdHostName,
	})

	annotationsKey := "ingress.opensearchDashboards.annotations"
	for key, value := range ingressAnnotations {
		kvs = append(kvs, bom.KeyValue{
			Key:       fmt.Sprintf("%s.%s", annotationsKey, key),
			Value:     value,
			SetString: true,
		})
	}

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.opensearchDashboards.tls[0].secretName",
		Value: "system-tls-osd",
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.opensearchDashboards.tls[0].hosts[0]",
		Value: osdHostName,
	})

	return kvs, nil
}

// appendOSIngressOverrides appends the additional overrides for OpenSearch ingress
func appendOSIngressOverrides(ingressAnnotations map[string]string, dnsSubDomain, ingressClassName string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	opensearchHostName := buildOSHostnameForDomain(dnsSubDomain)
	ingressAnnotations[`cert-manager\.io/common-name`] = opensearchHostName

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.opensearch.ingressClassName",
		Value: ingressClassName,
	})
	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.opensearch.host",
		Value: opensearchHostName,
	})

	annotationsKey := "ingress.opensearch.annotations"
	for key, value := range ingressAnnotations {
		kvs = append(kvs, bom.KeyValue{
			Key:       fmt.Sprintf("%s.%s", annotationsKey, key),
			Value:     value,
			SetString: true,
		})
	}

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.opensearch.tls[0].secretName",
		Value: "system-tls-os-ingest",
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.opensearch.tls[0].hosts[0]",
		Value: opensearchHostName,
	})

	return kvs, nil
}

// isReady checks if all the sts and deployments for OpenSearch are ready or not
func (o opensearchOperatorComponent) isReady(ctx spi.ComponentContext) bool {
	deployments := getEnabledDeployments(ctx)
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, getPrefix(ctx))
}

// IsSingleMasterNodeCluster returns true if the cluster has a single mater node
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
