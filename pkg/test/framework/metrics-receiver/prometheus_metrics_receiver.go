// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl

package metrics_receiver

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"os"
	"time"
)

const (
	// Prometheus related env vars and constants
	promPushURLEnvVarName      = "PROMETHEUS_GW_URL"
	promPushUserEnvVarName     = "PROMETHEUS_CREDENTIALS_USR"
	promPushPasswordEnvVarName = "PROMETHEUS_CREDENTIALS_PSW"

	defaultPushInterval        = time.Minute
)

var getenvFunc = os.Getenv

type PrometheusMetricsReceiverConfig struct {
	PushGatewayURL      string
	PushGatewayUser     string
	PushGatewayPassword string
	PushInterval        time.Duration
	Name                string
}

type PrometheusMetricsReceiver struct {
	genericRvr GenericMetricsReceiver
	receiverConfig PrometheusMetricsReceiverConfig
	pusher              *push.Pusher
	counters            map[string] prometheus.Counter
}


func (pmrs *PrometheusMetricsReceiver) Name() string {
	return pmrs.receiverConfig.Name
}

func (pmrs *PrometheusMetricsReceiver) pushData() error {
	if err := pmrs.pusher.Add(); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("could not push metric data to push gateway: %s", err.Error()))
	} else {
		pkg.Log(pkg.Info, "Successfully pushed counter metric")
		fmt.Println("Successfully pushed")
	}
	return nil
}

func (pmrs *PrometheusMetricsReceiver) IncrementGokitCounter(desc MetricDesc) error {
	err := pmrs.genericRvr.IncrementGokitCounter(desc)
	pcntr := pmrs.counters[desc.Name]
	if pcntr == nil {
		pcntr = prometheus.NewCounter(prometheus.CounterOpts{Name: desc.Name})
		pmrs.counters[desc.Name] = pcntr
		pmrs.pusher.Collector(pcntr)
	}
	pcntr.Inc()
	return err
}

func(pmrs * PrometheusMetricsReceiver) GetCounterValue(desc MetricDesc) int {
	return pmrs.genericRvr.GetCounterValue(desc)
}
func newPrometheusMetricsReceiver(rcvr GenericMetricsReceiver) (GokitMetricsReceiver, error) {
	cfg := PrometheusMetricsReceiverConfig{
		PushGatewayURL:      getenvFunc(promPushURLEnvVarName),
		PushGatewayUser:     getenvFunc(promPushUserEnvVarName),
		PushGatewayPassword: getenvFunc(promPushPasswordEnvVarName),
		PushInterval:        defaultPushInterval,
		Name:                rcvr.name,
	}

	pmrcvr := PrometheusMetricsReceiver{receiverConfig: cfg,
		                                counters : make(map[string] prometheus.Counter),
		                                genericRvr: rcvr}
	pmrcvr.pusher = push.New(cfg.PushGatewayURL, cfg.Name)
	if pmrcvr.receiverConfig.PushGatewayUser != "" && pmrcvr.receiverConfig.PushGatewayPassword != "" {
		pmrcvr.pusher = pmrcvr.pusher.BasicAuth(pmrcvr.receiverConfig.PushGatewayUser, pmrcvr.receiverConfig.PushGatewayPassword)
	}
	var v GokitMetricsReceiver
	v = &pmrcvr

	go func() {
		// push the counter to the gateway
		for {
			//time.Sleep(pmrcvr.receiverConfig.PushInterval)
			if err := pmrcvr.pushData(); err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("could not push metric to push gateway: %s", err.Error()))
			} else {
				pkg.Log(pkg.Info, "Successfully pushed metric data")
			}
			time.Sleep(pmrcvr.receiverConfig.PushInterval)
		}
	}()
	return v, nil
}





