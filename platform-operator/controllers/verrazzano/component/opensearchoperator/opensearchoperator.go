// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/override"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
	nodePoolKey                      = "openSearchCluster.nodePools.*"
	additionalConfigKey              = "additionalConfig"
)

var (
	clusterCertificates = []types.NamespacedName{
		{Name: fmt.Sprintf("%s-admin-cert", clusterName), Namespace: ComponentNamespace},
		{Name: fmt.Sprintf("%s-dashboards-cert", clusterName), Namespace: ComponentNamespace},
		{Name: fmt.Sprintf("%s-master-cert", clusterName), Namespace: ComponentNamespace},
		{Name: fmt.Sprintf("%s-node-cert", clusterName), Namespace: ComponentNamespace}}

	GetControllerRuntimeClient = GetClient
)

type NodePool struct {
	Component string   `json:"component"`
	Replicas  int32    `json:"replicas"`
	Roles     []string `json:"roles"`
}

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
			Name:      fmt.Sprintf("%s-dashboards", clusterName),
		})
	}
	return deployments
}

// GetOverrides gets the install overrides
func GetOverrides(object runtime.Object) interface{} {

	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		mergeNodePoolOverride := BuildNodePoolOverrides(*effectiveCR)
		if effectiveCR.Spec.Components.OpenSearchOperator != nil {
			mergeNodePoolOverride = append(mergeNodePoolOverride, effectiveCR.Spec.Components.OpenSearchOperator.ValueOverrides...)
		}
		return mergeNodePoolOverride
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		mergeNodePoolOverridev1beta1 := BuildNodePoolOverridesv1beta1(*effectiveCR)
		if effectiveCR.Spec.Components.OpenSearchOperator != nil {
			mergeNodePoolOverridev1beta1 = append(mergeNodePoolOverridev1beta1, effectiveCR.Spec.Components.OpenSearchOperator.ValueOverrides...)
		}
		return mergeNodePoolOverridev1beta1
	}

	return []vzapi.Overrides{}
}

// BuildNodePoolOverrides builds the opensearchCluster.nodePools v1alpha1 overrides for the operator
// Since nodePools are a list and not a map they are replaced when overridden via the CR
// To prevent that, all the opensearchCluster.nodePools overrides are merged here
// Precedence (from high to low):
// 1. User provided overrides
// 2. Configuration from current OpenSearch and OpenSearchDashboards components -> This will later be moved to CR conversion
// once new CR version is ready
// 3. Default configuration from the base profiles
func BuildNodePoolOverrides(cr vzapi.Verrazzano) []vzapi.Overrides {

	var mergedOverrides []vzapi.Overrides

	client, err := GetControllerRuntimeClient()
	if err != nil {
		return mergedOverrides
	}

	operatorOverrides := vzapi.ConvertValueOverridesToV1Beta1(cr.Spec.Components.OpenSearchOperator.ValueOverrides)
	overrideYaml, err := override.GetInstallOverridesYAMLUsingClient(client, operatorOverrides, cr.Namespace)

	if err != nil || len(overrideYaml) <= 0 {
		return mergedOverrides
	}

	// The default overrides are the last in the list
	defaultOverrides := overrideYaml[len(overrideYaml)-1]
	value, err := override.ExtractValueFromOverrideString(defaultOverrides, nodePoolKey)
	if err != nil {
		return mergedOverrides
	}

	// Merge the node configuration from the OpenSearch component
	// This will be part of CR conversion later
	existingOSConfig := ConvertOSNodeToInterface(cr.Spec.Components.Elasticsearch.Nodes)
	value, err = MergeNodePools(existingOSConfig, value)
	if err != nil {
		return mergedOverrides
	}

	// Merge the node configuration from rest of the user provided overrides
	for i := len(overrideYaml) - 2; i >= 0; i-- {
		newValue, err := override.ExtractValueFromOverrideString(overrideYaml[i], nodePoolKey)
		if err != nil {
			return mergedOverrides
		}
		value, err = MergeNodePools(newValue, value)
		if err != nil {
			return mergedOverrides
		}
	}

	mergedOverrides = []vzapi.Overrides{
		{
			Values: &apiextensionsv1.JSON{
				Raw: CreateOverridesAsJSON(value),
			},
		}}
	return mergedOverrides
}

