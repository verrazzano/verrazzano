// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// This is an e2e test to plant node related issues and validates it
// Followed by reverting the issues to normal state and validates it
package node

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	utility "github.com/verrazzano/verrazzano/tests/e2e/verify-analyze-tool"
	"time"
)

var (
	waitTimeout     = 10 * time.Second
	pollingInterval = 10 * time.Second
)

var t = framework.NewTestFramework("Vz Analysis Tool Node Issues")

var issuesToBeDiagnosed = []string{utility.InsufficientMemory, utility.InsufficientCPU}

// patches node for all the issues listed into 'issuesToBeDiagnosed'
func patch() error {
	for i := 0; i < len(issuesToBeDiagnosed); i++ {
		switch issuesToBeDiagnosed[i] {
		case utility.InsufficientMemory:
			patchErr := utility.PatchPod(utility.InsufficientMemory, []string{"memory=1000Gi", "memory=100Mi"})
			if patchErr != nil {
				return patchErr
			}
		case utility.InsufficientCPU:
			patchErr := utility.PatchPod(utility.InsufficientCPU, []string{"cpu=50000m", "cpu=128m"})
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

var _ = t.Describe("VZ Tools", Label("f:vz-tools-node-issues"), func() {
	t.Context("During Node Issue Analysis", func() {
		t.It("First Inject/ Revert Issue and Feed Analysis Report", func() {
			patch()
		})
		t.It("Should Have InsufficientMemory Issue Post Bad Memory Request", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.InsufficientMemory][0], utility.InsufficientMemory)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		t.It("Should Not Have InsufficientMemory Issue Post Rectifying Memory Request", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.InsufficientMemory][1], utility.InsufficientMemory)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
		t.It("Should Have InsufficientCPU Issue Post Bad CPU Request", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.InsufficientCPU][0], utility.InsufficientCPU)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		t.It("Should Not Have InsufficientCPU Issue Post Rectifying CPU Request", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.InsufficientCPU][1], utility.InsufficientCPU)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
	})
})
