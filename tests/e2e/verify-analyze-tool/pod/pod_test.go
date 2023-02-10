// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// This is an e2e test to plant pod related issues and validates it
// Followed by reverting the issues to normal state and validates it
package pod

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8util "github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	utility "github.com/verrazzano/verrazzano/tests/e2e/verify-analyze-tool"
	"k8s.io/client-go/kubernetes"
	"time"
)

var (
	waitTimeout     = 10 * time.Second
	pollingInterval = 10 * time.Second
)

var t = framework.NewTestFramework("Vz Analysis Tool Pod Issues")

var err error
var issuesToBeDiagnosed = []string{utility.PodProblemsNotReported}
var client = &kubernetes.Clientset{}

// Get the K8s Client to fetch deployment info
var _ = BeforeSuite(beforeSuite)
var beforeSuite = t.BeforeSuiteFunc(func() {
	client, err = k8util.GetKubernetesClientset()
	if err != nil {
		Fail(err.Error())
	}
})

// patches pod for all the issues listed into 'issuesToBeDiagnosed'
func patch() error {
	for i := 0; i < len(issuesToBeDiagnosed); i++ {
		switch issuesToBeDiagnosed[i] {
		case utility.PodProblemsNotReported:
			patchErr := utility.PatchImage(client, utility.DeploymentToBePatched, utility.PodProblemsNotReported, "nginx")
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
			patch()
		})

		t.It("Should Have PodProblemsNotReported Issue Post Bad Image Injection", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.PodProblemsNotReported][0], utility.PodProblemsNotReported)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		t.It("Should Not Have PodProblemsNotReported Issue Post Reviving Bad Image", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.PodProblemsNotReported][1], utility.PodProblemsNotReported)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
	})
})