// BuildNodePoolOverridesv1beta1 builds the opensearchCLuster.nodePools v1beta1 overrides for the operator
func BuildNodePoolOverridesv1beta1(cr installv1beta1.Verrazzano) []installv1beta1.Overrides {

	var mergedOverrides []installv1beta1.Overrides

	client, err := GetControllerRuntimeClient()
	if err != nil {
		return mergedOverrides
	}

	operatorOverrides := cr.Spec.Components.OpenSearchOperator.ValueOverrides
	overrideYaml, err := override.GetInstallOverridesYAMLUsingClient(client, operatorOverrides, cr.Namespace)

	if err != nil || len(overrideYaml) <= 0 {
		return mergedOverrides
	}

	// The default overrides are the last in the list
	defaultOverrides := overrideYaml[len(overrideYaml)-1]
	value, err := override.ExtractValueFromOverrideString(defaultOverrides, nodePoolKey)
	if err != nil {
		return mergedOverrides
	}

	existingOSConfig := ConvertOSNodeToInterfacev1beta1(cr.Spec.Components.OpenSearch.Nodes)
	value, err = MergeNodePools(existingOSConfig, value)
	if err != nil {
		return mergedOverrides
	}

	for i := len(overrideYaml) - 2; i >= 0; i-- {
		newValue, err := override.ExtractValueFromOverrideString(overrideYaml[i], nodePoolKey)
		if err != nil {
			return mergedOverrides
		}
		value, err = MergeNodePools(newValue, value)
		if err != nil {
			return mergedOverrides
		}
	}

	mergedOverrides = []installv1beta1.Overrides{
		{
			Values: &apiextensionsv1.JSON{
				Raw: CreateOverridesAsJSON(value),
			},
		}}
	return mergedOverrides
}

func ConvertOSNodeToInterface(nodes []vzapi.OpenSearchNode) interface{} {
	var ret []interface{}
	for _, node := range nodes {
		nodeMap := make(map[string]interface{})
		nodeMap["component"] = node.Name
		if node.JavaOpts != "" {
			nodeMap["jvm"] = node.JavaOpts
		}
		if node.Replicas != nil {
			nodeMap["replicas"] = *node.Replicas
		}
		if node.Storage != nil {
			nodeMap["diskSize"] = node.Storage.Size
		} else {
			nodeMap["persistence"] = map[string]interface{}{
				"emptyDir": map[string]interface{}{}, // denotes '{}'
			}
		}
		if node.Resources != nil {
			nodeMap["resources"] = node.Resources
		}
		if len(node.Roles) > 0 {
			var roles []interface{}
			for _, role := range node.Roles {
				roles = append(roles, string(role))
			}
			nodeMap["roles"] = roles
		}
		ret = append(ret, nodeMap)
	}
	return ret
}

func ConvertOSNodeToInterfacev1beta1(nodes []installv1beta1.OpenSearchNode) interface{} {
	var ret []interface{}
	for _, node := range nodes {
		nodeMap := make(map[string]interface{})
		nodeMap["component"] = node.Name
		if node.JavaOpts != "" {
			nodeMap["jvm"] = node.JavaOpts
		}
		if node.Replicas != nil {
			nodeMap["replicas"] = *node.Replicas
		}
		if node.Storage != nil {
			nodeMap["diskSize"] = node.Storage.Size
		} else {
			nodeMap["persistence"] = map[string]interface{}{
				"emptyDir": map[string]interface{}{}, // denotes '{}'
			}
		}
		if node.Resources != nil {
			nodeMap["resources"] = node.Resources
		}
		if len(node.Roles) > 0 {
			var roles []interface{}
			for _, role := range node.Roles {
				roles = append(roles, string(role))
			}
			nodeMap["roles"] = roles
		}
		ret = append(ret, nodeMap)
	}
	return ret
}

func CreateOverridesAsJSON(value interface{}) []byte {
	// Create the nested structure for openSearchCluster.nodePools.*
	nestedMap := make(map[string]interface{})

	_, ok := value.([]interface{})
	if !ok {
		return nil
	}
	nestedMap["openSearchCluster"] = map[string]interface{}{
		"nodePools": value.([]interface{}),
	}
	mergedOverrides, err := json.Marshal(nestedMap)
	if err != nil {
		return []byte{}
	}
	return mergedOverrides
}

// MergeNodePools merges two nodepool overrides
// np1 has higher precedence and is overlayed on top on np2
func MergeNodePools(np1, np2 interface{}) (interface{}, error) {
	_, ok1 := np1.([]interface{})
	_, ok2 := np2.([]interface{})

	if !ok1 || !ok2 {
		return nil, fmt.Errorf("Failed to merge nodes")
	}

	for _, oldNode := range np2.([]interface{}) {
		oldnp, ok := oldNode.(map[string]interface{})
		if !ok {
			continue
		}
		found := false
		for _, newNode := range np1.([]interface{}) {
			newnp, ok := newNode.(map[string]interface{})
			if !ok {
				continue
			}
			if name, ok := newnp["component"]; ok {
				if oldName, ok := oldnp["component"]; ok {
					if name == oldName {
						found = true
						// If values don't exist in np1, use the values from np2
						// Else don't change
						if _, ok := newnp["replicas"]; !ok {
							newnp["replicas"] = oldnp["replicas"]
						}
						if _, ok := newnp["jvm"]; !ok {
							newnp["jvm"] = oldnp["jvm"]
						}
						if _, ok := newnp["diskSize"]; !ok {
							newnp["diskSize"] = oldnp["diskSize"]
						}
						if _, ok := newnp["resources"]; !ok {
							newnp["resources"] = oldnp["resources"]
						}
						if _, ok := newnp["roles"]; !ok {
							newnp["roles"] = oldnp["roles"]
						}
						mergeAdditionalConfig(newnp, oldnp)
					}
				}
			}
		}
		if !found {
			np1 = append(np1.([]interface{}), oldNode)
		}
	}
	return np1, nil
}

