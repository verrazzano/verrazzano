package metricsexporter

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

// InitalizeMetricsEndpoint creates and serves a /metrics endpoint at 9100 for Prometheus to scrape metrics from
func InitalizeMetricsEndpoint(log *zap.SugaredLogger) {
	go registerMetricsHandlers(log)
	go wait.Until(func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":9100", nil)
		if err != nil {
			log.Errorf("Failed to start metrics server for verrazzano-platform-operator: %v", err)
		}
	}, time.Second*3, wait.NeverStop)
}

// This function increments the reconcile counter metric and calculates the duration time for a reconcile
// It assigns the duration time to a gauge vector with the key in the gauge vector being the current reconcile index
// Once the duration time is set, the reconcile index is incremented
// Could put a log but debug level
func CollectReconcileMetricsTime(startTime int64, log *zap.SugaredLogger) {
	reconcileCounterMetric.Add(float64(1))
	durationTime := (float64(time.Now().UnixMilli() - startTime)) / 1000.0
	metric, _ := reconcileLastDurationMetric.GetMetricWithLabelValues(strconv.Itoa(reconcileIndex))
	metric.Set(float64(durationTime))
	log.Debugf("Time duration metric updated with label %v", reconcileIndex)
	reconcileIndex = reconcileIndex + 1
}
func CollectReconcileMetricsError(log *zap.SugaredLogger) {
	reconcileErrorCounterMetric.Add(1)
	log.Debugf("Error counter for reconcile has been incremented by one")
}

// This function recieves the current VZCR and parses it for the latest upgrade and install time.
// It checks to see if these timestamps occur and if a "started" timestamp occurs for an operation, than a completed timestamp must also occur for that operation to be recorded
func AnalyzeVerrazzanoResourceMetrics(log vzlog.VerrazzanoLogger, cr vzapi.Verrazzano) {
	mapOfComponents := cr.Status.Components
	for componentName, componentStatusDetails := range mapOfComponents {
		var installCompletionTime string
		var upgradeCompletionTime string
		var upgradeStartTime string
		var installStartTime string
		for _, status := range componentStatusDetails.Conditions {
			if status.Type == vzapi.CondInstallStarted {
				installStartTime = status.LastTransitionTime
			}
			if status.Type == vzapi.CondInstallComplete {
				installCompletionTime = status.LastTransitionTime

			}
			if status.Type == vzapi.CondUpgradeStarted {
				upgradeStartTime = status.LastTransitionTime
			}
			if status.Type == vzapi.CondUpgradeComplete {
				upgradeCompletionTime = status.LastTransitionTime
			}
		}
		if (installStartTime == "" || installCompletionTime == "") && (upgradeStartTime == "" || upgradeCompletionTime == "") {
			return
		}
		if installStartTime != "" && installCompletionTime != "" {
			metricParserHelperFunction(log, componentName, installStartTime, installCompletionTime, "install")
		}
		if upgradeStartTime != "" && upgradeCompletionTime != "" {
			metricParserHelperFunction(log, componentName, upgradeStartTime, upgradeCompletionTime, "upgrade")
		}
	}
}
func GetErrorCounterMetric() float64 {
	return testutil.ToFloat64(reconcileErrorCounterMetric)
}
func GetReconcileCounterMetric() float64 {
	return testutil.ToFloat64(reconcileCounterMetric)
}
func GetValueOfInstallMetricForTesting(componentName string) float64 {
	return testutil.ToFloat64(metricsMap[componentName].LatestInstallDuration)
}
func GetValueOfUpgradeMetricForTesting(componentName string) float64 {
	return testutil.ToFloat64(metricsMap[componentName].LatestUpgradeDuration)
}
