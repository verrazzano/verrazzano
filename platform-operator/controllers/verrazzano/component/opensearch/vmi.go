// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	system = "system"
)

// updateFunc is passed into CreateOrUpdateVMI to create the necessary VMI resources
func updateFunc(ctx spi.ComponentContext, storage *common.ResourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance, existingVMI *vmov1.VerrazzanoMonitoringInstance) error {
	hasDataNodeOverride := hasNodeStorageOverride(ctx.ActualCR(), "nodes.data.requests.storage")
	hasMasterNodeOverride := hasNodeStorageOverride(ctx.ActualCR(), "nodes.master.requests.storage")
	opensearch, err := newOpenSearch(ctx.EffectiveCR(), ctx.ActualCR(), storage, existingVMI, hasDataNodeOverride, hasMasterNodeOverride)
	if err != nil {
		return err
	}
	vmi.Spec.Elasticsearch = *opensearch
	return nil
}

func hasNodeStorageOverride(cr *vzapi.Verrazzano, override string) bool {
	openSearch := cr.Spec.Components.Elasticsearch
	if openSearch == nil {
		return false
	}
	for _, arg := range openSearch.ESInstallArgs {
		if arg.Name == override {
			return true
		}
	}

	return false
}

func actualCRNodes(cr *vzapi.Verrazzano) map[string]vzapi.OpenSearchNode {
	nodeMap := map[string]vzapi.OpenSearchNode{}
	if cr != nil && cr.Spec.Components.Elasticsearch != nil {
		for _, node := range cr.Spec.Components.Elasticsearch.Nodes {
			nodeMap[node.Name] = node
		}
	}
	return nodeMap
}

//newOpenSearch creates a new OpenSearch resource for the VMI
// The storage settings for OpenSearch nodes follow this order of precedence:
// 1. ESInstallArgs values
// 2. VolumeClaimTemplate overrides
// 3. Profile values (which show as ESInstallArgs in the ActualCR)
// The data node storage may be changed on update. The master node storage may NOT.
func newOpenSearch(effectiveCR, actualCR *vzapi.Verrazzano, storage *common.ResourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance, hasDataOverride, hasMasterOverride bool) (*vmov1.Elasticsearch, error) {
	if effectiveCR.Spec.Components.Elasticsearch == nil {
		return &vmov1.Elasticsearch{}, nil
	}
	opensearchComponent := effectiveCR.Spec.Components.Elasticsearch
	opensearch := &vmov1.Elasticsearch{
		Enabled: opensearchComponent.Enabled != nil && *opensearchComponent.Enabled,
		Storage: vmov1.Storage{},
		MasterNode: vmov1.ElasticsearchNode{
			Resources: vmov1.Resources{},
		},
		IngestNode: vmov1.ElasticsearchNode{
			Resources: vmov1.Resources{
				RequestMemory: "4.8Gi",
			},
		},
		DataNode: vmov1.ElasticsearchNode{
			Resources: vmov1.Resources{
				RequestMemory: "2.5Gi",
			},
		},
		// adapt the VPO node list to VMI node list
		Nodes: nodeAdapter(effectiveCR, vmi, effectiveCR.Spec.Components.Elasticsearch.Nodes, actualCRNodes(actualCR), storage),
	}

	// Proxy any ISM policies to the VMI
	for _, policy := range opensearchComponent.Policies {
		opensearch.Policies = append(opensearch.Policies, *policy.DeepCopy())
	}

	// Set the values in the OpenSearch object from the Verrazzano component InstallArgs
	if err := populateOpenSearchFromInstallArgs(opensearch, opensearchComponent); err != nil {
		return nil, err
	}

	setVolumeClaimOverride := func(nodeStorage *vmov1.Storage, hasInstallOverride bool) *vmov1.Storage {
		// Use the volume claim override IFF it is present AND the user did not specify a data node storage override
		if !hasInstallOverride && storage != nil && len(storage.Storage) > 0 {
			nodeStorage = &vmov1.Storage{
				Size: storage.Storage,
			}
		}
		return nodeStorage
	}
	opensearch.MasterNode.Storage = setVolumeClaimOverride(opensearch.MasterNode.Storage, hasMasterOverride)
	opensearch.DataNode.Storage = setVolumeClaimOverride(opensearch.DataNode.Storage, hasDataOverride)

	if vmi != nil {
		if vmi.Spec.Elasticsearch.MasterNode.Replicas > 0 {
			// set to old storage if present
			opensearch.MasterNode.Storage = &vmov1.Storage{
				Size: vmi.Spec.Elasticsearch.Storage.Size,
			}
			// otherwise use node storage
			if vmi.Spec.Elasticsearch.MasterNode.Storage != nil {
				opensearch.MasterNode.Storage.Size = vmi.Spec.Elasticsearch.MasterNode.Storage.Size
			}
		}

		// PVC Names should be preserved
		if opensearch.DataNode.Storage == nil {
			opensearch.DataNode.Storage = &vmov1.Storage{
				Size: vmi.Spec.Elasticsearch.Storage.Size,
			}
		}
		if vmi.Spec.Elasticsearch.Storage.PvcNames != nil {
			opensearch.DataNode.Storage.PvcNames = vmi.Spec.Elasticsearch.Storage.PvcNames
		}
	}

	return opensearch, nil
}

