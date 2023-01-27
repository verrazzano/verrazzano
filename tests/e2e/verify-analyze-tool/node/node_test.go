// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// This is an e2e test to plant, validate and revert issues
// Here we are dealing with node related issues
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

var issuesToBeDiagnosed = []string{utility.InsufficientMemory}

var _ = BeforeSuite(beforeSuite)
var beforeSuite = t.BeforeSuiteFunc(func() {
})

// This method invoke patch method & feed vz analyze report to ReportAnalysis
// Each Iteration patch a deployment's image, validates issue via vz analyze report
// Also undo the patch and validates no issue via vz analyze report
func feedAnalysisReport() error {
	for i := 0; i < len(issuesToBeDiagnosed); i++ {
		switch issuesToBeDiagnosed[i] {
		case utility.InsufficientMemory:
			patchErr := utility.PatchPod(utility.InsufficientMemory, []string{"memory=1000Gi", "memory=100Mi"})
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
			feedAnalysisReport()
			Expect(utility.ReportAnalysis[utility.InsufficientMemory]).To(Not(nil))
		})
		t.It("Should Have InsufficientMemory Issue Post Bad Resource Request", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.InsufficientMemory].Patch, utility.InsufficientMemory)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		t.It("Should Not Have InsufficientMemory Issue Post Rectifying Resource Request", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.InsufficientMemory].Revive, utility.InsufficientMemory)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
	})
})
