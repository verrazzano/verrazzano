// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"path"
	"strings"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	securitySecretName  = "securityconfig-secret"
	securityNamespace   = constants.VerrazzanoLoggingNamespace
	securityConfigYaml  = "opensearch-operator/opensearch-securityconfig.yaml"
	configYaml          = "config.yml"
	usersYaml           = "internal_users.yml"
	hashSecName         = "admin-credentials-secret"
	clusterName         = "opensearch"
	opensearchNodeLabel = "verrazzano.io/opensearch-nodepool"
	testBomFilePath     = "../../../../verrazzano-bom.json"
)

var (
	clusterGVR = schema.GroupVersionResource{
		Group:    "opensearch.opster.io",
		Resource: "opensearchclusters",
		Version:  "v1",
	}
)

// ApplyManifestFile applies the file present in the thirdparty/manifest directory with the given args
func ApplyManifestFile(ctx spi.ComponentContext, fileName string, args map[string]interface{}) error {
	// substitute template values to all files in the directory and apply the resulting YAML
	filePath := path.Join(config.GetThirdPartyManifestsDir(), fileName)
	yamlApplier := k8sutil.NewYAMLApplier(ctx.Client(), "")
	err := yamlApplier.ApplyFT(filePath, args)

	return err
}

// BuildArgsForOpenSearchCR gets the required values from VZ CR and converts them to yaml templates
func BuildArgsForOpenSearchCR(ctx spi.ComponentContext) (map[string]interface{}, error) {
	args := make(map[string]interface{})

	args["clusterName"] = clusterName
	args["namespace"] = constants.VerrazzanoLoggingNamespace

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

	// Set drainDataNodes only when the cluster has at least 2 data nodes
	if IsSingleDataNodeCluster(ctx) {
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

	kvs, err := GetVMOImagesOverrides()
	if err != nil {
		return args, ctx.Log().ErrorfNewErr("Failed to get Opensearch images from BOM: %v", err)
	}

	for _, kv := range kvs {
		if kv.Key == "monitoringOperator.osImage" {
			args["opensearchImage"] = kv.Value
		} else if kv.Key == "monitoringOperator.osdImage" {
			args["osdImage"] = kv.Value
		} else if kv.Key == "monitoringOperator.osInitImage" {
			args["initImage"] = kv.Value
		}
	}

	return args, nil
}

// GetVMOImagesOverrides returns the images under the verrazzano-monitoring-operator subcomponent
func GetVMOImagesOverrides() ([]bom.KeyValue, error) {
	var bomFile bom.Bom
	var err error
	if bomFile, err = bom.NewBom(config.GetDefaultBOMFilePath()); err != nil {
		return nil, err
	}

	kvs, err := bomFile.BuildImageOverrides(VMOComponentName)
	if err != nil {
		return nil, err
	}

	initImage, err := bomFile.BuildImageOverrides("monitoring-init-images")
	if err != nil {
		return nil, err
	}

	kvs = append(kvs, initImage...)
	return kvs, nil
}

// IsSingleMasterNodeCluster returns true if the cluster has a single master node
func IsSingleMasterNodeCluster(ctx spi.ComponentContext) bool {
	var replicas int32
	if vzcr.IsOpenSearchEnabled(ctx.EffectiveCR()) && ctx.EffectiveCR().Spec.Components.Elasticsearch != nil {
		for _, node := range ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes {
			for _, role := range node.Roles {
				if role == "master" && node.Replicas != nil {
					replicas += *node.Replicas
				}
			}
		}
	}
	return replicas <= 1
}

// IsSingleDataNodeCluster returns true if the cluster has a single data node
func IsSingleDataNodeCluster(ctx spi.ComponentContext) bool {
	var replicas int32
	if vzcr.IsOpenSearchEnabled(ctx.EffectiveCR()) && ctx.EffectiveCR().Spec.Components.Elasticsearch != nil {
		for _, node := range ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes {
			for _, role := range node.Roles {
				if role == "data" && node.Replicas != nil {
					replicas += *node.Replicas
				}
			}
		}
	}
	return replicas <= 1
}

// IsUpgrade returns true if we are upgrading from <=1.6.x to > 1.6
func IsUpgrade(ctx spi.ComponentContext) bool {
	if vzcr.IsOpenSearchEnabled(ctx.EffectiveCR()) && ctx.EffectiveCR().Spec.Components.Elasticsearch != nil {
		for _, node := range ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes {
			// If PVs with this label exists for any node pool, then it's an upgrade
			pvList, err := GetPVsBasedOnLabel(ctx, opensearchNodeLabel, node.Name)
			if err != nil {
				return false
			}
			if len(pvList) > 0 {
				return true
			}
		}
	}

	return false
}

// getAdditionalConfig returns any existing additional config for the given node
func getAdditionalConfig(ctx spi.ComponentContext, nodeName string) (map[string]string, bool) {
	client, err := k8sutil.GetDynamicClient()
	if err != nil {
		return nil, false
	}

	resourceClient := client.Resource(clusterGVR)
	obj, err := resourceClient.Namespace(constants.VerrazzanoLoggingNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, false
		}
		return nil, false
	}
	value, exists, err := unstructured.NestedFieldNoCopy(obj.Object, "spec", "nodePools")
	if err != nil {
		return nil, false
	}
	if exists {
		nodePools, ok := value.([]interface{})
		if ok {
			for _, node := range nodePools {
				n, ok := node.(map[string]interface{})
				if ok {
					if n["component"] == nodeName {
						result := map[string]string{}
						links, ok := n["additionalConfig"].(map[string]interface{})
						if !ok {
							return nil, false
						}
						for k := range links {
							if v, ok := links[k].(string); ok {
								result[k] = v
							}
						}
						return result, true
					}
				}
			}
		}
	}
	return nil, false
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
	resourceRequest, err := FindStorageOverride(effectiveCR)

	if err != nil {
		return nodePools, err
	}

	for _, node := range effectiveCRNodes {
		nodePool := NodePool{
			Component: node.Name,
			Jvm:       node.JavaOpts,
		}
		if node.Replicas != nil {
			if *node.Replicas == 0 {
				continue
			}
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
		if exisitngAdditionalConfig, ok := getAdditionalConfig(ctx, nodePool.Component); ok {
			nodePool.AdditionalConfig = exisitngAdditionalConfig
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
	if vzcr.IsOpenSearchEnabled(ctx.EffectiveCR()) && ctx.EffectiveCR().Spec.Components.Elasticsearch != nil {
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
	}
	return ""
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

// getClient returns a controller runtime client for the Verrazzano resource
func getClient(ctx spi.ComponentContext) (client.Client, error) {
	return ctx.Client(), nil
}

// MergeSecretData merges config.yml and internal_users.yml from the security config secret and the
// helm config present in manifests directory.
func MergeSecretData(ctx spi.ComponentContext, helmManifestsDir string) error {
	// Get the security config yaml from the manifests directory and
	//extract the config.yml and internals_users.yml data
	filePath := path.Join(helmManifestsDir, securityConfigYaml)
	securityYaml, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	securityYamlData := make(map[string]interface{})
	err = yaml.Unmarshal(securityYaml, &securityYamlData)
	if err != nil {
		return err
	}
	configYamlFile, err := getYamlData(securityYamlData, configYaml)
	if err != nil {
		return err
	}
	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	// Get the security config secret and
	// extract the config.yml and internals_users.yml data
	var scr corev1.Secret
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: securityNamespace, Name: securitySecretName}, &scr)
	if err != nil {
		if errors.IsNotFound(err) {
			// Do nothing and return if secret doesn't exist
			return nil
		}
		return err
	}
	configYamlSecret, err := getSecretYamlData(&scr, configYaml)
	if err != nil {
		return err
	}

	// Unmarshal the YAML data into maps
	dataFileConfig, err := unmarshalYAML(configYamlFile)
	if err != nil {
		return err
	}
	dataSecretConfig, err := unmarshalYAML(configYamlSecret)
	if err != nil {
		return err
	}
	mergedConfig, err := mergeConfigYamlData(dataSecretConfig, dataFileConfig)
	if err != nil {
		return err
	}
	mergedConfigYAML, err := yaml.Marshal(mergedConfig)
	if err != nil {
		return err
	}

	// Update the secret with the merged config data
	scr.Data[configYaml] = mergedConfigYAML
	usersYamlFile, err := getYamlData(securityYamlData, usersYaml)
	if err != nil {
		return err
	}
	usersYamlSecret, err := getSecretYamlData(&scr, usersYaml)
	if err != nil {
		return err
	}

	dataFileUsers, err := unmarshalYAML(usersYamlFile)
	if err != nil {
		return err
	}
	dataSecretUsers, err := unmarshalYAML(usersYamlSecret)
	if err != nil {
		return err
	}
	var adminSecret corev1.Secret
	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: securityNamespace, Name: hashSecName}, &adminSecret); err != nil {
		return err
	}
	adminHash, err := getAdminHash(&adminSecret)
	if err != nil {
		return err
	}
	// Merge the internal_users.yml data
	mergedUsers, err := mergeUserYamlData(dataFileUsers, dataSecretUsers, adminHash)
	if err != nil {
		return err
	}
	mergedUsersYAML, err := yaml.Marshal(mergedUsers)
	if err != nil {
		return err
	}
	// Assign the YAML byte slice to the secret data
	scr.Data[usersYaml] = mergedUsersYAML
	// Update the secret
	if err := client.Update(context.TODO(), &scr); err != nil {
		return err
	}
	return nil
}