//populateOpenSearchFromInstallArgs loops through each of the install args and sets their value in the corresponding
// OpenSearch object
func populateOpenSearchFromInstallArgs(opensearch *vmov1.Elasticsearch, opensearchComponent *vzapi.ElasticsearchComponent) error {
	intSetter := func(val *int32, arg vzapi.InstallArgs) error {
		var intVal int32
		_, err := fmt.Sscan(arg.Value, &intVal)
		if err != nil {
			return err
		}
		*val = intVal
		return nil
	}
	// The install args were designed for helm chart, not controller code.
	// The switch statement is a shim around this design.
	for _, arg := range opensearchComponent.ESInstallArgs {
		switch arg.Name {
		case "nodes.master.replicas":
			if err := intSetter(&opensearch.MasterNode.Replicas, arg); err != nil {
				return err
			}
		case "nodes.master.requests.memory":
			opensearch.MasterNode.Resources.RequestMemory = arg.Value
		case "nodes.ingest.replicas":
			if err := intSetter(&opensearch.IngestNode.Replicas, arg); err != nil {
				return err
			}
		case "nodes.ingest.requests.memory":
			opensearch.IngestNode.Resources.RequestMemory = arg.Value
		case "nodes.data.replicas":
			if err := intSetter(&opensearch.DataNode.Replicas, arg); err != nil {
				return err
			}
		case "nodes.data.requests.memory":
			opensearch.DataNode.Resources.RequestMemory = arg.Value
		case "nodes.data.requests.storage":
			opensearch.DataNode.Storage = &vmov1.Storage{
				Size: arg.Value,
			}
		case "nodes.master.requests.storage":
			opensearch.MasterNode.Storage = &vmov1.Storage{
				Size: arg.Value,
			}
		}
	}

	return nil
}

func nodeAdapter(effectiveCR *vzapi.Verrazzano, vmi *vmov1.VerrazzanoMonitoringInstance, nodes []vzapi.OpenSearchNode, actualCRNodes map[string]vzapi.OpenSearchNode, storage *common.ResourceRequestValues) []vmov1.ElasticsearchNode {
	getQuantity := func(q *resource.Quantity) string {
		if q == nil || q.String() == "0" {
			return ""
		}
		return q.String()
	}
	var vmoNodes []vmov1.ElasticsearchNode
	for _, node := range nodes { // node is the merged profile node
		resources := vmov1.Resources{}
		if node.Resources != nil {
			resources.RequestCPU = getQuantity(node.Resources.Requests.Cpu())
			resources.LimitCPU = getQuantity(node.Resources.Limits.Cpu())
			resources.RequestMemory = getQuantity(node.Resources.Requests.Memory())
			resources.LimitMemory = getQuantity(node.Resources.Limits.Memory())
		}
		vmoNode := vmov1.ElasticsearchNode{
			Name:      node.Name,
			JavaOpts:  "",
			Replicas:  node.Replicas,
			Roles:     node.Roles,
			Resources: resources,
			Storage: &vmov1.Storage{
				Size: findStorageForNode(effectiveCR, node, actualCRNodes, storage),
			},
		}
		// if the node was present in an existing VMI and has PVC names, they should be carried over
		setPVCNames(vmi, &vmoNode)
		vmoNodes = append(vmoNodes, vmoNode)
	}
	return vmoNodes
}

func findStorageForNode(effectiveCR *vzapi.Verrazzano, node vzapi.OpenSearchNode, actualCRNodes map[string]vzapi.OpenSearchNode, storage *common.ResourceRequestValues) string {
	var storageSize string
	// Profile storage has the lowest precedence
	if node.Storage != nil {
		storageSize = node.Storage.Size
	}
	// volume claim storage has second precedence
	if storage != nil && storage.Storage != "" {
		storageSize = storage.Storage
	}
	// if using EmptyDir, zero out profile storage
	if effectiveCR.Spec.DefaultVolumeSource != nil && effectiveCR.Spec.DefaultVolumeSource.EmptyDir != nil {
		storageSize = ""
	}
	// user defined storage has the highest precedence
	if actualCRNode, ok := actualCRNodes[node.Name]; ok {
		if actualCRNode.Storage != nil {
			storageSize = actualCRNode.Storage.Size
		}
	}
	return storageSize
}

//setPVCNames persists any PVC names from an existing VMI
func setPVCNames(vmi *vmov1.VerrazzanoMonitoringInstance, node *vmov1.ElasticsearchNode) {
	if vmi != nil {
		for _, nodeGroup := range vmi.Spec.Elasticsearch.Nodes {
			if nodeGroup.Name == node.Name {
				if nodeGroup.Storage != nil {
					node.Storage.PvcNames = nodeGroup.Storage.PvcNames
				}
				return
			}
		}
	}
}
