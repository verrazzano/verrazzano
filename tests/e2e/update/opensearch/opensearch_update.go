// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"

	v1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
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

type OpensearchCleanUpModifier struct {
}

type OpensearchAllNodeRolesModifier struct {
}

func (u OpensearchCleanUpModifier) ModifyCRV1beta1(cr *vzapi.Verrazzano) {
	cr.Spec.Components.OpenSearch = &vzapi.OpenSearchComponent{}
}

type OpensearchMasterNodeGroupModifier struct {
	NodeReplicas int32
	NodeMemory   string
	NodeStorage  string
}

type OpenSearchPlugins struct {
	Enabled      bool
	InstanceList string
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

func (u OpensearchMasterNodeGroupModifier) ModifyCRV1beta1(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.OpenSearch == nil {
		cr.Spec.Components.OpenSearch = &vzapi.OpenSearchComponent{}
	}
	cr.Spec.Components.OpenSearch.Nodes = []vzapi.OpenSearchNode{}
	cr.Spec.Components.OpenSearch.Nodes =
		append(cr.Spec.Components.OpenSearch.Nodes,
			vzapi.OpenSearchNode{
				Name:      string(vmov1.MasterRole),
				Replicas:  &u.NodeReplicas,
				Roles:     []vmov1.NodeRole{vmov1.MasterRole},
				Resources: newResources(u.NodeMemory),
				Storage:   newNodeStorage(u.NodeStorage),
			},
		)
}

func (u OpenSearchPlugins) ModifyCRV1beta1(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.OpenSearch == nil {
		cr.Spec.Components.OpenSearch = &vzapi.OpenSearchComponent{}
	}
	cr.Spec.Components.OpenSearch.Plugins = vmov1.OpenSearchPlugins{}
	cr.Spec.Components.OpenSearch.Plugins =
		vmov1.OpenSearchPlugins{
			Enabled:     u.Enabled,
			InstallList: []string{u.InstanceList},
		}
}

func (u OpensearchIngestNodeGroupModifier) ModifyCRV1beta1(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.OpenSearch == nil {
		cr.Spec.Components.OpenSearch = &vzapi.OpenSearchComponent{}
	}
	cr.Spec.Components.OpenSearch.Nodes = []vzapi.OpenSearchNode{}
	cr.Spec.Components.OpenSearch.Nodes =
		append(cr.Spec.Components.OpenSearch.Nodes,
			vzapi.OpenSearchNode{
				Name:      string(vmov1.IngestRole),
				Replicas:  &u.NodeReplicas,
				Roles:     []vmov1.NodeRole{vmov1.MasterRole, vmov1.IngestRole},
				Storage:   newNodeStorage(u.NodeStorage),
				Resources: newResources(u.NodeMemory),
			},
		)
}

func (u OpensearchDataNodeGroupModifier) ModifyCRV1beta1(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.OpenSearch == nil {
		cr.Spec.Components.OpenSearch = &vzapi.OpenSearchComponent{}
	}
	cr.Spec.Components.OpenSearch.Nodes = []vzapi.OpenSearchNode{}
	cr.Spec.Components.OpenSearch.Nodes =
		append(cr.Spec.Components.OpenSearch.Nodes,
			vzapi.OpenSearchNode{
				Name:      string(vmov1.DataRole),
				Replicas:  &u.NodeReplicas,
				Roles:     []vmov1.NodeRole{vmov1.MasterRole, vmov1.DataRole},
				Storage:   newNodeStorage(u.NodeStorage),
				Resources: newResources(u.NodeMemory),
			},
		)
}

func (u OpensearchDuplicateNodeGroupModifier) ModifyCRV1beta1(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.OpenSearch == nil {
		cr.Spec.Components.OpenSearch = &vzapi.OpenSearchComponent{}
	}
	// FIXME: skeptical I got this right
	var replicas int32 = 1
	arg := vzapi.OpenSearchNode{
		Name:  "nodes.master.replicas",
		Replicas: &replicas,
		Roles: []v1.NodeRole{v1.MasterRole},
	}
	cr.Spec.Components.OpenSearch.Nodes = []vzapi.OpenSearchNode{
		arg,
		arg,
	}
}

func (u OpensearchAllNodeRolesModifier) ModifyCRV1beta1(cr *vzapi.Verrazzano) {
	cr.Spec.Components.OpenSearch= &vzapi.OpenSearchComponent{}
	cr.Spec.Components.OpenSearch.Nodes = []vzapi.OpenSearchNode{}
	cr.Spec.Components.OpenSearch.Nodes =
		append(cr.Spec.Components.OpenSearch.Nodes,
			vzapi.OpenSearchNode{
				Name:      string(vmov1.MasterRole),
				Replicas:  common.Int32Ptr(3),
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

var afterSuite = t.AfterSuiteFunc(func() {
	m := OpensearchAllNodeRolesModifier{}
	update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
})

var _ = ginkgo.AfterSuite(afterSuite)
