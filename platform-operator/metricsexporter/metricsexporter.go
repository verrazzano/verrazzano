// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package metricsexporter

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	//InstallStartTimeMap is a map that will have its keys as the component name and the time since the epoch in seconds as its value
	//It will be used to store the "true" time when a component install successfully begins
	installStartTimeMap = map[string]int64{}

	authproxyInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "authproxy_component_install_time",
		Help: "The install time for the authproxy component",
	})
)

//InitalizeMetricsEndpoint creates and serves a /metrics endpoint at 9100 for Prometheus to scrape metrics from
func InitalizeMetricsEndpoint() {
	go wait.Until(func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":9100", nil)
		if err != nil {
			zap.S().Errorf("Failed to start metrics server for verrazzano-platform-operator: %v", err)
		}
	}, time.Second*3, wait.NeverStop)
}
func AddAuthproxyInstallStartTime(start_time int64) {
	installStartTimeMap["verrazzano-authproxy"] = start_time
}
func CollectAuthProxyInstallTimeMetric() {
	end_time := time.Now().Unix()
	total_install_time := end_time - installStartTimeMap["verrazzano-authproxy"]
	authproxyInstallTimeMetric.Set(float64(total_install_time))

}
