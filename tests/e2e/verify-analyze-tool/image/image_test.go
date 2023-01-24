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
	kv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	waitTimeout     = 10 * time.Second
	pollingInterval = 10 * time.Second
)

const (
	ImagePullNotFound      string = "ImagePullNotFound"
	ImagePullBackOff       string = "ImagePullBackOff"
	PodProblemsNotReported string = "PodProblemsNotReported"
	PendingPods            string = "PendingPods"
	InsufficientMemory     string = "InsufficientMemory"
	VzSystemNS             string = "verrazzano-system"
	DeploymentToBePatched  string = "verrazzano-console"
)

type action struct {
	Patch  string
	Revive string
}

var err error
var reportAnalysis = make(map[string]action)
var issuesToBeDiagnosed = []string{ImagePullNotFound, ImagePullBackOff, PodProblemsNotReported, PendingPods, InsufficientMemory}
var c = &kubernetes.Clientset{}
var deploymentsClient kv1.DeploymentInterface

var t = framework.NewTestFramework("Vz Analysis Tool Image Issues")

// Get the K8s Client to fetch deployment info
var _ = BeforeSuite(beforeSuite)
var beforeSuite = t.BeforeSuiteFunc(func() {
	c, err = k8util.GetKubernetesClientset()
	if err != nil {
		Fail(err.Error())
	}
	deploymentsClient = c.AppsV1().Deployments(VzSystemNS)
})

