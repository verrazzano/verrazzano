// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	. "github.com/onsi/ginkgo/v2"
)

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
				Name:      "es-master",
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
				Name:      "ingest",
				Replicas:  u.NodeReplicas,
				Roles:     []vmov1.NodeRole{vmov1.IngestRole},
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
				Name:      "es-data",
				Replicas:  u.NodeReplicas,
				Roles:     []vmov1.NodeRole{vmov1.DataRole},
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
	t.It("opensearch explicit master replicas", func() {
		m := OpensearchMasterNodeGroupModifier{NodeReplicas: 2, NodeMemory: "256Mi", NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		validatePods(masterNodeName, constants.VerrazzanoSystemNamespace, 2, false)
	})

	t.It("opensearch explicit master node memory", func() {
		m := OpensearchMasterNodeGroupModifier{NodeReplicas: 1, NodeMemory: updatedNodeMemory, NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		validatePods(masterNodeName, constants.VerrazzanoSystemNamespace, 1, false)
		validatePodMemoryRequest(dataNodeName, constants.VerrazzanoSystemNamespace, "es-master", updatedNodeMemory)
	})

	t.It("opensearch explicit ingest replicas", func() {
		m := OpensearchIngestNodeGroupModifier{NodeReplicas: 2, NodeMemory: "256Mi", NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		validatePods(ingestNodeName, constants.VerrazzanoSystemNamespace, 2, true)
	})

	t.It("opensearch explicit ingest node memory", func() {
		m := OpensearchIngestNodeGroupModifier{NodeReplicas: 1, NodeMemory: updatedNodeMemory, NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		validatePods(ingestNodeName, constants.VerrazzanoSystemNamespace, 1, true)
		validatePodMemoryRequest(dataNodeName, constants.VerrazzanoSystemNamespace, "es-ingest", updatedNodeMemory)
	})

	t.It("opensearch explicit data replicas", func() {
		m := OpensearchDataNodeArgsModifier{NodeReplicas: 1, NodeMemory: "256Mi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		validatePods(dataNodeName, constants.VerrazzanoSystemNamespace, 1, false)
	})

	t.It("opensearch explicit data node memory", func() {
		m := OpensearchDataNodeArgsModifier{NodeReplicas: 1, NodeMemory: updatedNodeMemory, NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		validatePods(dataNodeName, constants.VerrazzanoSystemNamespace, 1, false)
		validatePodMemoryRequest(dataNodeName, constants.VerrazzanoSystemNamespace, "es-data", updatedNodeMemory)
	})
})
