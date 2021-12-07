// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testmetrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"time"

	gokitmetrics "github.com/go-kit/kit/metrics/prometheus"
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
	counters   map[string] *gokitmetrics.Counter
	gauges     map[string] *gokitmetrics.Gauge
}

func (pcfg *PrometheusMetricsReceiverConfig) GetReceiverType() string {
	return "PrometheusMetricsReceiver"
}

func (rcvr *PrometheusMetricsReceiver) SetGauge(name string, value float64) error {
	if rcvr.gauges == nil {
		rcvr.gauges = make(map[string] *gokitmetrics.Gauge)
	}
	metricName := rcvr.makeMetricName(name)
	gauge := rcvr.gauges[metricName]
	if &gauge == nil {
		gauge = gokitmetrics.NewGaugeFrom(prometheus.GaugeOpts{Name: metricName}, []string{})
		rcvr.gauges[metricName] = gauge
	}
	gauge.Set(value)
	pkg.Log(pkg.Info, fmt.Sprintf("Emitting gauge %s with value %f", metricName, value))

	// Asynchronously push the gauge to the Prometheus push gateway
	rcvr.asyncPush(gauge, metricName)
	return nil
}

func (rcvr *PrometheusMetricsReceiver) IncrementCounter(name string) error {
	if rcvr.counters == nil {
		rcvr.counters = make(map[string] *gokitmetrics.Counter)
	}
	metricName := rcvr.makeMetricName(name)
	ctr := rcvr.counters[metricName]
	if &ctr == nil {
		ctr = gokitmetrics.NewCounterFrom(prometheus.CounterOpts{Name: metricName},[]string{})
		rcvr.counters[metricName] = ctr
	}
	ctr.Add(1)
	pkg.Log(pkg.Info, fmt.Sprintf("Incrementing counter %s", metricName))

	promctr := prometheus.NewCounter(prometheus.CounterOpts{Name: metricName})
	ctrp := rcvr.counters[metricName]
	promctr.Add(ctrp.Value()) //have to copy the counter value from gokit prometheus to client golang counter.

	//rcvr.asyncPush(ctr, metricName)
	rcvr.asyncPush(rcvr.promPusher.Collector(promctr))

	return nil
}

// Use a goroutine to asynchronously kick off a push to the Prometheus gateway represented by rcvr.promPusher
//func (rcvr *PrometheusMetricsReceiver) asyncPush(ctr prometheus.Collector, metricName string) {
func (rcvr *PrometheusMetricsReceiver) asyncPush(ctr prometheus.Collector) {
	go func() {
		// push the counter to the gateway
		if err := rcvr.promPusher.Collector(ctr).Add(); err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("could not push metric %s to push gateway: %s", metricName, err.Error()))
		}
		pkg.Log(pkg.Info, fmt.Sprintf("Successfully emitted metric %s", metricName))
	}()
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
