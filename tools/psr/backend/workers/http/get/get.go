// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package http

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
)

var httpGetFunc func(url string) (resp *http.Response, err error) = http.Get

const (
	// ServiceName specifies the name of the service in the local cluster
	// By default, the ServiceName is not specified
	ServiceName = "SERVICE_NAME"

	// ServiceNamespace specifies the namespace of the service in the local cluster
	// By default, the ServiceNamespace is not specified
	ServiceNamespace = "SERVICE_NAMESPACE"

	// ServicePort specifies the port of the service in the local cluster
	// By default, the ServicePort is not specified
	ServicePort = "SERVICE_PORT"

	// Path specifies the path in the URL
	// By default, the path is not specified
	Path = "PATH"
)

type worker struct {
	metricDescList []prometheus.Desc
	*workerMetrics
}

var _ spi.Worker = worker{}

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	getRequestsCountTotal          metrics.MetricItem
	getRequestsSucceededCountTotal metrics.MetricItem
	getRequestsFailedCountTotal    metrics.MetricItem
}

func NewHTTPGetWorker() (spi.Worker, error) {
	w := worker{workerMetrics: &workerMetrics{
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
func (w worker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		WorkerType:  config.WorkerTypeHTTPGet,
		Description: "The get worker makes GET request on the given endpoint",
		MetricsName: config.WorkerTypeHTTPGet,
	}
}

func (w worker) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{
		{Key: ServiceName, DefaultVal: "", Required: true},
		{Key: ServiceNamespace, DefaultVal: "", Required: true},
		{Key: ServicePort, DefaultVal: "", Required: true},
		{Key: Path, DefaultVal: "", Required: true},
	}
}

func (w worker) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w worker) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.getRequestsCountTotal.BuildMetric(),
		w.getRequestsSucceededCountTotal.BuildMetric(),
		w.getRequestsFailedCountTotal.BuildMetric(),
	}
}

func (w worker) WantLoopInfoLogged() bool {
	return false
}

func (w worker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	var lc, ls, lf int64
	//increment getRequestsCountTotal
	lc = atomic.AddInt64(&w.workerMetrics.getRequestsCountTotal.Val, 1)
	resp, err := httpGetFunc("http://" + config.PsrEnv.GetEnv(ServiceName) +
		"." + config.PsrEnv.GetEnv(ServiceNamespace) +
		".svc.cluster.local:" +
		config.PsrEnv.GetEnv(ServicePort) +
		"/" + config.PsrEnv.GetEnv(Path))
	if err != nil {
		lf = atomic.AddInt64(&w.workerMetrics.getRequestsFailedCountTotal.Val, 1)
		return err
	}
	if resp == nil {
		lf = atomic.AddInt64(&w.workerMetrics.getRequestsFailedCountTotal.Val, 1)
		return fmt.Errorf("GET request to endpoint received a nil response")
	}
	if resp.StatusCode == 200 {
		ls = atomic.AddInt64(&w.workerMetrics.getRequestsSucceededCountTotal.Val, 1)
	} else {
		lf = atomic.AddInt64(&w.workerMetrics.getRequestsFailedCountTotal.Val, 1)
	}
	logMsg := fmt.Sprintf("HttpGet worker total requests %v, "+
		" total successful requests %v, total failed requests %v",
		lc, ls, lf)
	log.Debugf(logMsg)
	return nil
}
