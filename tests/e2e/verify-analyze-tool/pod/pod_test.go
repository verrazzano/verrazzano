// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// This is an e2e test to plant, validate and revert issues
// Here we are dealing with pod related issues
package pod

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8util "github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	utility "github.com/verrazzano/verrazzano/tests/e2e/verify-analyze-tool"
	"k8s.io/client-go/kubernetes"
	kv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"time"
)

var (
	waitTimeout     = 10 * time.Second
	pollingInterval = 10 * time.Second
)

var t = framework.NewTestFramework("Vz Analysis Tool Pod Issues")

var err error
var issuesToBeDiagnosed = []string{utility.PodProblemsNotReported, utility.PendingPods}
var client = &kubernetes.Clientset{}
var deploymentsClient kv1.DeploymentInterface

// Get the K8s Client to fetch deployment info
var _ = BeforeSuite(beforeSuite)
var beforeSuite = t.BeforeSuiteFunc(func() {
	client, err = k8util.GetKubernetesClientset()
	if err != nil {
		Fail(err.Error())
	}
	deploymentsClient = client.AppsV1().Deployments(utility.VzSystemNS)
})

// This method invoke patch method & feed vz analyze report to ReportAnalysis
// Each Iteration patch a deployment's image, validates issue via vz analyze report
// Also undo the patch and validates no issue via vz analyze report
func feedAnalysisReport() error {
	for i := 0; i < len(issuesToBeDiagnosed); i++ {
		switch issuesToBeDiagnosed[i] {
		case utility.PodProblemsNotReported:
			patchErr := utility.PatchImage(deploymentsClient, utility.DeploymentToBePatched, utility.PodProblemsNotReported, "nginx")
			if patchErr != nil {
				return patchErr
			}
		case utility.PendingPods:
			patchErr := utility.PatchPod(utility.PendingPods, []string{"cpu=50000m", "cpu=128m"})
			if patchErr != nil {
				return patchErr
			}
		}
		if i < len(issuesToBeDiagnosed)-1 {
			time.Sleep(time.Second * 20)
		}
	}
	return nil
}

var _ = t.Describe("VZ Tools", Label("f:vz-tools-pod-issues"), func() {
	t.Context("During Pod Issue Analysis", func() {
		t.It("First Inject/ Revert Issue and Feed Analysis Report", func() {
			feedAnalysisReport()
			Expect(utility.ReportAnalysis[utility.PodProblemsNotReported]).To(Not(nil))
			Expect(utility.ReportAnalysis[utility.PendingPods]).To(Not(nil))
		})

		t.It("Should Have PodProblemsNotReported Issue Post Bad Image Injection", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.PodProblemsNotReported].Patch, utility.PodProblemsNotReported)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		t.It("Should Not Have PodProblemsNotReported Issue Post Reviving Bad Image", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.PodProblemsNotReported].Revive, utility.PodProblemsNotReported)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})

		t.It("Should Have PendingPods Issue Post Bad Resource Request", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.PendingPods].Patch, utility.PendingPods)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		t.It("Should Not Have PendingPods Issue Post Rectifying Resource Request", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.PendingPods].Revive, utility.PendingPods)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
	})
})
