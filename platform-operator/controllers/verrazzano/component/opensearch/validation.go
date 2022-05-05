package opensearch

import (
	"fmt"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano-monitoring-operator/pkg/resources/nodes"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
)

const minClusterSize = 3

type (
	entryTracker struct {
		set map[string]bool
	}
)

func newTracker() entryTracker {
	return entryTracker{
		set: map[string]bool{},
	}
}

func (e entryTracker) add(entry string) error {
	if _, exists := e.set[entry]; exists {
		return fmt.Errorf("%s already exists", entry)
	}
	e.set[entry] = true
	return nil
}

func validateNoDuplicatedConfiguration(vz *vzapi.Verrazzano) error {
	if !vzconfig.IsElasticsearchEnabled(vz) {
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

func validateClusterTopology(old, new *vzapi.Verrazzano) error {
	oldNodes, err := nodeCount(old)
	if err != nil {
		return err
	}
	newNodes, err := nodeCount(new)
	if err != nil {
		return err
	}

	if newNodes.Replicas > 0 {
		if removesHalfOrMore(newNodes.MasterNodes, oldNodes.MasterNodes) {
			return nodeCountError(vmov1.MasterRole, oldNodes.MasterNodes)
		}
		if removesHalfOrMore(newNodes.DataNodes, oldNodes.DataNodes) {
			return nodeCountError(vmov1.DataRole, oldNodes.DataNodes)
		}
	}
	return nil
}

func nodeCountError(role vmov1.NodeRole, count int32) error {
	return fmt.Errorf("%d %s nodes may be removed unless you are deleting the OpenSearch cluster", (count/2)-1, string(role))
}

func removesHalfOrMore(n1, n2 int32) bool {
	return n2 >= (n1 / 2)
}

func nodeCount(vz *vzapi.Verrazzano) (*nodes.NodeCount, error) {
	vmi := &vmov1.VerrazzanoMonitoringInstance{}
	vmiOpenSearch := &vmov1.Elasticsearch{}
	vpoOpenSearch := vz.Spec.Components.Elasticsearch
	if err := populateOpenSearchFromInstallArgs(vmiOpenSearch, vpoOpenSearch); err != nil {
		return nil, err
	}
	vmi.Spec.Elasticsearch = *vmiOpenSearch
	vmi.Spec.Elasticsearch.Nodes = nodeAdapter(vmi, vpoOpenSearch.Nodes, nil)
	return nodes.GetNodeCount(vmi), nil
}
