// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var (
	MetricsExp           = metricsExporter{}
	DefaultLabelFunction func(index int64) string
	TestDelegate         = metricsDelegate{}
)

type metricsExporter struct {
	internalConfig configuration
	internalData   data
}
type configuration struct {
	// This Metric array will be automatically populated with all the metrics from each map.
	// Metrics not included in a map can be added to thisMetric array for registration.
	allMetrics []prometheus.Collector
	// This Metric map will be automatically populated with all metrics which were not registered correctly.
	// Metrics in thisMetric map will be retried periodically.
	failedMetrics map[prometheus.Collector]int
	registry      prometheus.Registerer
}
type data struct {
	simpleCounterMetricMap map[metricName]*SimpleCounterMetric
	durationMetricMap      map[metricName]*DurationMetrics
	//	webhookDuration   map[string]*DurationMetrics
	// 	webhookSuccessful map[string]*simpleCounterMetric
	// 	webhookFailed     map[string]*simpleCounterMetric
	// 	webhookDuration   map[string]*DurationMetrics
}
type metricsDelegate struct {
}

// Counter Metrics
type SimpleCounterMetric struct {
	metric prometheus.Counter
}

func (c *SimpleCounterMetric) Inc(log *zap.SugaredLogger, err error) {
	c.metric.Inc()
	if err != nil {
		log.Error(err)
	}
}
func (c *SimpleCounterMetric) Add(num float64) {
	c.metric.Add(num)
}
func (c *SimpleCounterMetric) Get() prometheus.Counter {
	return c.metric
}

// Duration Metrics
type DurationMetrics struct {
	timer  *prometheus.Timer
	metric prometheus.Summary
}

//Creates a new timer, and starts the timer
func (d *DurationMetrics) TimerStart() {
	d.timer = prometheus.NewTimer(d.metric)
}

//stops the timer and record the Duration since the last call to TimerStart
func (d *DurationMetrics) TimerStop() {
	d.timer.ObserveDuration()
}