func mergeAdditionalConfig(newnp map[string]interface{}, oldnp map[string]interface{}) {
	if _, ok := newnp[additionalConfigKey]; !ok {
		if _, ok := oldnp[additionalConfigKey]; !ok {
			return
		}
		newnp[additionalConfigKey] = oldnp[additionalConfigKey]
	} else {
		additionalConfigNewnp, ok1 := newnp[additionalConfigKey].(map[string]interface{})
		additionalConfigOldnp, ok2 := oldnp[additionalConfigKey].(map[string]interface{})
		if !ok1 || !ok2 {
			return
		}
		for key, val := range additionalConfigOldnp {
			if _, ok := additionalConfigNewnp[key]; !ok {
				additionalConfigOldnp[key] = val
			}
		}
	}
}

// GetMergedNodePools returns the list of nodes after merging all the overrides
func GetMergedNodePools(ctx spi.ComponentContext) ([]NodePool, error) {
	cr := ctx.EffectiveCR()
	var nodePools []NodePool

	overrides := BuildNodePoolOverrides(*cr)
	overridev1beta1 := vzapi.ConvertValueOverridesToV1Beta1(overrides)
	overrideYaml, err := override.GetInstallOverridesYAMLUsingClient(ctx.Client(), overridev1beta1, cr.Namespace)

	if err != nil {
		return nil, fmt.Errorf("failed to convert overrides to yaml: %v", err)
	}

	if len(overrideYaml) <= 0 {
		return nodePools, nil
	}
	value, err := override.ExtractValueFromOverrideString(overrideYaml[0], nodePoolKey)
	if err != nil {
		return nil, fmt.Errorf("failed to extract value from nodepool: %v", err)
	}
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return nodePools, err
	}

	err = json.Unmarshal(jsonValue, &nodePools)
	return nodePools, err
}

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
	osdReplica := ctx.EffectiveCR().Spec.Components.Kibana.Replicas
	if osdReplica != nil {
		kvs = append(kvs, bom.KeyValue{
			Key:   "openSearchCluster.dashboards.replicas",
			Value: fmt.Sprint(*osdReplica),
		})
	}

	// Append plugins list
	// This will later go as part of CR conversion
	osPlugins := ctx.EffectiveCR().Spec.Components.Elasticsearch.Plugins
	osdPlugins := ctx.EffectiveCR().Spec.Components.Kibana.Plugins

	if osPlugins.Enabled && len(osPlugins.InstallList) > 0 {
		for i, plugin := range osPlugins.InstallList {
			kvs = append(kvs, bom.KeyValue{
				Key:   fmt.Sprintf("openSearchCluster.general.pluginsList[%d]", i),
				Value: plugin,
			})
		}
	}

	if osdPlugins.Enabled && len(osdPlugins.InstallList) > 0 {
		for i, plugin := range osdPlugins.InstallList {
			kvs = append(kvs, bom.KeyValue{
				Key:   fmt.Sprintf("openSearchCluster.dashboards.pluginsList[%d]", i),
				Value: plugin,
			})
		}
	}

	return kvs, nil
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

func buildIngressOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	if vzcr.IsNGINXEnabled(ctx.EffectiveCR()) {

		dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
		if err != nil {
			return kvs, ctx.Log().ErrorfNewErr("Failed to build DNS subdomain: %v", err)
		}
		ingressClassName := vzconfig.GetIngressClassName(ctx.EffectiveCR())
		ingressTarget := fmt.Sprintf("verrazzano-ingress.%s", dnsSubDomain)

		ingressAnnotations := make(map[string]string)
		ingressAnnotations[`kubernetes\.io/tls-acme`] = "true"
		ingressAnnotations[`nginx\.ingress\.kubernetes\.io/proxy-body-size`] = "65M"
		ingressAnnotations[`nginx\.ingress\.kubernetes\.io/rewrite-target`] = "/$2"
		ingressAnnotations[`nginx\.ingress\.kubernetes\.io/service-upstream`] = "true"
		ingressAnnotations[`nginx\.ingress\.kubernetes\.io/upstream-vhost`] = "${service_name}.${namespace}.svc.cluster.local"
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
	_ = clientgoscheme.AddToScheme(scheme)
	return scheme
}

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

func buildOSHostnameForDomain(dnsDomain string) string {
	return fmt.Sprintf("%s.%s", opensearchHostName, dnsDomain)
}

func buildOSDHostnameForDomain(dnsDomain string) string {
	return fmt.Sprintf("%s.%s", osdHostName, dnsDomain)
}

func getPrefix(ctx spi.ComponentContext) string {
	return fmt.Sprintf("Component %s", ctx.GetComponent())
}
