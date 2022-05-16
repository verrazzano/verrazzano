// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = t.Describe("Update opensearch", Label("f:platform-lcm.update"), func() {

	// GIVEN a VZ custom resource in dev or prod profile,
	// WHEN no modification is done to the CR,
	// THEN default number of master/ingest/date nodes are in running state
	//t.Describe("verrazzano-opensearch verify", Label("f:platform-lcm.opensearch-verify"), func() {
	//	t.It("opensearch default replicas", func() {
	//		cr := update.GetCR()
	//		expectedMasterRunning := uint32(1)
	//		expectedIngestRunning := uint32(0)
	//		expectedDataRunning := uint32(0)
	//		if cr.Spec.Profile == "prod" || cr.Spec.Profile == "" {
	//			expectedMasterRunning = 3
	//			expectedIngestRunning = 1
	//			expectedDataRunning = 3
	//			update.ValidatePods(ingestNodeName, AppLabel, constants.VerrazzanoSystemNamespace, expectedIngestRunning, false)
	//			update.ValidatePods(dataNodeName, AppLabel, constants.VerrazzanoSystemNamespace, expectedDataRunning, false)
	//		}
	//		update.ValidatePods(masterNodeName, AppLabel, constants.VerrazzanoSystemNamespace, expectedMasterRunning, false)
	//	})
	//})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN install args section for opensearch component is updated for adding master nodes
	// THEN master pods gets created.
	//t.Describe("opensearch update master node replicas and memory", Label("f:platform-lcm.opensearch-update-replicas"), func() {
	//	t.It("opensearch explicit master replicas and memory", func() {
	//		m := OpensearchMasterNodeArgsModifier{NodeReplicas: 3, NodeMemory: "512Mi"}
	//		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
	//		update.ValidatePods(masterNodeName, AppLabel, constants.VerrazzanoSystemNamespace, 3, false)
	//		update.ValidatePodMemoryRequest(map[string]string {"app" : masterNodeName}, constants.VerrazzanoSystemNamespace, "es-master", "512Mi")
	//	})
	//})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN install args section for opensearch component is updated for adding ingest nodes,
	// THEN ingress pods gets created.
	//t.Describe("opensearch update ingest node replicas and memory", Label("f:platform-lcm.opensearch-update-replicas"), func() {
	//	t.It("opensearch explicit ingest replicas and memory", func() {
	//		m := OpensearchIngestNodeArgsModifier{NodeReplicas: 1, NodeMemory: "512Mi"}
	//		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
	//		update.ValidatePods(ingestNodeName, AppLabel, constants.VerrazzanoSystemNamespace, 1, false)
	//		update.ValidatePodMemoryRequest(map[string]string{"app": ingestNodeName}, constants.VerrazzanoSystemNamespace, "es-", "512Mi" )
	//	})
	//})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN install args section for opensearch component is updated for adding data nodes,
	// THEN data pods gets created.
	//t.Describe("opensearch update data node replicas", Label("f:platform-lcm.opensearch-update-replicas"), func() {
	//	t.It("opensearch explicit data replicas", func() {
	//		m := OpensearchDataNodeArgsModifier{NodeReplicas: 1, NodeMemory: "256Mi"}
	//		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
	//		update.ValidatePods(dataNodeName, AppLabel, constants.VerrazzanoSystemNamespace, 1, false)
	//		update.ValidatePodMemoryRequest(map[string]string{"app": dataNodeName}, constants.VerrazzanoSystemNamespace, "es-", "256Mi")
	//	})
	//})
})
