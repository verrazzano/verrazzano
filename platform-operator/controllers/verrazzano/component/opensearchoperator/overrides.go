// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	vzyaml "github.com/verrazzano/verrazzano/pkg/yaml"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/override"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type OpenSearch struct {
	OpenSearchCluster `json:"openSearchCluster"`
}

type OpenSearchCluster struct {
	NodePools []NodePool `json:"nodePools" patchStrategy:"merge,retainKeys" patchMergeKey:"component"`
}

// NodePool defines the nodepool configuration
// The following struct are taken from upstream as it is not possible to import their API at the moment
type NodePool struct {
	Component                 string                            `json:"component"`
	Replicas                  int32                             `json:"replicas"`
	DiskSize                  string                            `json:"diskSize,omitempty"`
	Resources                 corev1.ResourceRequirements       `json:"resources,omitempty"`
	Jvm                       string                            `json:"jvm,omitempty"`
	Roles                     []string                          `json:"roles"`
	Tolerations               []corev1.Toleration               `json:"tolerations,omitempty"`
	NodeSelector              map[string]string                 `json:"nodeSelector,omitempty"`
	Affinity                  *corev1.Affinity                  `json:"affinity,omitempty"`
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
	Persistence               *PersistenceConfig                `json:"persistence,omitempty"`
	AdditionalConfig          map[string]string                 `json:"additionalConfig,omitempty"`
	Labels                    map[string]string                 `json:"labels,omitempty"`
	Annotations               map[string]string                 `json:"annotations,omitempty"`
	Env                       []corev1.EnvVar                   `json:"env,omitempty"`
	PriorityClassName         string                            `json:"priorityClassName,omitempty"`
}

// PersistenceConfig defines options for data persistence
type PersistenceConfig struct {
	PersistenceSource `json:","`
}

type PersistenceSource struct {
	PVC      *PVCSource                   `json:"pvc,omitempty"`
	EmptyDir *corev1.EmptyDirVolumeSource `json:"emptyDir,omitempty"`
	HostPath *corev1.HostPathVolumeSource `json:"hostPath,omitempty"`
}

