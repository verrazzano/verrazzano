// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafanadb

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	threeMinutes         = 3 * time.Minute
	pollingInterval      = 10 * time.Second
	documentFile         = "testdata/upgrade/grafana/dashboard.json"
	grafanaErrMsgFmt     = "Failed to GET Grafana testDashboard: status=%d: body=%s"
	testDashboardTitle   = "E2ETestDashboard"
	systemDashboardTitle = "Host Metrics"
)

type DashboardMetadata struct {
	ID      int    `json:"id"`
	Slug    string `json:"slug"`
	Status  string `json:"status"`
	UID     string `json:"uid"`
	URL     string `json:"url"`
	Version int    `json:"version"`
}

var clientset = k8sutil.GetKubernetesClientsetOrDie()
var testDashboard DashboardMetadata
var t = framework.NewTestFramework("grafana")

var _ = t.BeforeSuite(func() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	supported := pkg.IsGrafanaEnabled(kubeconfigPath)
	// Only run tests if Grafana component is enabled in Verrazzano CR
	if !supported {
		Skip("Grafana component is not enabled")
	}
	// Create the test Grafana dashboard
	file, err := pkg.FindTestDataFile(documentFile)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("failed to find test data file: %v", err))
		Fail(err.Error())
	}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("failed to read test data file: %v", err))
		Fail(err.Error())
	}
	Eventually(func() bool {
		resp, err := pkg.CreateGrafanaDashboard(string(data))
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Failed to create Grafana testDashboard: %v", err))
			return false
		}
		if resp.StatusCode != http.StatusOK {
			pkg.Log(pkg.Error, fmt.Sprintf("Failed to create Grafana testDashboard: status=%d: body=%s", resp.StatusCode, string(resp.Body)))
			return false
		}
		json.Unmarshal(resp.Body, &testDashboard)
		return true
	}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue(),
		"It should be possible to create a Grafana dashboard and persist it.")
})

var _ = t.Describe("Test Grafana Dashboard Persistence", Label("f:observability.logging.es"), func() {

	// GIVEN a running grafana instance,
	// WHEN a GET call is made  to Grafana with the UID of the newly created testDashboard,
	// THEN the testDashboard metadata of the corresponding testDashboard is returned.
	It("Get details of the test Grafana Dashboard", func() {
		Eventually(func() bool {
			// UID of testDashboard, which is created by the previous test.
			uid := testDashboard.UID
			if uid == "" {
				return false
			}
			resp, err := pkg.GetGrafanaDashboard(uid)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			if resp.StatusCode != http.StatusOK {
				pkg.Log(pkg.Error, fmt.Sprintf(grafanaErrMsgFmt, resp.StatusCode, string(resp.Body)))
				return false
			}
			body := make(map[string]map[string]string)
			json.Unmarshal(resp.Body, &body)
			return strings.Contains(body["dashboard"]["title"], testDashboardTitle)
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue())
	})

	// GIVEN a running Grafana instance,
	// WHEN a search is done based on the dashboard title,
	// THEN the details of the dashboards matching the search query is returned.
	It("Search the test Grafana Dashboard using its title", func() {
		Eventually(func() bool {
			resp, err := pkg.SearchGrafanaDashboard(map[string]string{"query": testDashboardTitle})
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			if resp.StatusCode != http.StatusOK {
				pkg.Log(pkg.Error, fmt.Sprintf(grafanaErrMsgFmt, resp.StatusCode, string(resp.Body)))
				return false
			}
			var body []map[string]string
			json.Unmarshal(resp.Body, &body)
			for _, dashboard := range body {
				if dashboard["title"] == "E2ETestDashboard" {
					return true
				}
			}
			return false

		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue())
	})

	// GIVEN a running grafana instance,
	// WHEN a GET call is made  to Grafana with the system dashboard UID,
	// THEN the dashboard metadata of the corresponding testDashboard is returned.
	It("Get details of the system Grafana Dashboard", func() {
		// UID of system testDashboard, which is created by the VMO on startup.
		uid := "H0xWYyyik"
		Eventually(func() bool {
			resp, err := pkg.GetGrafanaDashboard(uid)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			if resp.StatusCode != http.StatusOK {
				pkg.Log(pkg.Error, fmt.Sprintf(grafanaErrMsgFmt, resp.StatusCode, string(resp.Body)))
				return false
			}
			body := make(map[string]map[string]string)
			json.Unmarshal(resp.Body, &body)
			return strings.Contains(body["dashboard"]["title"], systemDashboardTitle)
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue())
	})

	// GIVEN a running grafana instance,
	// WHEN the pod is deleted,
	// THEN pod eventually comes back up.
	It("Delete and wait for the pod to come back up", func() {
		Eventually(func() bool {
			pods, err := clientset.CoreV1().Pods(constants.VerrazzanoSystemNamespace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: "app=system-grafana",
			})
			if err != nil {
				pkg.Log(pkg.Error, "Failed to find grafana pod")
				return false
			}

			for i := range pods.Items {
				pod := &pods.Items[i]
				if err := clientset.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{}); err != nil {
					pkg.Log(pkg.Error, "Failed to delete grafana pod")
					return false
				}
			}

			pods, err = clientset.CoreV1().Pods(constants.VerrazzanoSystemNamespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				pkg.Log(pkg.Info, "Failed to get grafana pod")
				return false
			}

			for _, pod := range pods.Items {
				if !IsPodReadyOrCompleted(pod) {
					return false
				}
			}
			return true
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue())
	})

	// GIVEN a running Grafana instance,
	// WHEN a search is made for the dashboard using its title,
	// THEN the dashboard metadata is returned.
	/*
		It("Search the test Grafana Dashboard using its title", func() {
			Eventually(func() bool {
				resp, err := pkg.SearchGrafanaDashboard(map[string]string{"query": testDashboardTitle})
				if err != nil {
					pkg.Log(pkg.Error, err.Error())
					return false
				}
				if resp.StatusCode != http.StatusOK {
					pkg.Log(pkg.Error, fmt.Sprintf("Failed to GET Grafana testDashboard: status=%d: body=%s", resp.StatusCode, string(resp.Body)))
					return false
				}
				var body []map[string]string
				json.Unmarshal(resp.Body, &body)
				for _, dashboard := range body {
					if dashboard["title"] == testDashboardTitle {
						return true
					}
				}
				return false

			}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue())
		})
	*/

	// GIVEN a running grafana instance,
	// WHEN a GET call is made  to Grafana with the UID of the system dashboard,
	// THEN the dashboard metadata of the corresponding System dashboard is returned.
	It("Get details of the system Grafana dashboard", func() {
		// UID of system testDashboard, which is created by the VMO on startup.
		uid := "H0xWYyyik"
		Eventually(func() bool {
			resp, err := pkg.GetGrafanaDashboard(uid)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			if resp.StatusCode != http.StatusOK {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to GET Grafana testDashboard: status=%d: body=%s", resp.StatusCode, string(resp.Body)))
				return false
			}
			body := make(map[string]map[string]string)
			json.Unmarshal(resp.Body, &body)
			return strings.Contains(body["dashboard"]["title"], systemDashboardTitle)
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue())
	})
})

func IsPodReadyOrCompleted(pod corev1.Pod) bool {
	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		return true
	case corev1.PodRunning:
		for _, c := range pod.Status.ContainerStatuses {
			if !c.Ready {
				return false
			}
		}
		return true
	default:
		return false
	}
}
