// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"strconv"
	"time"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
)

const (
	masterNodeName  = "system-es-master"
	ingestNodeName  = "system-es-ingest"
	dataNodeName    = "system-es-data"
	waitTimeout     = 20 * time.Minute
	pollingInterval = 10 * time.Second
	//updatedReplicaCount = 5
	//updatedNodeMemory   = "512Mi"
	//updatedNodeStorage  = "2Gi"
	//defaultProdMasterCount = 3
	//defaultProdIngestCount = 1
	//defaultProdDataCount   = 3
	//defaultDevMasterCount  = 1
	AppLabel = "app"
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

var t = framework.NewTestFramework("update opensearch")

var _ = t.AfterSuite(func() {
	m := OpensearchCleanUpArgsModifier{}
	update.UpdateCR(m)
})

var _ = t.Describe("Update opensearch", Label("f:platform-lcm.update"), func() {

	// GIVEN a VZ custom resource in dev or prod profile,
	// WHEN no modification is done to the CR,
	// THEN default number of master/ingest/date nodes are in running state
	t.Describe("verrazzano-opensearch verify", Label("f:platform-lcm.opensearch-verify"), func() {
		t.It("opensearch default replicas", func() {
			cr := update.GetCR()
			expectedMasterRunning := uint32(1)
			expectedIngestRunning := uint32(0)
			expectedDataRunning := uint32(0)
			if cr.Spec.Profile == "prod" || cr.Spec.Profile == "" {
				expectedMasterRunning = 3
				expectedIngestRunning = 1
				expectedDataRunning = 3
				update.ValidatePods(ingestNodeName, AppLabel, constants.VerrazzanoSystemNamespace, expectedIngestRunning, false)
				update.ValidatePods(dataNodeName, AppLabel, constants.VerrazzanoSystemNamespace, expectedDataRunning, false)
			}
			update.ValidatePods(masterNodeName, AppLabel, constants.VerrazzanoSystemNamespace, expectedMasterRunning, false)
		})
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN install args section for opensearch component is updated for adding master nodes
	// THEN master pods gets created.
	t.Describe("opensearch update master node replicas and memory", Label("f:platform-lcm.opensearch-update-replicas"), func() {
		t.It("opensearch explicit master replicas and memory", func() {
			m := OpensearchMasterNodeArgsModifier{NodeReplicas: 3, NodeMemory: "512Mi"}
			update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
			update.ValidatePods(masterNodeName, AppLabel, constants.VerrazzanoSystemNamespace, 3, false)
			update.ValidatePodMemoryRequest(map[string]string {"app" : masterNodeName}, constants.VerrazzanoSystemNamespace, "es-master", "512Mi")
		})
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN install args section for opensearch component is updated for adding ingest nodes,
	// THEN ingress pods gets created.
	t.Describe("opensearch update ingest node replicas and memory", Label("f:platform-lcm.opensearch-update-replicas"), func() {
		t.It("opensearch explicit ingest replicas and memory", func() {
			m := OpensearchIngestNodeArgsModifier{NodeReplicas: 1, NodeMemory: "512Mi"}
			update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
			update.ValidatePods(ingestNodeName, AppLabel, constants.VerrazzanoSystemNamespace, 1, false)
			update.ValidatePodMemoryRequest(map[string]string{"app": ingestNodeName}, constants.VerrazzanoSystemNamespace, "es-", "512Mi" )
		})
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN install args section for opensearch component is updated for adding data nodes,
	// THEN data pods gets created.
	t.Describe("opensearch update data node replicas", Label("f:platform-lcm.opensearch-update-replicas"), func() {
		t.It("opensearch explicit data replicas", func() {
			m := OpensearchDataNodeArgsModifier{NodeReplicas: 1, NodeMemory: "256Mi"}
			update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
			update.ValidatePods(dataNodeName, AppLabel, constants.VerrazzanoSystemNamespace, 1, false)
			update.ValidatePodMemoryRequest(map[string]string{"app": dataNodeName}, constants.VerrazzanoSystemNamespace, "es-", "256Mi")
		})
	})
})

//func validatePodStorage(deployName string, nameSpace string, expectedStorage uint32, hasPending bool) {
//	Eventually(func() bool {
//		var err error
//		pods, err := pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"app": deployName}}, nameSpace)
//		if err != nil {
//			return false
//		}
//		// Compare the number of running/pending pods to the expected numbers
//		var runningPods uint32 = 0
//		for _, pod := range pods {
//			if pod.Status.Phase == corev1.PodRunning {
//				runningPods++
//			}
//		}
//		return true
//	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get correct number of running and pending pods")
//}