type PVCSource struct {
	StorageClassName string                              `json:"storageClass,omitempty"`
	AccessModes      []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`
}

// GetOverrides gets the install overrides
// Since nodePools are a list and not a map they are replaced when user overrides via the CR
// So merge the nodePools and append the effective node pool overrides at the top so, it has the highest precedence
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*v1alpha1.Verrazzano); ok {
		if effectiveCR.Spec.Components.OpenSearchOperator != nil {
			mergeNodePoolOverride := BuildNodePoolOverrides(effectiveCR)
			overrides := effectiveCR.Spec.Components.OpenSearchOperator.ValueOverrides
			if mergeNodePoolOverride != nil {
				mergeNodePoolOverride = append(mergeNodePoolOverride, overrides...)
				return mergeNodePoolOverride
			}
			return overrides
		}
		return []v1alpha1.Overrides{}
	} else if effectiveCR, ok := object.(*v1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.OpenSearchOperator != nil {
			mergeNodePoolOverridev1beta1 := Buildv1beta1NodePoolOverrides(effectiveCR)
			overrides := effectiveCR.Spec.Components.OpenSearchOperator.ValueOverrides
			if mergeNodePoolOverridev1beta1 != nil {
				mergeNodePoolOverridev1beta1 = append(mergeNodePoolOverridev1beta1, overrides...)
				return mergeNodePoolOverridev1beta1
			}
			return overrides
		}
		return []v1beta1.Overrides{}
	}

	return []v1alpha1.Overrides{}
}

// BuildNodePoolOverrides builds the node pool overrides
func BuildNodePoolOverrides(cr *v1alpha1.Verrazzano) []v1alpha1.Overrides {
	var mergedOverrides []v1alpha1.Overrides
	client, err := GetControllerRuntimeClient()
	if err != nil {
		return mergedOverrides
	}

	var existingOSConfig []NodePool
	if cr.Spec.Components.Elasticsearch != nil {
		existingOSConfig = convertOSNodesToNodePools(cr.Spec.Components.Elasticsearch.Nodes)
	}

	mergedYaml, err := MergeNodePoolOverrides(cr, client, existingOSConfig)
	if err != nil {
		return mergedOverrides
	}

	// Convert back to overrides object
	data, err := yaml.YAMLToJSON([]byte(mergedYaml))
	if err != nil {
		return mergedOverrides
	}
	mergedOverrides = []v1alpha1.Overrides{
		{
			Values: &apiextensionsv1.JSON{
				Raw: data,
			},
		}}
	return mergedOverrides
}

// Buildv1beta1NodePoolOverrides builds the node pool overrides
func Buildv1beta1NodePoolOverrides(cr *v1beta1.Verrazzano) []v1beta1.Overrides {
	var mergedOverrides []v1beta1.Overrides
	client, err := GetControllerRuntimeClient()
	if err != nil {
		return mergedOverrides
	}

	var existingOSConfig []NodePool
	if cr.Spec.Components.OpenSearch != nil {
		existingOSConfig = convertv1beta1OSNodesToNodePools(cr.Spec.Components.OpenSearch.Nodes)
	}
	mergedYaml, err := MergeNodePoolOverrides(cr, client, existingOSConfig)
	if err != nil {
		return mergedOverrides
	}

	// Convert back to overrides object
	data, err := yaml.YAMLToJSON([]byte(mergedYaml))
	if err != nil {
		return mergedOverrides
	}
	mergedOverrides = []v1beta1.Overrides{
		{
			Values: &apiextensionsv1.JSON{
				Raw: data,
			},
		}}
	return mergedOverrides
}

// MergeNodePoolOverrides merges the openSearchCluster.nodePools overrides
// Returns a single yaml string with merged node pools
// Since nodePools are a list and not a map they are replaced when user overrides via the CR
// To prevent that, all the openSearchCluster.nodePools overrides are merged here
// Precedence (from high to low):
// 1. User provided overrides
// 2. Configuration from current OpenSearch and OpenSearchDashboards components
// 3. Default configuration from the base profiles
func MergeNodePoolOverrides(object runtime.Object, client client.Client, existingOSConfig []NodePool) (string, error) {

	var overridesYaml []string
	var err error

	if effectiveCR, ok := object.(*v1alpha1.Verrazzano); ok {
		if effectiveCR.Spec.Components.OpenSearchOperator != nil {
			operatorOverrides := v1alpha1.ConvertValueOverridesToV1Beta1(effectiveCR.Spec.Components.OpenSearchOperator.ValueOverrides)
			overridesYaml, err = override.GetInstallOverridesYAMLUsingClient(client, operatorOverrides, effectiveCR.Namespace)
		}
	} else if effectiveCR, ok := object.(*v1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.OpenSearchOperator != nil {
			operatorOverrides := effectiveCR.Spec.Components.OpenSearchOperator.ValueOverrides
			overridesYaml, err = override.GetInstallOverridesYAMLUsingClient(client, operatorOverrides, effectiveCR.Namespace)
		}
	}

	if err != nil || len(overridesYaml) <= 0 {
		return "", err
	}

	// First merge the default override from the dev/prod manifest and the existing configuration from Elasticsearch/OpenSearch Component
	// The Elasticsearch/OpenSearch Component will later be converted to overrides as part of v1beta2 CR conversion
	// So this extra step won't be required here

	defaultOverrides := overridesYaml[len(overridesYaml)-1] // Default overrides are last in the list
	var existingOSYaml []byte

	if len(existingOSConfig) > 0 {
		existingOSYaml, err = yaml.Marshal(OpenSearch{
			OpenSearchCluster{NodePools: existingOSConfig},
		})
		if err != nil {
			return "", err
		}
	}

	mergedNodePools, err := vzyaml.StrategicMerge(OpenSearch{}, defaultOverrides, string(existingOSYaml))
	if err != nil {
		return "", err
	}

	// Merge the rest of user provided overrides
	for i := len(overridesYaml) - 2; i >= 0; i-- {
		mergedNodePools, err = vzyaml.StrategicMerge(OpenSearch{}, mergedNodePools, overridesYaml[i])
		if err != nil {
			return "", err
		}
	}

	return mergedNodePools, nil
}

// AppendOverrides appends the additional overrides for install
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// TODO: Image overrides once the BFS images are done

	nodePools, err := GetMergedNodePools(ctx)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to get the list of merged nodepools: %v", err)
	}
	// Bootstrap pod overrides
	if IsUpgrade(ctx, nodePools) || IsSingleMasterNodeCluster(nodePools) {
		kvs = append(kvs, bom.KeyValue{
			Key:   `openSearchCluster.bootstrap.additionalConfig.cluster\.initial_master_nodes`,
			Value: fmt.Sprintf("%s-%s-0", clusterName, getMasterNode(nodePools)),
		})
	}

	kvs, err = buildIngressOverrides(ctx, kvs)
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
				Key:   "openSearchCluster.dashboards.replicas",
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
					Key:   fmt.Sprintf("openSearchCluster.general.pluginsList[%d]", i),
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
					Key:   fmt.Sprintf("openSearchCluster.dashboards.pluginsList[%d]", i),
					Value: plugin,
				})
			}
		}
	}

	return kvs, nil
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
			Key:   "ingress.openSearch.enable",
			Value: "false",
		})
		kvs = append(kvs, bom.KeyValue{
			Key:   "ingress.openSearchDashboards.enable",
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
		Key:   "ingress.openSearchDashboards.ingressClassName",
		Value: ingressClassName,
	})
	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearchDashboards.host",
		Value: osdHostName,
	})

	annotationsKey := "ingress.openSearchDashboards.annotations"
	for key, value := range ingressAnnotations {
		kvs = append(kvs, bom.KeyValue{
			Key:       fmt.Sprintf("%s.%s", annotationsKey, key),
			Value:     value,
			SetString: true,
		})
	}

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearchDashboards.tls[0].secretName",
		Value: "system-tls-osd",
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearchDashboards.tls[0].hosts[0]",
		Value: osdHostName,
	})

	return kvs, nil
}

// appendOSIngressOverrides appends the additional overrides for OpenSearch ingress
func appendOSIngressOverrides(ingressAnnotations map[string]string, dnsSubDomain, ingressClassName string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	opensearchHostName := buildOSHostnameForDomain(dnsSubDomain)
	ingressAnnotations[`cert-manager\.io/common-name`] = opensearchHostName

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearch.ingressClassName",
		Value: ingressClassName,
	})
	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearch.host",
		Value: opensearchHostName,
	})

	annotationsKey := "ingress.openSearch.annotations"
	for key, value := range ingressAnnotations {
		kvs = append(kvs, bom.KeyValue{
			Key:       fmt.Sprintf("%s.%s", annotationsKey, key),
			Value:     value,
			SetString: true,
		})
	}

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearch.tls[0].secretName",
		Value: "system-tls-os-ingest",
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearch.tls[0].hosts[0]",
		Value: opensearchHostName,
	})

	return kvs, nil
}

// convertOSNodesToNodePools converts OpenSearchNode to NodePool type
func convertOSNodesToNodePools(nodes []v1alpha1.OpenSearchNode) []NodePool {
	var nodePools []NodePool
	for _, node := range nodes {
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
		} else {
			nodePool.Persistence = &PersistenceConfig{
				PersistenceSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}
		}
		for _, role := range node.Roles {
			nodePool.Roles = append(nodePool.Roles, string(role))
		}
		nodePools = append(nodePools, nodePool)
	}
	return nodePools
}

// convertv1beta1OSNodesToNodePools converts OpenSearchNode to NodePool type
func convertv1beta1OSNodesToNodePools(nodes []v1beta1.OpenSearchNode) []NodePool {
	var nodePools []NodePool
	for _, node := range nodes {
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
		} else {
			nodePool.Persistence = &PersistenceConfig{
				PersistenceSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}
		}
		for _, role := range node.Roles {
			nodePool.Roles = append(nodePool.Roles, string(role))
		}
		nodePools = append(nodePools, nodePool)
	}
	return nodePools
}
