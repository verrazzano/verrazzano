// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	. "github.com/onsi/ginkgo/v2"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const NodeGroupLabel = "node-group"

type OpensearchMasterNodeGroupModifier struct {
	NodeReplicas int32
	NodeMemory   string
	NodeStorage  string
}

type OpensearchIngestNodeGroupModifier struct {
	NodeReplicas int32
	NodeMemory   string
	NodeStorage  string
}

type OpensearchDataNodeGroupModifier struct {
	NodeReplicas int32
	NodeStorage  string
	NodeMemory   string
}

func (u OpensearchMasterNodeGroupModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Elasticsearch == nil {
		cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{}
	}
	cr.Spec.Components.Elasticsearch.Nodes = []vzapi.OpenSearchNode{}
	cr.Spec.Components.Elasticsearch.Nodes =
		append(cr.Spec.Components.Elasticsearch.Nodes,
			vzapi.OpenSearchNode{
				Name:      string(vmov1.MasterRole),
				Replicas:  u.NodeReplicas,
				Roles:     []vmov1.NodeRole{vmov1.MasterRole},
				Resources: newResources(u.NodeMemory),
				Storage:   newNodeStorage(u.NodeStorage),
			},
		)
}

func (u OpensearchIngestNodeGroupModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Elasticsearch == nil {
		cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{}
	}
	cr.Spec.Components.Elasticsearch.Nodes = []vzapi.OpenSearchNode{}
	cr.Spec.Components.Elasticsearch.Nodes =
		append(cr.Spec.Components.Elasticsearch.Nodes,
			vzapi.OpenSearchNode{
				Name:      string(vmov1.IngestRole),
				Replicas:  u.NodeReplicas,
				Roles:     []vmov1.NodeRole{vmov1.MasterRole, vmov1.IngestRole},
				Storage:   newNodeStorage(u.NodeStorage),
				Resources: newResources(u.NodeMemory),
			},
		)
}

func (u OpensearchDataNodeGroupModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Elasticsearch == nil {
		cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{}
	}
	cr.Spec.Components.Elasticsearch.Nodes = []vzapi.OpenSearchNode{}
	cr.Spec.Components.Elasticsearch.Nodes =
		append(cr.Spec.Components.Elasticsearch.Nodes,
			vzapi.OpenSearchNode{
				Name:      string(vmov1.DataRole),
				Replicas:  u.NodeReplicas,
				Roles:     []vmov1.NodeRole{vmov1.MasterRole, vmov1.DataRole},
				Storage:   newNodeStorage(u.NodeStorage),
				Resources: newResources(u.NodeMemory),
			},
		)
}

func newNodeStorage(size string) *vzapi.OpenSearchNodeStorage {
	storage := new(vzapi.OpenSearchNodeStorage)
	storage.Size = size
	return storage
}

func newResources(requestMemory string) *corev1.ResourceRequirements {
	memoryReq, err := resource.ParseQuantity(requestMemory)
	if err != nil {
		pkg.Log(pkg.Error, err.Error())
		return nil
	}
	resourceRequirements := new(corev1.ResourceRequirements)
	resourceRequirements.Requests = make(corev1.ResourceList)
	resourceRequirements.Requests[corev1.ResourceMemory] = memoryReq
	return resourceRequirements
}

var _ = t.Describe("Update opensearch", Label("f:platform-lcm.update"), func() {

	// GIVEN a VZ custom resource in dev profile,
	// WHEN node group section for opensearch component is updated for adding master nodes
	// THEN master pods gets created.
	t.It("opensearch update master node group", func() {
		m := OpensearchMasterNodeGroupModifier{NodeReplicas: 2, NodeMemory: "512Mi", NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		update.ValidatePods(string(vmov1.MasterRole), NodeGroupLabel, constants.VerrazzanoSystemNamespace, 2, false)
		update.ValidatePodMemoryRequest(map[string]string{NodeGroupLabel: string(vmov1.MasterRole)},
			constants.VerrazzanoSystemNamespace, "es-master", "512Mi")
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN node group section for opensearch component is updated for adding ingest nodes
	// THEN ingest pods gets created.
	t.It("opensearch update ingest node group", func() {
		m := OpensearchIngestNodeGroupModifier{NodeReplicas: 2, NodeMemory: "512Mi", NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		update.ValidatePods(string(vmov1.IngestRole), NodeGroupLabel, constants.VerrazzanoSystemNamespace, 2, false)
		update.ValidatePodMemoryRequest(map[string]string{NodeGroupLabel: string(vmov1.IngestRole)},
			constants.VerrazzanoSystemNamespace, "es-", "512Mi")
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN node group section for opensearch component is updated for adding data nodes
	// THEN data pods gets created.
	t.It("opensearch update data node group", func() {
		m := OpensearchDataNodeGroupModifier{NodeReplicas: 2, NodeMemory: "512Mi", NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		update.ValidatePods(string(vmov1.DataRole), NodeGroupLabel, constants.VerrazzanoSystemNamespace, 2, false)
		update.ValidatePodMemoryRequest(map[string]string{NodeGroupLabel: string(vmov1.DataRole)},
			constants.VerrazzanoSystemNamespace, "es-", "512Mi")
	})
})
