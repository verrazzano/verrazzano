// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl

package metrics_receiver

import (
	"github.com/go-kit/kit/metrics/generic"
	"os"
)

const (
	metricsReceiverTypeEnvVarName = "METRICS_RECEIVER_TYPE"
	promReceiverType              = "Prometheus"
)

type Labels map[string]string
type MetricDesc struct {
	Namespace   string
	Subsystem   string
	Name        string
	Help        string
	ConstLabels Labels
}

func (descp *MetricDesc) FQName() string {
	return descp.Namespace + "." + descp.Subsystem + "." + descp.Name
}

type MetricsReceiver interface {
    //push()
	//createCounter(MetricOpts) *metrics.Counter
	IncrementCounter(MetricDesc) error
}

type GenericMetricsReceiver struct {
	name string
	counters   map[string] *generic.Counter
	gauges     map[string] *generic.Gauge
	histograms map[string] *generic.Histogram
}


func newMetricsReceiver(name string) (MetricsReceiver, error) {
	var rcvr GenericMetricsReceiver = GenericMetricsReceiver{name : name,
	                                   counters : make(map[string] *generic.Counter),
									   gauges : make(map[string] *generic.Gauge),
		                               histograms : make(map[string] *generic.Histogram),
	}
	metricsReceiverType, isSet := os.LookupEnv(metricsReceiverTypeEnvVarName)
	var metReceiver MetricsReceiver
	var err error = nil
	if !isSet {
		metricsReceiverType = promReceiverType
	}
	if metricsReceiverType == promReceiverType {
		metReceiver, err = newPrometheusMetricsReceiver(rcvr)
	}
	return metReceiver, err
}

func (rcvr * GenericMetricsReceiver) IncrementCounter(desc MetricDesc) error {
	fqName := desc.FQName()
	cntrp := rcvr.counters[fqName]
	if cntrp == nil {
		cntrp := generic.NewCounter(fqName)
		rcvr.counters[fqName] = cntrp
	}
	cntrp.Add(1)
	return nil
}

func(rcvr * GenericMetricsReceiver) getCounterValue(desc MetricDesc) float64 {
	fqName := desc.FQName()
	cntrp := rcvr.counters[fqName]
	return cntrp.Value();
}
