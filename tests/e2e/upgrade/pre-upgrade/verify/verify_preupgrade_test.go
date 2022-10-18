// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verify

import (
	"fmt"
	"time"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	"gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

var waitTimeout = 15 * time.Minute
var pollingInterval = 30 * time.Second
var shortPollingInterval = 10 * time.Second

var t = framework.NewTestFramework("verify")

var _ = t.BeforeSuite(func() {
	start := time.Now()
	updateConfigMap()
	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("before_suite_elapsed_time", time.Since(start).Milliseconds()))
})

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || framework.VzCurrentGinkgoTestDescription().Failed()
})

var _ = t.AfterSuite(func() {
	start := time.Now()
	if failed || !beforeSuitePassed {
		pkg.ExecuteBugReport()
	}
	metrics.Emit(t.Metrics.With("after_suite_elapsed_time", time.Since(start).Milliseconds()))
})

func updateConfigMap() {
	t.Logs.Info("Update prometheus configmap")
	Eventually(func() error {
		configMap, scrapeConfigs, configYaml, err := pkg.GetPrometheusConfig()
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Failed getting prometheus config: %v", err))
			return err
		}
		testJobFound := false
		updateMap := false
		for _, nsc := range scrapeConfigs {
			scrapeConfig := nsc.(map[interface{}]interface{})
			// Change the default value of an existing default job
			if scrapeConfig[vzconst.PrometheusJobNameKey] == "prometheus" && scrapeConfig["scrape_interval"].(string) != vzconst.TestPrometheusJobScrapeInterval {
				scrapeConfig["scrape_interval"] = vzconst.TestPrometheusJobScrapeInterval
				updateMap = true
			}

			// Create test job only once
			if scrapeConfig[vzconst.PrometheusJobNameKey] == vzconst.TestPrometheusScrapeJob {
				testJobFound = true
			}
		}

		if !testJobFound {
			// Add a test scrape config
			dummyScrapConfig := make(map[interface{}]interface{})
			dummyScrapConfig[vzconst.PrometheusJobNameKey] = vzconst.TestPrometheusScrapeJob
			scrapeConfigs = append(scrapeConfigs, dummyScrapConfig)
			updateMap = true
		}

		if updateMap {
			configYaml["scrape_configs"] = scrapeConfigs
			newConfigYaml, err := yaml.Marshal(&configYaml)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed updating configmap yaml: %v", err))
				return err
			}

			configMap.Data["prometheus.yml"] = string(newConfigYaml)
			err = pkg.UpdateConfigMap(configMap)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed updating configmap: %v", err))
				return err
			}
		}

		return nil
	}, waitTimeout, shortPollingInterval).Should(BeNil())
}

var _ = t.Describe("Update prometheus configmap", Label("f:platform-lcm.upgrade", "f:observability.monitoring.prom"), func() {
	// Verify that prometheus configmap is updated
	// GIVEN the prometheus configmap is created
	// WHEN the upgrade has not started and vmo pod is not restarted
	// THEN the file updated prometheus configmap contains updated scrape interval and test job
	t.Context("check prometheus configmap", func() {
		t.It("before upgrade", func() {
			Eventually(func() bool {
				_, scrapeConfigs, _, err := pkg.GetPrometheusConfig()
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Failed getting prometheus config: %v", err))
					return false
				}

				intervalUpdated := false
				testJobFound := false
				for _, nsc := range scrapeConfigs {
					scrapeConfig := nsc.(map[interface{}]interface{})
					// Check that interval is updated
					if scrapeConfig[vzconst.PrometheusJobNameKey] == "prometheus" {
						intervalUpdated = (scrapeConfig["scrape_interval"].(string) == vzconst.TestPrometheusJobScrapeInterval)
					}

					// Check that test scrape config is created
					if scrapeConfig[vzconst.PrometheusJobNameKey] == vzconst.TestPrometheusScrapeJob {
						testJobFound = true
					}
				}
				return intervalUpdated && testJobFound
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

})
