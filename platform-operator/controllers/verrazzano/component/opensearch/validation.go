// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano-monitoring-operator/pkg/resources/nodes"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

//entryTracker is a Set like construct to track if a value was seen already
type entryTracker struct {
		set map[string]bool
}

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

//validateNoDuplicateArgs rejects InstallArgs with duplicated names
func validateNoDuplicateArgs(opensearch *vzapi.ElasticsearchComponent) error {
	tracker := newTracker()
	for _, arg := range opensearch.ESInstallArgs {
		if err := tracker.add(arg.Name); err != nil {
			return fmt.Errorf("duplicate OpenSearch install argument: %v", err)
		}
	}
	return nil
}

//validateNoDuplicateNodeGroups rejects Nodes with duplicated group names
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
// e.g., removing half or more master nodes without deleting the cluster
func validateClusterTopology(old, new *vzapi.Verrazzano) error {
	if old.Spec.Components.Elasticsearch == nil || new.Spec.Components.Elasticsearch == nil {
		return nil
	}

	// count node replicas and roles for the old and new states
	oldNodes, err := nodeCount(old)
	if err != nil {
		return err
	}
	newNodes, err := nodeCount(new)
	if err != nil {
		return err
	}

	// if the new cluster has nodes, verify that the update would not corrupt the cluster state
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

//nodeCountError emits an error related to why the node count may not be updated
func nodeCountError(role vmov1.NodeRole, count int32) error {
	removableNodes := (count / 2) - 1
	if removableNodes < 1 {
		return fmt.Errorf("no %s nodes may be removed from OpenSearch cluster, unless you are deleting the cluster", string(role))
	}
	return fmt.Errorf("at most %d %s node(s) may be removed from the OpenSearch, unless you are deleting the cluster", removableNodes, string(role))
}

//allowMasterUpdate rejects master node updates if the update would:
// - reduce the master node count below 3 without deleting the cluster
// - reduce the master node count by half or more
// Updates are always allowed if the cluster is 1-2 nodes
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

//allowNodeUpdate rejects nodes updates if the update would:
// - reduce the node count by half or more
func allowNodeUpdate(new, old int32) bool {
	return new > (old / 2)
}

//nodeCount adapts the Verrazzano Nodes API to the VMI, and generates a NodeCount of node roles and replicas
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
