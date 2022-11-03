// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second
)

var t = framework.NewTestFramework("grafana")

var _ = t.BeforeSuite(func() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf(pkg.KubeConfigErrorFmt, err))
	}
	supported := pkg.IsGrafanaEnabled(kubeconfigPath)
	// Only run tests if Grafana component is enabled in Verrazzano CR
	if !supported {
		Skip("Grafana component is not enabled")
	}
})

var _ = t.Describe("Post Upgrade Grafana Dashboard", Label("f:observability.logging.es"), func() {

	// GIVEN a running Grafana instance,
	// WHEN a search is made for the dashboard using its title,
	// THEN the dashboard metadata is returned.
	t.It("Search the test Grafana Dashboard using its title", func() {
		pkg.TestSearchGrafanaDashboard(pollingInterval, waitTimeout)
	})

	// GIVEN a running grafana instance,
	// WHEN a GET call is made  to Grafana with the UID of the system dashboard,
	// THEN the dashboard metadata of the corresponding System dashboard is returned.
	t.It("Get details of the system Grafana dashboard", func() {
		pkg.TestSystemGrafanaDashboard(pollingInterval, waitTimeout)
	})

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Expect(err).To(BeNil(), fmt.Sprintf(pkg.KubeConfigErrorFmt, err))
	}

	// GIVEN a running grafana instance
	// WHEN a call is made to Grafana Dashboard with UID corresponding to OpenSearch Summary Dashboard
	// THEN the dashboard metadata of the corresponding dashboard is returned
	if ok, _ := pkg.IsVerrazzanoMinVersion("1.3.0", kubeconfigPath); ok {
		t.It("Get details of the OpenSearch Grafana Dashboard", func() {
			pkg.TestOpenSearchGrafanaDashBoard(pollingInterval, waitTimeout)
		})
	}
})
