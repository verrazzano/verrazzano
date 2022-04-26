// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"strconv"
	"time"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/verrazzano/verrazzano/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
)

const (
	masterNodeName      = "vmi-system-es-master"
	ingestNodeName      = "vmi-system-es-ingest"
	dataNodeName        = "vmi-system-es-data"
	waitTimeout         = 5 * time.Minute
	pollingInterval     = 5 * time.Second
	updatedReplicaCount = 5
	updatedNodeMemory   = "512Mi"
	updatedNodeStorage  = "2Gi"

)

var(

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
	if cr.Spec.Components.Elasticsearch.ESInstallArgs == nil {
		cr.Spec.Components.Elasticsearch.ESInstallArgs = make([]vzapi.InstallArgs, 2)
	}
	cr.Spec.Components.Elasticsearch.ESInstallArgs =
		append(cr.Spec.Components.Elasticsearch.ESInstallArgs,
			vzapi.InstallArgs{Name: "nodes.master.replicas", Value: strconv.FormatUint(u.NodeReplicas, 10)},
			vzapi.InstallArgs{Name: "nodes.master.requests.memory", Value: u.NodeMemory})
}

func (u OpensearchIngestNodeArgsModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Elasticsearch == nil {
		cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{}
	}
	if cr.Spec.Components.Elasticsearch.ESInstallArgs == nil {
		cr.Spec.Components.Elasticsearch.ESInstallArgs = make([]vzapi.InstallArgs, 2)
	}
	cr.Spec.Components.Elasticsearch.ESInstallArgs =
		append(cr.Spec.Components.Elasticsearch.ESInstallArgs,
			vzapi.InstallArgs{Name: "nodes.ingest.replicas", Value: strconv.FormatUint(u.NodeReplicas, 10)},
			vzapi.InstallArgs{Name: "nodes.ingest.requests.memory", Value: u.NodeMemory})
}

func (u OpensearchDataNodeArgsModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.Elasticsearch == nil {
		cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{}
	}
	if cr.Spec.Components.Elasticsearch.ESInstallArgs == nil {
		cr.Spec.Components.Elasticsearch.ESInstallArgs = make([]vzapi.InstallArgs, 2)
	}
	cr.Spec.Components.Elasticsearch.ESInstallArgs =
		append(cr.Spec.Components.Elasticsearch.ESInstallArgs,
			vzapi.InstallArgs{Name: "nodes.data.replicas", Value: strconv.FormatUint(u.NodeReplicas, 10)},
			vzapi.InstallArgs{Name: "nodes.data.requests.memory", Value: u.NodeMemory},
			vzapi.InstallArgs{Name: "nodes.data.requests.storage", Value: u.NodeStorage})
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
	cr := update.GetCR()

	expectedRunning := uint32(1)
	if cr.Spec.Profile == "prod" || cr.Spec.Profile == "" {
		expectedRunning = 3
	}
	validatePods(masterNodeName, constants.VerrazzanoSystemNamespace, expectedRunning, false)
	validatePods(ingestNodeName, constants.VerrazzanoSystemNamespace, expectedRunning, false)
	validatePods(dataNodeName, constants.VerrazzanoSystemNamespace, expectedRunning, false)
})

