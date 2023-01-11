// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package issues

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8util "github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os/exec"
	"strings"
	"time"
)

var (
	waitTimeout     = 10 * time.Second
	pollingInterval = 10 * time.Second
)

const (
	ImagePullNotFound string = "ISSUE (ImagePullNotFound)"
	//ImagePullBackOff string = "ISSUE (ImagePullBackOff)"
)

var err error
var t = framework.NewTestFramework("Vz Tools Analysis Image Issues")
var _ = BeforeSuite(beforeSuite)
var _ = t.AfterEach(func() {})

var beforeSuite = t.BeforeSuiteFunc(func() {
})

func patchImage(patchImage, patchImageName, namespace string) error {
	c, err := k8util.GetKubernetesClientset()
	if err != nil {
		Fail(err.Error())
	}
	deploymentsClient := c.AppsV1().Deployments(namespace)
	result, getErr := deploymentsClient.Get(context.TODO(), patchImageName, v1.GetOptions{})
	if getErr != nil {
		return getErr
	}
	for ind, i := range result.Spec.Template.Spec.Containers {
		if i.Name == patchImageName {
			result.Spec.Template.Spec.Containers[ind].Image = patchImage
		}
	}
	_, updateErr := deploymentsClient.Update(context.TODO(), result, v1.UpdateOptions{})
	if updateErr != nil {
		return updateErr
	}
	return nil
}

var _ = t.Describe("VZ Tools", Label("f:vz-tools-image-issues"), func() {
	t.Context("During Analysis", func() {
		imageTobePatched := []string{"ghcr.io/verrazzano/console:v1.5.X-20221118195745-5347193", "ghcr.io/verrazzano/console:v1.5.0-20221118195745-5347193"}
		out := make([]string, len(imageTobePatched))
		for i := 0; i < len(imageTobePatched); i++ {
			patchErr := patchImage(imageTobePatched[i], "verrazzano-console", "verrazzano-system")
			if patchErr != nil {
				Fail(patchErr.Error())
			}
			time.Sleep(time.Second * waitTimeout)
			out[i], err = RunVzAnalyze()
			if err != nil {
				Fail(err.Error())
			}
			if i == 1 {
				continue
			}
			time.Sleep(time.Second * 30)
		}
		t.It("Should Have ImagePullNotFound Issue Post Bad Image Inject", func() {
			Eventually(func() bool {
				return verifyIssue(out[0], ImagePullNotFound)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Should Not Have ImagePullNotFound Issue Post Correct Image Inject", func() {
			Eventually(func() bool {
				return verifyIssue(out[1], ImagePullNotFound)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
	})
})

func RunVzAnalyze() (string, error) {
	cmd := exec.Command("vz", "analyze")
	out, err := cmd.Output()
	return string(out), err
}

func verifyIssue(out, issueType string) bool {
	if strings.Contains(out, issueType) {
		return true
	}
	return false
}
