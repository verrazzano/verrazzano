// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl

package metricsreceiver

import (
	"fmt"
	"github.com/go-kit/kit/metrics/generic"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
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
	//return descp.Namespace + "." + descp.Subsystem + "." + descp.Name
	return descp.Name
}

type GokitMetricsReceiver interface {
	Name() string
	IncrementGokitCounter(MetricDesc) error
	GetCounterValue(MetricDesc) int
}

type GenericMetricsReceiver struct {
	name       string
	counters   map[string]*generic.Counter
	gauges     map[string]*generic.Gauge
	histograms map[string]*generic.Histogram
}

func NewMetricsReceiver(name string) (GokitMetricsReceiver, error) {
	var rcvr GenericMetricsReceiver = GenericMetricsReceiver{name: name,
		counters:   make(map[string]*generic.Counter),
		gauges:     make(map[string]*generic.Gauge),
		histograms: make(map[string]*generic.Histogram),
	}
	metricsReceiverType, isSet := os.LookupEnv(metricsReceiverTypeEnvVarName)
	var metReceiver GokitMetricsReceiver
	var err error = nil
	if !isSet {
		metricsReceiverType = promReceiverType
	}
	if metricsReceiverType == promReceiverType {
		//if metricsReceiverType == "Prometheus" {
		metReceiver, err = newPrometheusMetricsReceiver(rcvr)
		//}
	}
	return metReceiver, err
}

func (rcvr *GenericMetricsReceiver) IncrementGokitCounter(desc MetricDesc) error {
	fqName := desc.FQName()
	cntrp := rcvr.counters[fqName]
	if cntrp == nil {
		cntrp = generic.NewCounter(fqName)
		rcvr.counters[fqName] = cntrp
	}
	cntrp.Add(1)
	pkg.Log(pkg.Info, fmt.Sprintf("Incrementing counter %s", fqName))
	return nil
}

func (rcvr *GenericMetricsReceiver) GetCounterValue(desc MetricDesc) int {
	fqName := desc.FQName()
	cntrp := rcvr.counters[fqName]
	return int(cntrp.Value())
}
