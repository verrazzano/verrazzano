// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
)

const (
	nodePoolLabel     = "opster.io/opensearch-nodepool"
	osMasterNodegroup = "es-master"
)

var _ = t.Describe("Update opensearch", Label("f:platform-lcm.update"), func() {

	// GIVEN a VZ custom resource in dev profile,
	// WHEN node group section for opensearch component is updated for adding master nodes
	// THEN master pods gets created.
	t.It("opensearch update master node group", func() {
		m := OpensearchMasterNodeGroupModifier{NodeReplicas: 3, NodeMemory: "512Mi", NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		update.ValidatePods(string(vmov1.MasterRole), nodePoolLabel, constants.VerrazzanoLoggingNamespace, 3, false)
		update.ValidatePodMemoryRequest(map[string]string{nodePoolLabel: string(vmov1.MasterRole)},
			constants.VerrazzanoLoggingNamespace, "opensearch", "512Mi")
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN node group section for opensearch component is updated for adding ingest nodes
	// THEN ingest pods gets created.
	t.It("opensearch update ingest node group", func() {
		m := OpensearchIngestNodeGroupModifier{NodeReplicas: 3, NodeMemory: "512Mi", NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		update.ValidatePods(string(vmov1.IngestRole), nodePoolLabel, constants.VerrazzanoLoggingNamespace, 3, false)
		update.ValidatePodMemoryRequest(map[string]string{nodePoolLabel: string(vmov1.IngestRole)},
			constants.VerrazzanoLoggingNamespace, "opensearch", "512Mi")
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN node group section for opensearch component is updated for adding data nodes
	// THEN data pods gets created.
	t.It("opensearch update data node group", func() {
		m := OpensearchDataNodeGroupModifier{NodeReplicas: 3, NodeMemory: "512Mi", NodeStorage: "2Gi"}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		update.ValidatePods(string(vmov1.DataRole), nodePoolLabel, constants.VerrazzanoLoggingNamespace, 3, false)
		update.ValidatePodMemoryRequest(map[string]string{nodePoolLabel: string(vmov1.DataRole)},
			constants.VerrazzanoLoggingNamespace, "opensearch", "512Mi")
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN opensearch plugin is updated with wrong value
	// THEN master pods gets in failure state.
	// In last opensearch plugin is disabled
	// Then all pods are back to normal state
	t.It("opensearch update plugin", func() {
		m := OpenSearchPlugins{Enabled: true, InstanceList: "abc"}
		update.UpdateCRWithPlugins(m, pollingInterval, waitTimeout)
		update.ValidatePods(osMasterNodegroup, nodePoolLabel, constants.VerrazzanoLoggingNamespace, 0, false)
		m = OpenSearchPlugins{Enabled: true, InstanceList: "analysis-stempel"}
		update.UpdateCRWithPlugins(m, pollingInterval, waitTimeout)
		var pods []corev1.Pod
		var err error
		Eventually(func() error {
			pods, err = pkg.GetPodsFromSelector(&v1.LabelSelector{MatchLabels: map[string]string{nodePoolLabel: osMasterNodegroup}}, constants.VerrazzanoLoggingNamespace)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return err
			}
			return nil
		}).WithPolling(20*time.Second).WithTimeout(2*time.Minute).Should(BeNil(), "failed to fetch the opensearch master pods")
		update.ValidatePods(osMasterNodegroup, nodePoolLabel, constants.VerrazzanoLoggingNamespace, uint32(len(pods)), false)
	})

})