// getAdminHash fetches the hash value from the admin secret
func getAdminHash(secret *corev1.Secret) (string, error) {
	hashBytes, ok := secret.Data["hash"]
	if !ok {
		// Handle the case where the 'hash' key is not present in the secret
		return "", fmt.Errorf("hash not found")
	}
	return string(hashBytes), nil
}

// getSecretYamlData gets the data corresponding to the yaml name
func getSecretYamlData(secret *corev1.Secret, yamlName string) ([]byte, error) {
	var byteYaml []byte
	for key, val := range secret.Data {
		if key == yamlName {
			byteYaml = val
		}
	}
	return byteYaml, nil
}

// unmarshalYAML unmarshals the data into the map
func unmarshalYAML(yamlData []byte) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// getYamlData gets the given data from the helm config yaml
func getYamlData(helmMap map[string]interface{}, yamlName string) ([]byte, error) {
	// Check if "stringData" exists and is of the expected type
	stringData, ok := helmMap["stringData"]
	if !ok {
		return nil, fmt.Errorf("stringData is not found in the yaml")
	}

	// Assert the type of stringData to map[interface{}]interface{}
	stringDataMap, ok := stringData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("stringData is not of type map[string]interface{}")
	}

	// Check if "yamlName" exists in stringDataMap
	dataValue, ok := stringDataMap[yamlName]
	if !ok {
		return nil, fmt.Errorf("%s not found in stringData", yamlName)
	}

	// Convert dataValue to a string
	stringValue, ok := dataValue.(string)
	if !ok {
		return nil, fmt.Errorf("%s is not a string", yamlName)
	}
	return []byte(stringValue), nil
}

