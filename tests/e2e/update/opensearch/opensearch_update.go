// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"strconv"
	"time"
)

const (
	//masterNodeName  = "system-es-master"
	//ingestNodeName  = "system-es-ingest"
	//dataNodeName    = "system-es-data"

	waitTimeout     = 20 * time.Minute
	pollingInterval = 10 * time.Second

	//updatedReplicaCount = 5
	//updatedNodeMemory   = "512Mi"
	//updatedNodeStorage  = "2Gi"
	//defaultProdMasterCount = 3
	//defaultProdIngestCount = 1
	//defaultProdDataCount   = 3
	//defaultDevMasterCount  = 1

	//AppLabel       = "app"
	NodeGroupLabel = "node-group"
)

type OpensearchMasterNodeArgsModifier struct {
	NodeReplicas uint64
	NodeMemory   string
}

type OpensearchIngestNodeArgsModifier struct {
	NodeReplicas uint64
	NodeMemory   string
}

type OpensearchDataNodeArgsModifier struct {
	NodeReplicas uint64
	NodeStorage  string
	NodeMemory   string
}

type OpensearchCleanUpArgsModifier struct {
}

func (u OpensearchMasterNodeArgsModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Elasticsearch == nil {
		cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{}
	}
	cr.Spec.Components.Elasticsearch.ESInstallArgs = []vzapi.InstallArgs{}
	cr.Spec.Components.Elasticsearch.ESInstallArgs =
		append(cr.Spec.Components.Elasticsearch.ESInstallArgs,
			vzapi.InstallArgs{Name: "nodes.master.replicas", Value: strconv.FormatUint(u.NodeReplicas, 10)},
			vzapi.InstallArgs{Name: "nodes.master.requests.memory", Value: u.NodeMemory})
}

func (u OpensearchIngestNodeArgsModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Elasticsearch == nil {
		cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{}
	}
	cr.Spec.Components.Elasticsearch.ESInstallArgs = []vzapi.InstallArgs{}
	defaultMasterNodeCount := "1"
	defaultDataNodeCount := "0"
	if cr.Spec.Profile == vzapi.Prod {
		defaultMasterNodeCount = "3"
		defaultDataNodeCount = "3"
	}
	cr.Spec.Components.Elasticsearch.ESInstallArgs =
		append(cr.Spec.Components.Elasticsearch.ESInstallArgs,
			vzapi.InstallArgs{Name: "nodes.ingest.replicas", Value: strconv.FormatUint(u.NodeReplicas, 10)},
			vzapi.InstallArgs{Name: "nodes.ingest.requests.memory", Value: u.NodeMemory},
			vzapi.InstallArgs{Name: "node.master.replicas", Value: defaultMasterNodeCount},
			vzapi.InstallArgs{Name: "node.data.replicas", Value: defaultDataNodeCount},
		)
}

func (u OpensearchDataNodeArgsModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Elasticsearch == nil {
		cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{}
	}
	cr.Spec.Components.Elasticsearch.ESInstallArgs = []vzapi.InstallArgs{}
	defaultMasterNodeCount := "1"
	defaultIngestNodeCount := "0"
	if cr.Spec.Profile == vzapi.Prod {
		defaultMasterNodeCount = "3"
		defaultIngestNodeCount = "1"
	}
	cr.Spec.Components.Elasticsearch.ESInstallArgs =
		append(cr.Spec.Components.Elasticsearch.ESInstallArgs,
			vzapi.InstallArgs{Name: "nodes.data.replicas", Value: strconv.FormatUint(u.NodeReplicas, 10)},
			vzapi.InstallArgs{Name: "nodes.data.requests.memory", Value: u.NodeMemory},
			vzapi.InstallArgs{Name: "nodes.data.requests.storage", Value: u.NodeStorage},
			vzapi.InstallArgs{Name: "nodes.master.replicas", Value: defaultMasterNodeCount},
			vzapi.InstallArgs{Name: "node.ingest.replicas", Value: defaultIngestNodeCount})
}

func (u OpensearchCleanUpArgsModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Elasticsearch == nil {
		cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{}
	}
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
	cr.Spec.Components.Elasticsearch.Nodes = []vzapi.OpenSearchNode{}
	cr.Spec.Components.Elasticsearch.Nodes =
		append(cr.Spec.Components.Elasticsearch.Nodes,
			vzapi.OpenSearchNode{
				Name:      string(u.Name),
				Replicas:  1,
				Roles:     []vmov1.NodeRole{vmov1.MasterRole, vmov1.DataRole},
				Storage:   newNodeStorage("2Gi"),
				Resources: newResources("512Mi"),
			},
			vzapi.OpenSearchNode{
				Name:      string(u.Name),
				Replicas:  1,
				Roles:     []vmov1.NodeRole{vmov1.MasterRole, vmov1.DataRole},
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
	m := OpensearchCleanUpArgsModifier{}
	_ = update.UpdateCRExpectError(m)
})
