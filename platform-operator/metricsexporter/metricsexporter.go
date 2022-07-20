// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

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
	simpleCounterMetricMap map[metricName]*simpleCounterMetric
	simpleGaugeMetricMap   map[metricName]*simpleGaugeMetric
	durationMetricMap      map[metricName]*durationMetric
	metricsComponentMap    map[metricName]*metricsComponent
}
type simpleCounterMetric struct {
	metric prometheus.Counter
}

func (c *simpleCounterMetric) Inc(log *zap.SugaredLogger, err error) {
	c.metric.Inc()
	if err != nil {
		log.Error(err)
	}
}

func (c *simpleCounterMetric) Add(num float64) {
	c.metric.Add(num)
}
func (c *simpleCounterMetric) Get() prometheus.Counter {
	return c.metric
}

type simpleGaugeMetric struct {
	metric prometheus.Gauge
}

func (g *simpleGaugeMetric) Set(num float64) {
	g.metric.Set(num)
}

func (g *simpleGaugeMetric) SetToCurrentTime() {
	g.metric.SetToCurrentTime()
}

func (g *simpleGaugeMetric) Add(num float64) {
	g.metric.Add(num)
}
func (c *simpleGaugeMetric) Get() prometheus.Gauge {
	return c.metric
}

type durationMetric struct {
	metric prometheus.Summary
	timer  *prometheus.Timer
}

//Creates a new timer, and starts the timer
func (d *durationMetric) TimerStart() {
	d.timer = prometheus.NewTimer(d.metric)
}

//stops the timer and record the duration since the last call to TimerStart
func (d *durationMetric) TimerStop() {
	d.timer.ObserveDuration()
}

type metricsComponent struct {
	latestInstallDuration *simpleGaugeMetric
	latestUpgradeDuration *simpleGaugeMetric
}

func (m *metricsComponent) getInstall() *simpleGaugeMetric {
	return m.latestInstallDuration
}
func (m *metricsComponent) getUpgrade() *simpleGaugeMetric {
	return m.latestUpgradeDuration
}
