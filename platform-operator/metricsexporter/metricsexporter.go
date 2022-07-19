// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import "github.com/prometheus/client_golang/prometheus"

type MetricsExporter struct {
	internalConfig configuration
	internalData   data
}

type configuration struct {
	allMetrics    []prometheus.Collector       //thisMetric array will be automatically populated with all the metrics from each map. Metrics not included in a map can be added to thisMetric array for registration.
	failedMetrics map[prometheus.Collector]int //thisMetric map will be automatically populated with all metrics which were not registered correctly. Metrics in thisMetric map will be retried periodically.
	registry      prometheus.Registerer
}

type data struct {
	simpleCounterMetricMap map[metricName]*SimpleCounterMetric
	simpleGaugeMetricMap   map[metricName]*SimpleGaugeMetric
	durationMetricMap      map[metricName]*DurationMetric
	metricsComponentMap    map[metricName]*MetricsComponent
}
type SimpleCounterMetric struct {
	metric prometheus.Counter
}

func (c *SimpleCounterMetric) Inc() {
	c.metric.Inc()
}

func (c *SimpleCounterMetric) Add(num float64) {
	c.metric.Add(num)
}

type SimpleGaugeMetric struct {
	metric prometheus.Gauge
}

func (g *SimpleGaugeMetric) Set(num float64) {
	g.metric.Set(num)
}

func (g *SimpleGaugeMetric) SetToCurrentTime() {
	g.metric.SetToCurrentTime()
}

func (g *SimpleGaugeMetric) Add(num float64) {
	g.metric.Add(num)
}

type DurationMetric struct {
	metric prometheus.Summary
	timer  *prometheus.Timer
}

//Creates a new timer, and starts the timer
func (d *DurationMetric) TimerStart() {
	d.timer = prometheus.NewTimer(d.metric)
}

//stops the timer and record the duration since the last call to TimerStart
func (d *DurationMetric) TimerStop() {
	d.timer.ObserveDuration()
}

type MetricsComponent struct {
	LatestInstallDuration *SimpleGaugeMetric
	LatestUpgradeDuration *SimpleGaugeMetric
}

func getInstall(m *MetricsComponent) *SimpleGaugeMetric {
	return m.LatestInstallDuration
}
func getUpgrade(m *MetricsComponent) *SimpleGaugeMetric {
	return m.LatestUpgradeDuration
}
