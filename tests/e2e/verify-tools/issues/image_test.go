// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package issues

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
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

var beforeSuite = t.BeforeSuiteFunc(func() {
	hopClusterImageIssues()
})

var _ = t.Describe("VZ Tools", Label("f:vz-tools-image-issues"), func() {
	t.Context("During Analysis", func() {
		t.It("Doesn't Have Image Pull Not Found Issue", func() {
			Eventually(func() bool {
				return testClusterImageIssues(ImagePullNotFound)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
		t.It("Doesn't Have Image Pull Back Off Issue", func() {
			Eventually(func() bool {
				return testClusterImageIssues(ImagePullBackOff)
			}, waitTimeout, pollingInterval).Should(BeFalse())
		})
	})
})

func testClusterImageIssues(issueType string) bool {
	if _, ok := clusterImageIssues[issueType]; ok {
		return true
	}
	return false
}

func hopClusterImageIssues() {
	for _, installedNamespace := range getAllNamespaces() {
		populateClusterImageIssues(installedNamespace)
	}
}

// Return all installed cluster namespaces
func getAllNamespaces() []string {
	namespaces, err := pkg.ListNamespaces(metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	var clusterNamespaces []string
	for _, namespaceItem := range namespaces.Items {
		clusterNamespaces = append(clusterNamespaces, namespaceItem.Name)
	}
	return clusterNamespaces
}

func populateClusterImageIssues(installedNamespace string) {
	podsList, err := pkg.ListPods(installedNamespace, metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for _, pod := range podsList.Items {
		podLabels := pod.GetLabels()
		_, ok := podLabels["job-name"]
		if pod.Status.Phase != corev1.PodRunning && ok {
			continue
		}
		if len(pod.Status.InitContainerStatuses) > 0 {
			for _, initContainerStatus := range pod.Status.InitContainerStatuses {
				if initContainerStatus.State.Waiting != nil {
					if initContainerStatus.State.Waiting.Reason == ImagePullNotFound {
						clusterImageIssues[ImagePullNotFound] = true
					}
					if initContainerStatus.State.Waiting.Reason == ImagePullBackOff {
						clusterImageIssues[ImagePullBackOff] = true
					}
				}
			}
		}
		if len(pod.Status.ContainerStatuses) > 0 {
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.State.Waiting != nil {
					if containerStatus.State.Waiting.Reason == ImagePullNotFound {
						clusterImageIssues[ImagePullNotFound] = true
					}
					if containerStatus.State.Waiting.Reason == ImagePullBackOff {
						clusterImageIssues[ImagePullBackOff] = true
					}
				}
			}
		}
	}
}
