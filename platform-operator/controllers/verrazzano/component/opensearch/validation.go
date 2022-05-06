// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano-monitoring-operator/pkg/resources/nodes"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

type (

	//entryTracker is a Set like construct to track if a value was seen already
	entryTracker struct {
		set map[string]bool
	}
)

const minClusterSize = 3

func newTracker() entryTracker {
	return entryTracker{
		set: map[string]bool{},
	}
}

//add an item to the set. If it's already present, return an error.
func (e entryTracker) add(entry string) error {
	if _, exists := e.set[entry]; exists {
		return fmt.Errorf("%s already exists", entry)
	}
	e.set[entry] = true
	return nil
}

//validateNoDuplicatedConfiguration rejects any updates that contain duplicated argument names:
// Node group names or InstallArg names.
func validateNoDuplicatedConfiguration(vz *vzapi.Verrazzano) error {
	if vz.Spec.Components.Elasticsearch == nil {
		return nil
	}
	opensearch := vz.Spec.Components.Elasticsearch
	if err := validateNoDuplicateArgs(opensearch); err != nil {
		return err
	}
	return validateNoDuplicateNodeGroups(opensearch)

}

func validateNoDuplicateArgs(opensearch *vzapi.ElasticsearchComponent) error {
	tracker := newTracker()
	for _, arg := range opensearch.ESInstallArgs {
		if err := tracker.add(arg.Name); err != nil {
			return fmt.Errorf("duplicate OpenSearch install argument: %v", err)
		}
	}
	return nil
}

func validateNoDuplicateNodeGroups(opensearch *vzapi.ElasticsearchComponent) error {
	tracker := newTracker()
	for _, group := range opensearch.Nodes {
		if err := tracker.add(group.Name); err != nil {
			return fmt.Errorf("duplicate OpenSearch node group: %v", err)
		}
	}
	return nil
}

//validateClusterTopology rejects any updates that would corrupt the cluster state
func validateClusterTopology(old, new *vzapi.Verrazzano) error {
	if old.Spec.Components.Elasticsearch == nil || new.Spec.Components.Elasticsearch == nil {
		return nil
	}

	oldNodes, err := nodeCount(old)
	if err != nil {
		return err
	}
	newNodes, err := nodeCount(new)
	if err != nil {
		return err
	}

	if newNodes.Replicas > 0 {
		if !allowMasterUpdate(newNodes.MasterNodes, oldNodes.MasterNodes) {
			return nodeCountError(vmov1.MasterRole, oldNodes.MasterNodes)
		}
		if oldNodes.DataNodes > 0 && !allowNodeUpdate(newNodes.DataNodes, oldNodes.DataNodes) {
			return nodeCountError(vmov1.DataRole, oldNodes.DataNodes)
		}
	}
	return nil
}

func nodeCountError(role vmov1.NodeRole, count int32) error {
	removableNodes := (count/2)-1
	if removableNodes < 0 {
		removableNodes = 0
	}

	return fmt.Errorf("%d %s node(s) may be removed unless you are deleting the OpenSearch cluster", removableNodes, string(role))
}

func allowMasterUpdate(new, old int32) bool {
	// if we have 1-2 node cluster, we have to allow updates
	if old < minClusterSize {
		return true
	}
	// if old cluster is present but new cluster would be less than min size,
	// we cannot scale (data corruption)
	if new < minClusterSize {
		return false
	}
	return allowNodeUpdate(new, old)
}

func allowNodeUpdate(new, old int32) bool {
	return new > (old / 2)
}

func nodeCount(vz *vzapi.Verrazzano) (*nodes.NodeCount, error) {
	vmi := &vmov1.VerrazzanoMonitoringInstance{}
	vmiOpenSearch := &vmov1.Elasticsearch{
		MasterNode: vmov1.ElasticsearchNode{
			Roles: []vmov1.NodeRole{
				vmov1.MasterRole,
			},
		},
		DataNode: vmov1.ElasticsearchNode{
			Roles: []vmov1.NodeRole{
				vmov1.DataRole,
			},
		},
		IngestNode: vmov1.ElasticsearchNode{
			Roles: []vmov1.NodeRole{
				vmov1.IngestRole,
			},
		},
	}
	vpoOpenSearch := vz.Spec.Components.Elasticsearch
	if err := populateOpenSearchFromInstallArgs(vmiOpenSearch, vpoOpenSearch); err != nil {
		return nil, err
	}
	vmi.Spec.Elasticsearch = *vmiOpenSearch
	vmi.Spec.Elasticsearch.Nodes = nodeAdapter(vmi, vpoOpenSearch.Nodes, nil)
	return nodes.GetNodeCount(vmi), nil
}
