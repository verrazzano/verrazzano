// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
)

type httpFunc func(url string) (resp *http.Response, err error)

var httpGetFunc httpFunc = http.Get

const (
	// metricsPrefix is the prefix that is automatically pre-pended to all metrics exported by this worker.
	metricsPrefix = "http_get"

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
	Path = "URL_PATH"
)

type worker struct {
	metricDescList []prometheus.Desc
	*workerMetricDef
}

var _ spi.Worker = worker{}

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetricDef struct {
	httpGetMetricDef    httpMetricDef
	httpPutMetricDef    httpMetricDef
	httpDeleteMetricDef httpMetricDef
}

type httpMetricDef struct {
	requestsCountTotal          metrics.MetricItem
	requestsSucceededCountTotal metrics.MetricItem
	requestsFailedCountTotal    metrics.MetricItem
	requestDurationMillis       metrics.MetricItem
}

func NewTodoWorker() (spi.Worker, error) {
	getMetricDef := httpMetricDef{
		requestsCountTotal: metrics.MetricItem{
			Name: "get_request_count_total",
			Help: "The total number of GET requests",
			Type: prometheus.CounterValue,
		}}

	w := worker{
		metricDescList: nil,
		workerMetricDef: &workerMetricDef{
			httpGetMetricDef: httpMetricDef{
				requestsCountTotal: metrics.MetricItem{
					Name: "get_request_count_total",
					Help: "The total number of GET requests",
					Type: prometheus.CounterValue,
				},
				requestsSucceededCountTotal: metrics.MetricItem{
					Name: "get_request_succeeded_count_total",
					Help: "The total number of successful GET requests",
					Type: prometheus.CounterValue,
				},
				requestsFailedCountTotal: metrics.MetricItem{
					Name: "get_request_failed_count_total",
					Help: "The total number of failed GET requests",
					Type: prometheus.CounterValue,
				},
				requestDurationMillis: metrics.MetricItem{
					Name: "get_request_duration_millis",
					Help: "The duration of GET request round trip in milliseconds",
					Type: prometheus.CounterValue,
				},
			},
			httpPutMetricDef: httpMetricDef{
				requestsCountTotal: metrics.MetricItem{
					Name: "put_request_count_total",
					Help: "The total number of PUT requests",
					Type: prometheus.CounterValue,
				},
				requestsSucceededCountTotal: metrics.MetricItem{
					Name: "put_request_succeeded_count_total",
					Help: "The total number of successful PUT requests",
					Type: prometheus.CounterValue,
				},
				requestsFailedCountTotal: metrics.MetricItem{
					Name: "put_request_failed_count_total",
					Help: "The total number of failed PUT requests",
					Type: prometheus.CounterValue,
				},
				requestDurationMillis: metrics.MetricItem{
					Name: "put_request_duration_millis",
					Help: "The duration of PUT request round trip in milliseconds",
					Type: prometheus.CounterValue,
				},
			},
			httpDeleteMetricDef: httpMetricDef{
				requestsCountTotal: metrics.MetricItem{
					Name: "delete_request_count_total",
					Help: "The total number of DELETE requests",
					Type: prometheus.CounterValue,
				},
				requestsSucceededCountTotal: metrics.MetricItem{
					Name: "delete_request_succeeded_count_total",
					Help: "The total number of successful DELETE requests",
					Type: prometheus.CounterValue,
				},
				requestsFailedCountTotal: metrics.MetricItem{
					Name: "delete_request_failed_count_total",
					Help: "The total number of failed DELETE requests",
					Type: prometheus.CounterValue,
				},
				requestDurationMillis: metrics.MetricItem{
					Name: "delete_request_duration_millis",
					Help: "The duration of DELETE request round trip in milliseconds",
					Type: prometheus.CounterValue,
				},
			},
		},
	}

	if err := config.PsrEnv.LoadFromEnv(w.GetEnvDescList()); err != nil {
		return w, err
	}

	metricsLabels := map[string]string{
		config.PsrWorkerTypeMetricsName: config.PsrEnv.GetEnv(config.PsrWorkerType),
	}

	w.metricDescList = metrics.BuildMetricDescList([]*metrics.MetricItem{
		&w.httpGetMetricDef.requestsCountTotal,
		&w.httpGetMetricDef.requestsSucceededCountTotal,
		&w.httpGetMetricDef.requestsFailedCountTotal,
		&w.httpGetMetricDef.requestDurationMillis,
		&w.httpPutMetricDef.requestsCountTotal,
		&w.httpPutMetricDef.requestsSucceededCountTotal,
		&w.httpPutMetricDef.requestsFailedCountTotal,
		&w.httpPutMetricDef.requestDurationMillis,
		&w.httpDeleteMetricDef.requestsCountTotal,
		&w.httpDeleteMetricDef.requestsSucceededCountTotal,
		&w.httpDeleteMetricDef.requestsFailedCountTotal,
		&w.httpDeleteMetricDef.requestDurationMillis,
	}, metricsLabels, w.GetWorkerDesc().MetricsPrefix)
	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w worker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		WorkerType:    config.WorkerTypeWlsTodo,
		Description:   "The get worker makes inserts an entry into TODO_LIST, gets it, then deletes it",
		MetricsPrefix: metricsPrefix,
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
		w.httpGetMetricDef.requestsCountTotal.BuildMetric(),
		w.httpGetMetricDef.requestsSucceededCountTotal.BuildMetric(),
		w.httpGetMetricDef.requestsFailedCountTotal.BuildMetric(),
		w.httpGetMetricDef.requestDurationMillis.BuildMetric(),
		w.httpPutMetricDef.requestsCountTotal.BuildMetric(),
		w.httpPutMetricDef.requestsSucceededCountTotal.BuildMetric(),
		w.httpPutMetricDef.requestsFailedCountTotal.BuildMetric(),
		w.httpPutMetricDef.requestDurationMillis.BuildMetric(),
		w.httpDeleteMetricDef.requestsCountTotal.BuildMetric(),
		w.httpDeleteMetricDef.requestsSucceededCountTotal.BuildMetric(),
		w.httpDeleteMetricDef.requestsFailedCountTotal.BuildMetric(),
		w.httpDeleteMetricDef.requestDurationMillis.BuildMetric(),
	}
}

