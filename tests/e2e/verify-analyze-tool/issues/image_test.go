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
	//ImagePullNotFound string = "ImagePullNotFound"
	ImagePullBackOff string = "ImagePullBackOff"
)

var t = framework.NewTestFramework("Vz Tools Analysis Image Issues")
var _ = BeforeSuite(beforeSuite)
var _ = t.AfterEach(func() {})

var beforeSuite = t.BeforeSuiteFunc(func() {
})

var _ = t.Describe("VZ Tools", Label("f:vz-tools-image-issues"), func() {
	t.Context("During Analysis", func() {
		c, err := k8util.GetKubernetesClientset()
		if err != nil {
			Fail(err.Error())
		}
		//patch := []byte(`{"spec":{"template":{"spec":{"containers":[{"image":"ghcr.io/oracle/coherence-operator:3.YY","name":"coherence-operator"}]}}}}`)
		deploymentsClient := c.AppsV1().Deployments("verrazzano-system")
		result, getErr := deploymentsClient.Get(context.TODO(), "verrazzano-console", v1.GetOptions{})
		if getErr != nil {
			Fail(err.Error())
		}
		//fmt.Println(result.Spec.Template.Spec.Containers[0].Image)
		for _, i := range result.Spec.Template.Spec.Containers {
			if i.Name == "verrazzano-console" {
				i.Image = "ghcr.io/verrazzano/console:v1.5.X-20221118195745-5347193"
			}
		}
		_, updateErr := deploymentsClient.Update(context.TODO(), result, v1.UpdateOptions{})
		Fail(updateErr.Error())
		out, err := RunVzAnalyze()
		if err != nil {
			Fail(err.Error())
		}
		Eventually(func() bool {
			return testIssues(out, ImagePullBackOff)
		}, waitTimeout, pollingInterval).Should(BeTrue())

		time.Sleep(time.Second * waitTimeout)

		for _, i := range result.Spec.Template.Spec.Containers {
			if i.Name == "verrazzano-console" {
				i.Image = "ghcr.io/verrazzano/console:v1.5.0-20221118195745-5347193"
			}
		}
		_, updateErr = deploymentsClient.Update(context.TODO(), result, v1.UpdateOptions{})
		Fail(updateErr.Error())
		out, err = RunVzAnalyze()
		if err != nil {
			Fail(err.Error())
		}
		Eventually(func() bool {
			return testIssues(out, ImagePullBackOff)
		}, waitTimeout, pollingInterval).Should(BeFalse())
	})
})

func RunVzAnalyze() (string, error) {
	cmd := exec.Command("vz", "analyze")
	out, err := cmd.Output()
	return string(out), err
}

func testIssues(out, issueType string) bool {
	if strings.Contains(out, issueType) {
		return true
	}
	return false
}
