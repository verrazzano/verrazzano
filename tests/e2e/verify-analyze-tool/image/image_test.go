// Copyright (c) 2023, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// This is an e2e test to plant image related issues and validates it
// Followed by reverting the issues to normal state and validates it
package image

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8util "github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	utility "github.com/verrazzano/verrazzano/tests/e2e/verify-analyze-tool"
	"k8s.io/client-go/kubernetes"
)

var (
	waitTimeout     = 10 * time.Second
	pollingInterval = 10 * time.Second
)

var t = framework.NewTestFramework("Vz Analysis Tool Image Issues")

var err error
var issuesToBeDiagnosed = []string{utility.ImagePullBackOff, utility.ImagePullNotFound}
var client = &kubernetes.Clientset{}

// Get the K8s Client to fetch deployment info
var _ = BeforeSuite(beforeSuite)
var beforeSuite = t.BeforeSuiteFunc(func() {
	client, err = k8util.GetKubernetesClientset()
	if err != nil {
		Fail(err.Error())
	}

	err := patch()
	if err != nil {
		Fail(fmt.Sprintf("error while patching verrazzano-related resources: %v", err.Error()))
	}
})

// patches an image for all the issues listed into 'issuesToBeDiagnosed'
func patch() error {
	for i := 0; i < len(issuesToBeDiagnosed); i++ {
		switch issuesToBeDiagnosed[i] {
		case utility.ImagePullNotFound:
			patchErr := utility.PatchImage(client, utility.DeploymentToBePatched, utility.ImagePullNotFound, "X")
			if patchErr != nil {
				return patchErr
			}
		case utility.ImagePullBackOff:
			patchErr := utility.PatchImage(client, utility.DeploymentToBePatched, utility.ImagePullBackOff, "nginxx/nginx:1.14.0")
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

var _ = t.Describe("VZ Tools", Label("f:vz-tools-image-issues"), func() {
	t.Context("During Image Issue Analysis", func() {
		t.It("Should Have ImagePullNotFound Issue Post Bad Image Injection", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.ImagePullNotFound][0], utility.ImagePullNotFound)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		t.It("Should Not Have ImagePullNotFound Issue Post Reviving Bad Image", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.ImagePullNotFound][1], utility.ImagePullNotFound)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})

		t.It("Should Have ImagePullBackOff Issue Post Bad Image Injection", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.ImagePullBackOff][0], utility.ImagePullBackOff)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		t.It("Should Not Have ImagePullBackOff Issue Post Reviving Bad Image", func() {
			Eventually(func() bool {
				return utility.VerifyIssue(utility.ReportAnalysis[utility.ImagePullBackOff][1], utility.ImagePullBackOff)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
	})
})
