// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testmetrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

type PrometheusMetricsReceiverConfig struct {
	PushGatewayURL      string
	PushGatewayUser     string
	PushGatewayPassword string
	PushInterval        time.Duration
	Name                string
}

type PrometheusMetricsReceiver struct {
	promPusher *push.Pusher
	Name       string
	counters   map[string]prometheus.Counter
	gauges     map[string]prometheus.Gauge
}

func (pcfg *PrometheusMetricsReceiverConfig) GetReceiverType() string {
	return "PrometheusMetricsReceiver"
}

func (rcvr *PrometheusMetricsReceiver) SetGauge(name string, value float64) error {
	if rcvr.gauges == nil {
		rcvr.gauges = make(map[string]prometheus.Gauge)
	}
	metricName := rcvr.makeMetricName(name)
	gauge := rcvr.gauges[metricName]
	if gauge == nil {
		gauge = prometheus.NewGauge(prometheus.GaugeOpts{Name: metricName})
		rcvr.gauges[metricName] = gauge
	}
	gauge.Set(value)
	pkg.Log(pkg.Info, fmt.Sprintf("Emitting gauge %s with value %f", metricName, value))

	// push the gauge to the gateway
	if err := rcvr.promPusher.Collector(gauge).Add(); err != nil {
		fmt.Println("Could not push to Pushgateway:", err)
	} else {
		pkg.Log(pkg.Info, fmt.Sprintf("Successfully emitted guage %s with value %f", metricName, value))
	}

	return nil
}

func (rcvr *PrometheusMetricsReceiver) IncrementCounter(name string) error {
	if rcvr.counters == nil {
		rcvr.counters = make(map[string]prometheus.Counter)
	}
	metricName := rcvr.makeMetricName(name)
	ctr := rcvr.counters[metricName]
	if ctr == nil {
		ctr = prometheus.NewCounter(prometheus.CounterOpts{Name: metricName})
		rcvr.counters[metricName] = ctr
	}
	ctr.Inc()
	pkg.Log(pkg.Info, fmt.Sprintf("Incrementing counter %s", metricName))

	// push the counter to the gateway
	if err := rcvr.promPusher.Collector(ctr).Add(); err != nil {
		fmt.Println("Could not push to Pushgateway:", err)
	} else {
		pkg.Log(pkg.Info, fmt.Sprintf("Successfully incremented counter %s", metricName))
	}

	return nil
}

// Create a new PrometheusMetricsReceiver based on the configuration options provided
func NewPrometheusMetricsReceiver(cfg PrometheusMetricsReceiverConfig) (*PrometheusMetricsReceiver, error) {
	receiver := PrometheusMetricsReceiver{}
	pusher := push.New(cfg.PushGatewayURL, cfg.Name)
	if cfg.PushGatewayUser != "" && cfg.PushGatewayPassword != "" {
		pusher = pusher.BasicAuth(cfg.PushGatewayUser, cfg.PushGatewayPassword)
	}
	receiver.promPusher = pusher
	receiver.Name = cfg.Name
	return &receiver, nil
}

// overridePusher overrides the Prometheus pusher used by this metrics receiver - tests may
// use this function to mock the pusher
func (rcvr *PrometheusMetricsReceiver) overridePusher(pusher push.Pusher) {
	rcvr.promPusher = &pusher
}

func (rcvr *PrometheusMetricsReceiver) makeMetricName(name string) string {
	if rcvr.Name != "" {
		return rcvr.Name + "_" + name
	}
	return name
}
