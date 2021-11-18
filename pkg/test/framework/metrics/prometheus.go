// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testmetrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

type PrometheusMetricsReceiverConfig struct {
	PushGatewayUrl string
	PushGatewayUser string
	PushGatewayPassword string
	PushInterval time.Duration
	Name string
}

type PrometheusMetricsReceiver struct {
	promPusher *push.Pusher
	counters map[string]prometheus.Counter
	gauges map[string]prometheus.Gauge
}


func (pcfg *PrometheusMetricsReceiverConfig) GetReceiverType() string {
	return "PrometheusMetricsReceiver"
}

func (rcvr *PrometheusMetricsReceiver) SetGauge(name string, value float64) error {
	if rcvr.gauges == nil {
		rcvr.gauges = make(map[string]prometheus.Gauge)
	}
	gauge := rcvr.gauges[name]
	if gauge == nil {
		gauge = prometheus.NewGauge(prometheus.GaugeOpts{Name: name})
		rcvr.gauges[name] = gauge
	}
	gauge.Set(value)
	// TODO push the gauge
	return nil
}

func (rcvr *PrometheusMetricsReceiver) IncrementCounter(name string) error {
	if rcvr.counters == nil {
		rcvr.counters = make(map[string]prometheus.Counter)
	}
	ctr := rcvr.counters[name]
	if ctr == nil {
		ctr = prometheus.NewCounter(prometheus.CounterOpts{Name: name})
		rcvr.counters[name] = ctr
	}
	ctr.Inc()
	// TODO push the counter
	return nil
}

func NewPrometheusMetricsReceiver(cfg PrometheusMetricsReceiverConfig) (*PrometheusMetricsReceiver, error) {
	receiver := PrometheusMetricsReceiver{}
	pusher := push.New(cfg.PushGatewayUrl, cfg.Name)
	if cfg.PushGatewayUser != "" && cfg.PushGatewayPassword != "" {
		pusher = pusher.BasicAuth(cfg.PushGatewayUser, cfg.PushGatewayPassword)
	}
	receiver.promPusher = pusher
	return &receiver, nil
}