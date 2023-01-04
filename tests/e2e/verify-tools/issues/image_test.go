// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package issues

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8util "github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os/exec"
	"strings"
	"time"
)

var (
	waitTimeout     = 10 * time.Second
	pollingInterval = 10 * time.Second
)

const (
	ImagePullNotFound string = "ImagePullNotFound"
	ImagePullBackOff  string = "ImagePullBackOff"
)

var t = framework.NewTestFramework("Vz Tools Analysis Image Issues")
var _ = BeforeSuite(beforeSuite)
var _ = t.AfterEach(func() {})
var clusterImageIssues = make(map[string]bool)
var clusterPodStatus = make(map[string][]corev1.Pod)

var beforeSuite = t.BeforeSuiteFunc(func() {
})

var _ = t.Describe("VZ Tools", Label("f:vz-tools-image-issues"), func() {
	t.Context("During Analysis", func() {
		out, err := RunVzAnalyze()
		if err != nil {
			Fail(err.Error())
		}
		fmt.Println("11111111 \n", out)
		out, err = InjectIssues()
		if err != nil {
			Fail(err.Error())
		}
		fmt.Println("22222222 \n", out)
		out, err = RunVzAnalyze()
		if err != nil {
			Fail(err.Error())
		}
		fmt.Println("33333333 \n", out)
		t.It("Doesn't Have Image Pull Back Off Issue", func() {
			Eventually(func() bool {
				return testIssues(out,"ImagePullBackOff")
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		out, err = RevertIssues()
		if err != nil {
			Fail(err.Error())
		}
		fmt.Println("44444444 \n", out)

		c, err := k8util.GetKubernetesClientset()
		if err != nil {
			Fail(err.Error())
		}
		deploymentsClient := c.ExtensionsV1beta1().Deployments("verrazzano-system")
		patch := []byte(`{"spec":{"template":{"spec":{"containers":[{"image":"ghcr.io/oracle/coherence-operator:3.YY","name":"coherence-operator"}]}}}}`)
		res, err := deploymentsClient.Patch(context.TODO(),"coherence-operator", types.MergePatchType, patch,  v1.PatchOptions{}, "")
		if err != nil {
			Fail(err.Error())
		}
		fmt.Println(res)
	})
})

func InjectIssues() (string, error) {
	cmd := exec.Command("kubectl", "apply", "-f", "issue.yaml")
	out, err := cmd.Output()
	return string(out), err
}

func RevertIssues() (string, error) {
	cmd := exec.Command("kubectl", "apply", "-f", "revertIssues.yaml")
	out, err := cmd.Output()
	return string(out), err

}
func testIssues(out, issueType string) bool {
	if strings.Contains(out, issueType) {
		return true
	}
	return false
}

func RunVzAnalyze() (string, error) {
	cmd := exec.Command("vz", "analyze")
	out, err := cmd.Output()
	return string(out), err
}

