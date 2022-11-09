// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"time"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	waitTimeout     = 20 * time.Minute
	pollingInterval = 10 * time.Second

	NodeGroupLabel = "node-group"
)

type CRModifier interface {
	ModifyCR(cr *vzapi.Verrazzano)
}

type OpensearchCleanUpModifier struct {
}

type OpensearchAllNodeRolesModifier struct {
	NodeReplicas int32
}

func (u OpensearchCleanUpModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{}
}

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

type OpensearchDuplicateNodeGroupModifier struct {
	Name string
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

func (u OpensearchDuplicateNodeGroupModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Elasticsearch == nil {
		cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{}
	}
	arg := vzapi.InstallArgs{
		Name:  "nodes.master.replicas",
		Value: "1",
	}
	cr.Spec.Components.Elasticsearch.ESInstallArgs = []vzapi.InstallArgs{
		arg,
		arg,
	}
}

func (u OpensearchAllNodeRolesModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{}
	cr.Spec.Components.Elasticsearch.Nodes = []vzapi.OpenSearchNode{}
	cr.Spec.Components.Elasticsearch.Nodes =
		append(cr.Spec.Components.Elasticsearch.Nodes,
			vzapi.OpenSearchNode{
				Name:      string(vmov1.MasterRole),
				Replicas:  u.NodeReplicas,
				Roles:     []vmov1.NodeRole{vmov1.MasterRole, vmov1.DataRole, vmov1.IngestRole},
				Storage:   newNodeStorage("2Gi"),
				Resources: newResources("512Mi"),
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

// Initialize the Test Framework
var t = framework.NewTestFramework("update opensearch")

var _ = t.AfterSuite(func() {
	m := OpensearchAllNodeRolesModifier{}
	update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
})
