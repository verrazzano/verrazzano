// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	. "github.com/onsi/ginkgo/v2"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
)

var _ = t.Describe("Update opensearch", Label("f:platform-lcm.update"), func() {

	// GIVEN a VZ custom resource in dev profile,
	// WHEN node group section for opensearch component is updated for adding master nodes
	// THEN master pods gets created.
	t.It("opensearch update master node group", func() {
		// Disable the default node
		d := OpensearchAllNodeRolesModifier{NodeReplicas: 0}
		update.UpdateCRWithRetries(d, pollingInterval, waitTimeout)
		m := OpensearchMasterNodeGroupModifier{NodeReplicas: 3, NodeMemory: "512Mi", NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		update.ValidatePods(string(vmov1.MasterRole), NodeGroupLabel, constants.VerrazzanoSystemNamespace, 3, false)
		update.ValidatePodMemoryRequest(map[string]string{NodeGroupLabel: string(vmov1.MasterRole)},
			constants.VerrazzanoSystemNamespace, "es-master", "512Mi")
		// disabling the master node-pool once testing is done so that we don't get error in VPO regarding master sts > 1.
		m = OpensearchMasterNodeGroupModifier{NodeReplicas: 0, NodeMemory: "512Mi", NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN node group section for opensearch component is updated for adding ingest nodes
	// THEN ingest pods gets created.
	t.It("opensearch update ingest node group", func() {
		m := OpensearchIngestNodeGroupModifier{NodeReplicas: 3, NodeMemory: "512Mi", NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		update.ValidatePods(string(vmov1.IngestRole), NodeGroupLabel, constants.VerrazzanoSystemNamespace, 3, false)
		update.ValidatePodMemoryRequest(map[string]string{NodeGroupLabel: string(vmov1.IngestRole)},
			constants.VerrazzanoSystemNamespace, "es-", "512Mi")
		// disabling this node-pool once testing is done so that we don't get error in VPO regarding master sts > 1.
		m = OpensearchIngestNodeGroupModifier{NodeReplicas: 0, NodeMemory: "512Mi", NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN node group section for opensearch component is updated for adding data nodes
	// THEN data pods gets created.
	t.It("opensearch update data node group", func() {
		m := OpensearchDataNodeGroupModifier{NodeReplicas: 3, NodeMemory: "512Mi", NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		update.ValidatePods(string(vmov1.DataRole), NodeGroupLabel, constants.VerrazzanoSystemNamespace, 3, false)
		update.ValidatePodMemoryRequest(map[string]string{NodeGroupLabel: string(vmov1.DataRole)},
			constants.VerrazzanoSystemNamespace, "es-", "512Mi")
		// disabling this node-pool once testing is done so that we don't get error in VPO regarding master sts > 1.
		m = OpensearchDataNodeGroupModifier{NodeReplicas: 0, NodeMemory: "512Mi", NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		// enable the default master node
		d := OpensearchAllNodeRolesModifier{NodeReplicas: 1}
		update.UpdateCRWithRetries(d, pollingInterval, waitTimeout)
	})
})