func (w worker) WantLoopInfoLogged() bool {
	return false
}

func (w worker) PreconditionsMet() (bool, error) {
	return true, nil
}

func (w worker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	var lc, ls, lf int64

	URL := "http://" + config.PsrEnv.GetEnv(ServiceName) +
		"." + config.PsrEnv.GetEnv(ServiceNamespace) +
		".svc.cluster.local:" +
		config.PsrEnv.GetEnv(ServicePort) +
		"/" + config.PsrEnv.GetEnv(Path)
	err := w.doHttp(log, URL, httpGetFunc, &w.workerMetricDef.httpGetMetricDef)
	if err != nil {
		return err
	}

	return nil
}

func (w worker) doHttp(log vzlog.VerrazzanoLogger, URL string, f httpFunc,
	metricDef *httpMetricDef) error {
	var lc, ls, lf int64

	lc = atomic.AddInt64(&metricDef.requestsCountTotal.Val, 1)

	startTime := time.Now().UnixNano()
	resp, err := f(URL)
	durationMillis := (time.Now().UnixNano() - startTime) / 1000
	atomic.StoreInt64(&metricDef.requestDurationMillis.Val, durationMillis)

	if err != nil {
		atomic.AddInt64(&metricDef.requestsFailedCountTotal.Val, 1)
		return log.ErrorfNewErr("HTTP request %s returned error %v", URL, err)
	}
	if resp == nil {
		atomic.AddInt64(&metricDef.requestsFailedCountTotal.Val, 1)
		return log.ErrorfNewErr("HTTP request %s returned nil", URL)
	}
	if resp.StatusCode != 200 {
		atomic.AddInt64(&metricDef.requestsFailedCountTotal.Val, 1)
		return log.ErrorfNewErr("HTTP request %s error code &v", URL, resp.StatusCode)
	}
	// Success
	atomic.AddInt64(&metricDef.requestsSucceededCountTotal.Val, 1)
	log.Progressf("Http request to URL %s succeeded", URL)
	return nil
}