var _ = t.Describe("Update opensearch", Label("f:platform-lcm.update"), func() {
	t.Describe("verrazzano-opensearch verify", Label("f:platform-lcm.opensearch-verify"), func() {
		t.It("opensearch default replicas", func() {
			cr := update.GetCR()
			expectedMasterRunning := uint32(1)
			validatePods(masterNodeName, constants.VerrazzanoSystemNamespace, expectedMasterRunning, false)
			expectedIngestRunning := uint32(0)
			expectedDataRunning := uint32(0)
			if cr.Spec.Profile == "prod" || cr.Spec.Profile == "" {
				expectedMasterRunning = 3
				expectedIngestRunning = 3
				expectedDataRunning = 3
				validatePods(masterNodeName, constants.VerrazzanoSystemNamespace, expectedMasterRunning, false)
				validatePods(ingestNodeName, constants.VerrazzanoSystemNamespace, expectedIngestRunning, false)
				validatePods(dataNodeName, constants.VerrazzanoSystemNamespace, expectedDataRunning, false)
			}

		})
	})

	t.Describe("opensearch update master node replicas", Label("f:platform-lcm.opensearch-update-replicas"), func() {
		t.It("opensearch explicit master replicas", func() {
			m := OpensearchMasterNodeArgsModifier{NodeReplicas: updatedReplicaCount}
			update.UpdateCR(m)
			validatePods(masterNodeName, constants.VerrazzanoSystemNamespace, updatedReplicaCount, false)
		})
	})

	t.Describe("opensearch update master node memory", Label("f:platform-lcm.opensearch-update-memory"), func() {
		t.It("opensearch explicit master node memory", func() {
			m := OpensearchMasterNodeArgsModifier{NodeMemory: updatedNodeMemory}
			update.UpdateCR(m)
			validatePods(masterNodeName, constants.VerrazzanoSystemNamespace, updatedReplicaCount, false)
		})
	})

	t.Describe("opensearch update ingest node replicas", Label("f:platform-lcm.opensearch-update-replicas"), func() {
		t.It("opensearch explicit ingest replicas", func() {
			m := OpensearchIngestNodeArgsModifier{NodeReplicas: updatedReplicaCount}
			update.UpdateCR(m)
			validatePods(ingestNodeName, constants.VerrazzanoSystemNamespace, updatedReplicaCount, false)
		})
	})

	t.Describe("opensearch update ingest node memory", Label("f:platform-lcm.opensearch-update-memory"), func() {
		t.It("opensearch explicit ingest node memory", func() {
			m := OpensearchIngestNodeArgsModifier{NodeMemory: updatedNodeMemory}
			update.UpdateCR(m)
			validatePods(masterNodeName, constants.VerrazzanoSystemNamespace, updatedReplicaCount, false)
		})
	})

	t.Describe("opensearch update data node replicas", Label("f:platform-lcm.opensearch-update-replicas"), func() {
		t.It("opensearch explicit data replicas", func() {
			m := OpensearchMasterNodeArgsModifier{NodeReplicas: updatedReplicaCount}
			update.UpdateCR(m)
			validatePods(masterNodeName, constants.VerrazzanoSystemNamespace, updatedReplicaCount, false)
		})
	})

	t.Describe("opensearch update data node memory", Label("f:platform-lcm.opensearch-update-memory"), func() {
		t.It("opensearch explicit data node memory", func() {
			m := OpensearchDataNodeArgsModifier{NodeMemory: updatedNodeMemory}
			update.UpdateCR(m)
			validatePods(masterNodeName, constants.VerrazzanoSystemNamespace, updatedReplicaCount, false)
		})
	})

})

func validatePods(deployName string, nameSpace string, expectedPodsRunning uint32, hasPending bool) {
	Eventually(func() bool {
		var err error
		pods, err := pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"app": deployName}}, nameSpace)
		if err != nil {
			return false
		}
		// Compare the number of running/pending pods to the expected numbers
		var runningPods uint32 = 0
		var pendingPods = false
		for _, pod := range pods {
			if pod.Status.Phase == corev1.PodRunning {
				runningPods++
			}
			if pod.Status.Phase == corev1.PodPending {
				pendingPods = true
			}
		}
		return runningPods == expectedPodsRunning && pendingPods == hasPending
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get correct number of running and pending pods")
}

//func validatePodMemoryRequest(deployName string, nameSpace string, containerName string, expectedMemoryRequest string, hasPending bool) {
//	Eventually(func() bool {
//		var err error
//		pods, err := pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"app": deployName}}, nameSpace)
//		if err != nil {
//			return false
//		}
//		// Compare the number of running/pending pods to the expected numbers
//		var runningPods uint32 = 0
//		var pendingPods = false
//		for _, pod := range pods {
//			if pod.Status.Phase == corev1.PodRunning {
//				runningPods++
//			}
//			if pod.Status.Phase == corev1.PodPending {
//				pendingPods = true
//			}
//			for _, container := range pod.Spec.Containers {
//				if container.Name != containerName {
//					continue
//				}
//				expectedNodeMemory, err := resource.ParseQuantity(updatedNodeMemory)
//				if err != nil {
//					pkg.Log(pkg.Error, err.Error())
//					return false
//				}
//				if container.Resources.Requests.Memory() != &expectedNodeMemory {
//					return false
//				}
//			}
//		}
//		return  pendingPods == hasPending
//	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get correct number of running and pending pods")
//}

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