// mergeConfigYamlData merges the config.yml data from the secret and the helm config
func mergeConfigYamlData(dataSecret, dataFile map[string]interface{}) (map[string]interface{}, error) {
	mergedData := make(map[string]interface{})
	authcFile, ok := dataFile["config"].(map[string]interface{})["dynamic"].(map[string]interface{})["authc"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config not found")
	}
	authcSecret := make(map[string]interface{})
	configSecret, ok := dataSecret["config"].(map[string]interface{})
	if ok {
		dynamicSecret, ok := configSecret["dynamic"].(map[string]interface{})
		if ok {
			authcSecret = dynamicSecret["authc"].(map[string]interface{})
		}
	}
	for key1, val1 := range authcFile {
		mergedData[key1] = val1
	}
	for key2, val2 := range authcSecret {
		if _, ok := mergedData[key2]; ok && strings.HasPrefix(key2, "vz") {
			continue
		}
		mergedData[key2] = val2
	}
	dataSecret["config"].(map[string]interface{})["dynamic"].(map[string]interface{})["authc"] = mergedData
	return dataSecret, nil
}

// mergeUserYamlData merges the internal_users.yml data from the secret and the helm config
func mergeUserYamlData(dataFile, dataSecret map[string]interface{}, hashFromSecret string) (map[string]interface{}, error) {
	mergedData := make(map[string]interface{})
	adminData, ok := dataFile["admin"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	adminData["hash"] = hashFromSecret
	for keyFile, valFile := range dataFile {
		if keyFile == "admin" {
			mergedData[keyFile] = adminData
		} else {
			mergedData[keyFile] = valFile
		}
	}
	for keySecret, valSecret := range dataSecret {
		if _, exists := mergedData[keySecret]; exists {
			continue
		} else {
			mergedData[keySecret] = valSecret
		}
	}
	return mergedData, nil
}
