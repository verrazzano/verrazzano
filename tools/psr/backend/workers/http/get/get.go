// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package http

import (
	"io"
	"net/http"
)

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"sync/atomic"
)

type get struct {
	spi.Worker
	metricDescList []prometheus.Desc
	*workerMetrics
}

var _ spi.Worker = get{}

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	getRequestsCountTotal          metrics.MetricItem
	getRequestsSucceededCountTotal metrics.MetricItem
	getRequestsFailedCountTotal    metrics.MetricItem
}

func NewHTTPGetWorker() (spi.Worker, error) {
	w := get{workerMetrics: &workerMetrics{
		getRequestsCountTotal: metrics.MetricItem{
			Name: "get_request_count_total",
			Help: "The total number of GET requests",
			Type: prometheus.CounterValue,
		},
		getRequestsSucceededCountTotal: metrics.MetricItem{
			Name: "get_request_succeeded_count_total",
			Help: "The total number of successful GET requests",
			Type: prometheus.CounterValue,
		},
		getRequestsFailedCountTotal: metrics.MetricItem{
			Name: "get_request_failed_count_total",
			Help: "The total number of failed GET requests",
			Type: prometheus.CounterValue,
		},
	}}

	w.metricDescList = []prometheus.Desc{
		*w.getRequestsCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.getRequestsSucceededCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.getRequestsFailedCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
	}
	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w get) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		EnvName:     config.WorkerTypeHTTPGet,
		Description: "The get worker makes GET request on the given endpoint",
		MetricsName: "httpget",
	}
}

func (w get) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{
		{Key: config.ServiceName, DefaultVal: "", Required: true},
		{Key: config.ServiceNamespace, DefaultVal: "", Required: true},
		{Key: config.ServicePort, DefaultVal: "", Required: true},
		{Key: config.Path, DefaultVal: "", Required: true},
	}
}

func (w get) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w get) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.getRequestsCountTotal.BuildMetric(),
		w.getRequestsSucceededCountTotal.BuildMetric(),
		w.getRequestsFailedCountTotal.BuildMetric(),
	}
}

func (w get) WantIterationInfoLogged() bool {
	return false
}

func (w get) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	var lc, ls, lf int64
	//increment getRequestsCountTotal
	lc = atomic.AddInt64(&w.workerMetrics.getRequestsCountTotal.Val, 1)
	resp, err := http.Get("http://" + config.PsrEnv.GetEnv(config.ServiceName) +
		"." + config.PsrEnv.GetEnv(config.ServiceNamespace) +
		".svc.cluster.local:" +
		config.PsrEnv.GetEnv(config.ServicePort) +
		"/" + config.PsrEnv.GetEnv(config.Path))
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("GET request to endpoint received a nil response")
	}
	if resp.StatusCode == 200 {
		ls = atomic.AddInt64(&w.workerMetrics.getRequestsSucceededCountTotal.Val, 1)
	} else {
		lf = atomic.AddInt64(&w.workerMetrics.getRequestsFailedCountTotal.Val, 1)
		//Read the response body on the line below.
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Error reading response body: %v", err)
		}
		//Convert the body to type string
		sb := string(body)
		log.Errorf("The response body: ", sb)
	}
	logMsg := fmt.Sprintf("HttpGet worker total requests %v, "+
		" total successful requests %v, total failed requests %v",
		lc, ls, lf)
	log.Debugf(logMsg)
	return nil
}
