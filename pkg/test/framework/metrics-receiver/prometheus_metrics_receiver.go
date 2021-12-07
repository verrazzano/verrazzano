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
	pushGatewayURL      string
	pushGatewayUser     string
	pushGatewayPassword string
	pushInterval        time.Duration
	Name                string
}

type PrometheusMetricsReceiver struct {
	genericRvr GenericMetricsReceiver
	receiverConfig PrometheusMetricsReceiverConfig
	pusher              *push.Pusher
}

func (pmrs *PrometheusMetricsReceiver) pushData() error {
	for cntrName, cntrp := range pmrs.genericRvr.counters {
		ctr := prometheus.NewCounter(prometheus.CounterOpts{Name: cntrName})
		ctr.Add(cntrp.Value())
		pmrs.pusher.Collector(ctr)
	}
	if err := pmrs.pusher.Add(); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("could not push metric data to push gateway: %s", err.Error()))
	}
	return nil
}

func (pmrs *PrometheusMetricsReceiver) IncrementCounter(desc MetricDesc) error {
	return pmrs.genericRvr.IncrementCounter(desc)
}

func newPrometheusMetricsReceiver(rcvr GenericMetricsReceiver) (MetricsReceiver, error) {
	cfg := PrometheusMetricsReceiverConfig{
		pushGatewayURL:      getenvFunc(promPushURLEnvVarName),
		pushGatewayUser:     getenvFunc(promPushUserEnvVarName),
		pushGatewayPassword: getenvFunc(promPushPasswordEnvVarName),
		pushInterval:        defaultPushInterval,
		Name:                rcvr.name,
	}

	pmrcvr := PrometheusMetricsReceiver{receiverConfig: cfg, genericRvr: rcvr}
	pmrcvr.pusher = push.New(cfg.pushGatewayURL, cfg.Name)
	var v MetricsReceiver
	v = &pmrcvr

	go func() {
		// push the counter to the gateway
		for {
			time.Sleep(pmrcvr.receiverConfig.pushInterval)
			if err := pmrcvr.pushData(); err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("could not push metric to push gateway: %s", err.Error()))
			}
			pkg.Log(pkg.Info,"Successfully push metric data")
		}
	}()
	return v, nil
}

/*
func (pmrs *PrometheusMetricsReceiver) createCounter(opts MetricOpts) *metrics.Counter {
   var myCounter metrics.Counter
   //var optLabels stdprometheus.Labels
   //promCtrOpts
   var optLabels stdprometheus.Labels = make(map[string]string)
   if  opts.ConstLabels != nil {
	   for index, element := range opts.ConstLabels {
		   optLabels[index] = element
	   }
   }
   var promCtrOpts stdprometheus.CounterOpts = stdprometheus.CounterOpts{
	                  Namespace:opts.Namespace,
					  Subsystem: opts.Subsystem,
					  Name: opts.Name,
					  Help: opts.Help,
					  ConstLabels: optLabels,
   }

   myCounter = prometheus.NewCounterFrom(promCtrOpts, []string{})
   return &myCounter
}
*/




