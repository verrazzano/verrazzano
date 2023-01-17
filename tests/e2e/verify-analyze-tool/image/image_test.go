// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// This is an e2e test to plant, validate and revert issues
// Here we are dealing with image related issues
package image

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8util "github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os/exec"
	"strings"
	"time"
)

var (
	waitTimeout     = 10 * time.Second
	pollingInterval = 10 * time.Second
)

const (
	ImagePullNotFound     string = "ISSUE (ImagePullNotFound)"
	NameSpace             string = "verrazzano-system"
	DeploymentToBePatched string = "verrazzano-console"
)

var err error
var reportAnalysis []string
var c = &kubernetes.Clientset{}
var t = framework.NewTestFramework("Vz Analysis Tool Image Issues")

// Get the K8s Client to fetch deployment info
var _ = BeforeSuite(beforeSuite)
var beforeSuite = t.BeforeSuiteFunc(func() {
	c, err = k8util.GetKubernetesClientset()
	if err != nil {
		Fail(err.Error())
	}
})

// This method invoke patch method & feed vz analyze report to reportAnalysis
// First Iteration patch a deployment's image and captures vz analyze report
// Second Iteration undo the patch and captures vz analyze report
func feedAnalysisReport() []string {
	out := make([]string, 2)
	for i := 0; i < len(out); i++ {
		patchErr := patchImage(DeploymentToBePatched, NameSpace, i == 0)
		if patchErr != nil {
			Fail(patchErr.Error())
		}
		time.Sleep(waitTimeout)
		out[i], err = RunVzAnalyze()
		if err != nil {
			Fail(err.Error())
		}
		if i == 0 {
			time.Sleep(time.Second * 30)
		}
	}
	reportAnalysis = append(reportAnalysis, out[0], out[1])
	return reportAnalysis
}

// This Method implements the patch image execution on the basis of patch flag
func patchImage(deploymentName, namespace string, patch bool) error {
	deploymentsClient := c.AppsV1().Deployments(namespace)
	result, getErr := deploymentsClient.Get(context.TODO(), deploymentName, v1.GetOptions{})
	if getErr != nil {
		return getErr
	}
	for i, container := range result.Spec.Template.Spec.Containers {
		if container.Name == deploymentName {
			image := result.Spec.Template.Spec.Containers[i].Image
			if patch {
				result.Spec.Template.Spec.Containers[i].Image = image + "X"
				break
			}
			result.Spec.Template.Spec.Containers[i].Image = image[:len(image)-1]
			break
		}
	}
	_, updateErr := deploymentsClient.Update(context.TODO(), result, v1.UpdateOptions{})
	if updateErr != nil {
		return updateErr
	}
	return nil
}

var _ = t.Describe("VZ Tools", Label("f:vz-tools-image-issues"), func() {
	t.Context("During Image Issue Analysis", func() {
		t.It("First Inject/ Revert Issue and Feed Analysis Report", func() {
			feedAnalysisReport()
			Expect(len(reportAnalysis)).To(Equal(2))
		})
		t.It("Should Have ImagePullNotFound Issue Post Bad Image Inject", func() {
			Eventually(func() bool {
				return verifyIssue(reportAnalysis[0], ImagePullNotFound)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Should Not Have ImagePullNotFound Issue Post Correct Image Inject", func() {
			Eventually(func() bool {
				return verifyIssue(reportAnalysis[1], ImagePullNotFound)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
	})
})

// utility method to run vz analyze and deliver its report
func RunVzAnalyze() (string, error) {
	cmd := exec.Command("/home/opc/go/src/github.com/verrazzano/vz", "analyze")
	out, err := cmd.Output()
	return string(out), err
}

// utility method to verify issue into vz analyze report
func verifyIssue(out, issueType string) bool {
	return strings.Contains(out, issueType)
}