// This method invoke patch method & feed vz analyze report to reportAnalysis
// Each Iteration patch a deployment's image, validates issue via vz analyze report
// Also undo the patch and validates no issue via vz analyze report
func feedAnalysisReport() error {
	for i := 0; i < len(issuesToBeDiagnosed); i++ {
		switch issuesToBeDiagnosed[i] {
		case ImagePullNotFound:
			patchErr := patchImage(DeploymentToBePatched, ImagePullNotFound, "X")
			if patchErr != nil {
				return patchErr
			}
		case ImagePullBackOff:
			patchErr := patchImage(DeploymentToBePatched, ImagePullBackOff, "nginxx/nginx:1.14.0")
			if patchErr != nil {
				return patchErr
			}
		case PodProblemsNotReported:
			patchErr := patchImage(DeploymentToBePatched, PodProblemsNotReported, "nginx")
			if patchErr != nil {
				return patchErr
			}
		case PendingPods:
			patchErr := patchPod(PendingPods, []string{"cpu=3000m", "cpu=128m"})
			if patchErr != nil {
				return patchErr
			}
		case InsufficientMemory:
			patchErr := patchPod(InsufficientMemory, []string{"memory=1000Gi", "memory=100Mi"})
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

// This Method implements the patching of bad image & its revival
func patchImage(deploymentName, issueType, patchImage string) error {
	result, getErr := deploymentsClient.Get(context.TODO(), deploymentName, v1.GetOptions{})
	if getErr != nil {
		return getErr
	}
	for i, container := range result.Spec.Template.Spec.Containers {
		if container.Name == deploymentName {
			image := result.Spec.Template.Spec.Containers[i].Image
			// PATCHING
			if issueType == ImagePullNotFound {
				patchImage = image + patchImage
			}
			result.Spec.Template.Spec.Containers[i].Image = patchImage
			_, updateErr := deploymentsClient.Update(context.TODO(), result, v1.UpdateOptions{})
			if updateErr != nil {
				return updateErr
			}
			time.Sleep(waitTimeout)
			out1, err := RunVzAnalyze()
			if err != nil {
				return err
			}
			time.Sleep(time.Second * 20)
			result, getErr = deploymentsClient.Get(context.TODO(), deploymentName, v1.GetOptions{})
			if getErr != nil {
				return getErr
			}
			// REVIVING
			result.Spec.Template.Spec.Containers[i].Image = image
			_, updateErr = deploymentsClient.Update(context.TODO(), result, v1.UpdateOptions{})
			if updateErr != nil {
				return updateErr
			}
			time.Sleep(waitTimeout)
			out2, err := RunVzAnalyze()
			if err != nil {
				return err
			}
			reportAnalysis[issueType] = action{out1, out2}
			break
		}
	}
	return nil
}

func patchPod(issueType string, resourceReq []string) error {
	out := make([]string, 2)
	for i := 0; i < len(resourceReq); i++ {
		_, err = SetDepResources(DeploymentToBePatched, VzSystemNS, resourceReq[i])
		if err != nil {
			return err
		}
		time.Sleep(waitTimeout)
		out[i], err = RunVzAnalyze()
		if err != nil {
			return err
		}
		if i == 0 {
			time.Sleep(time.Second * 20)
		}
	}
	reportAnalysis[issueType] = action{out[0], out[1]}
	return nil
}

var _ = t.Describe("VZ Tools", Label("f:vz-tools-image-issues"), func() {
	t.Context("During Image Issue Analysis", func() {
		t.It("First Inject/ Revert Issue and Feed Analysis Report", func() {
			feedAnalysisReport()
			Expect(len(reportAnalysis)).To(Equal(len(issuesToBeDiagnosed)))
		})
		t.It("Should Have ImagePullNotFound Issue Post Bad Image Injection", func() {
			Eventually(func() bool {
				return verifyIssue(reportAnalysis[ImagePullNotFound].Patch, ImagePullNotFound)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Should Not Have ImagePullNotFound Issue Post Reviving Bad Image", func() {
			Eventually(func() bool {
				return verifyIssue(reportAnalysis[ImagePullNotFound].Revive, ImagePullNotFound)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})

		t.It("Should Have ImagePullBackOff Issue Post Bad Image Injection", func() {
			Eventually(func() bool {
				return verifyIssue(reportAnalysis[ImagePullBackOff].Patch, ImagePullBackOff)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Should Not Have ImagePullBackOff Issue Post Reviving Bad Image", func() {
			Eventually(func() bool {
				return verifyIssue(reportAnalysis[ImagePullBackOff].Revive, ImagePullBackOff)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
		t.It("Should Have PodProblemsNotReported Issue Post Bad Image Injection", func() {
			Eventually(func() bool {
				return verifyIssue(reportAnalysis[PodProblemsNotReported].Patch, PodProblemsNotReported)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Should Not Have PodProblemsNotReported Issue Post Reviving Bad Image", func() {
			Eventually(func() bool {
				return verifyIssue(reportAnalysis[PodProblemsNotReported].Revive, PodProblemsNotReported)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
		t.It("Should Have PendingPods Issue Post Bad Resource Request", func() {
			Eventually(func() bool {
				return verifyIssue(reportAnalysis[PendingPods].Patch, PendingPods)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		t.It("Should Not Have PendingPods Issue Post Rectifying Resource Request", func() {
			Eventually(func() bool {
				return verifyIssue(reportAnalysis[PendingPods].Revive, PendingPods)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
		t.It("Should Have InsufficientMemory Issue Post Bad Resource Request", func() {
			Eventually(func() bool {
				return verifyIssue(reportAnalysis[InsufficientMemory].Patch, InsufficientMemory)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		t.It("Should Not Have InsufficientMemory Issue Post Rectifying Resource Request", func() {
			Eventually(func() bool {
				return verifyIssue(reportAnalysis[InsufficientMemory].Revive, InsufficientMemory)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
	})
})

// utility method to run vz analyze and deliver its report
func RunVzAnalyze() (string, error) {
	cmd := exec.Command("./vz", "analyze")
	if goRepoPath := os.Getenv("GO_REPO_PATH"); goRepoPath != "" {
		cmd.Dir = goRepoPath
	}
	out, err := cmd.Output()
	return string(out), err
}

// utility function to set deployment pod's resources (cpu/ memory)
func SetDepResources(dep, ns, req string) (string, error) {
	out, err := exec.Command("kubectl", "set", "resources", "deploy/"+dep, "--requests="+req, "-n", ns).Output()
	return string(out), err
}

// utility method to verify issue into vz analyze report
func verifyIssue(out, issueType string) bool {
	return strings.Contains(out, issueType)
}
