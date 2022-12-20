// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"github.com/prometheus/client_golang/prometheus"
)

type MetricsExporter struct {
	internalConfig configuration
	internalData   data
}

// The alMetrics array will be automatically populated with all the metrics from each map. Metrics not included in a map can be added to thisMetric array for registration.
// The failedMetrics map will be automatically populated with all metrics which were not registered correctly. Metrics in thisMetric map will be retried periodically.
type configuration struct {
	allMetrics    []prometheus.Collector
	failedMetrics map[prometheus.Collector]int
	registry      prometheus.Registerer
}

type data struct {
	simpleCounterMetricMap   map[metricName]*SimpleCounterMetric
	simpleGaugeMetricMap     map[metricName]*SimpleGaugeMetric
	durationMetricMap        map[metricName]*DurationMetric
	metricsComponentMap      map[metricName]*MetricsComponent
	componentHealth          *ComponentHealth
	componentInstallDuration *ComponentInstallDuration
	componentUpgradeDuration *ComponentUpgradeDuration
}
type SimpleCounterMetric struct {
	metric prometheus.Counter
}

// This member function increases a simpleCounterMetric by one
func (c *SimpleCounterMetric) Inc() {
	c.metric.Inc()
}

// This member function increases a simpleCounterMetric by a user provided float64 number
func (c *SimpleCounterMetric) Add(num float64) {
	c.metric.Add(num)
}

// This member function returns the underlying metric in a simpleCounterMetric
func (c *SimpleCounterMetric) Get() prometheus.Counter {
	return c.metric
}

type SimpleGaugeMetric struct {
	metric prometheus.Gauge
}

// This member function sets a SimpleGaugeMetric to a user provided float64 number
func (g *SimpleGaugeMetric) Set(num float64) {
	g.metric.Set(num)
}

// This member function sets a SimpleGaugeMetric to the current time
func (g *SimpleGaugeMetric) SetToCurrentTime() {
	g.metric.SetToCurrentTime()
}

// This member function increases a SimpleGaugeMetric by a user provided float64 number
func (g *SimpleGaugeMetric) Add(num float64) {
	g.metric.Add(num)
}

// This member function returns the underlying metric in a simpleGaugeMetric
func (g *SimpleGaugeMetric) Get() prometheus.Gauge {
	return g.metric
}

type DurationMetric struct {
	metric prometheus.Summary
	timer  *prometheus.Timer
}

// This function creates a new timer, and starts the timer
func (d *DurationMetric) TimerStart() {
	d.timer = prometheus.NewTimer(d.metric)
}

// This function stops the timer and record the Duration since the last call to TimerStart
func (d *DurationMetric) TimerStop() {
	d.timer.ObserveDuration()
}

type MetricsComponent struct {
	metricName            string
	latestInstallDuration *SimpleGaugeMetric
	latestUpgradeDuration *SimpleGaugeMetric
}

// This member function returns the simpleGaugeMetric that holds the install time for a component
func (m *MetricsComponent) getInstallDuration() *SimpleGaugeMetric {
	return m.latestInstallDuration
}

// This member function returns the simpleGaugeMetric that holds the upgrade time for a component
func (m *MetricsComponent) getUpgradeDuration() *SimpleGaugeMetric {
	return m.latestUpgradeDuration
}

type ComponentHealth struct {
	available *prometheus.GaugeVec
}

type ComponentInstallDuration struct {
	installDuration *prometheus.GaugeVec
}

type ComponentUpgradeDuration struct {
	upgradeDuration *prometheus.GaugeVec
}

// This member function returns the simpleGaugeMetric that holds the upgrade time for a component
func (c *ComponentHealth) SetComponentHealth(name string, availability bool, isEnabled bool) (prometheus.Gauge, error) {
	//isEnabled : true => 0, isEnabled : false => -1
	enabledVal := -1
	if isEnabled {
		enabledVal = 0
	}
	//availability : true => 1, availability : false => 0
	availableVal := 0
	if availability {
		availableVal = 1
	}
	//setting : enabled and available => 1, enabled and unavailable => 0, disabled = > -1
	setting := enabledVal + availableVal
	metric, err := c.available.GetMetricWithLabelValues(name)
	if err != nil {
		return nil, err
	}
	metric.Set(float64(setting))
	return metric, nil
}
