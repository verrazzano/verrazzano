// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package delete

import (
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/weblogic/todo"
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

const (
	// metricsPrefix is the prefix that is automatically pre-pended to all metrics exported by this worker.
	metricsPrefix = "todo"

	// ServiceName specifies the name of the service in the local cluster
	// By default, the ServiceName is not specified
	ServiceName = "SERVICE_NAME"

	// ServiceNamespace specifies the namespace of the service in the local cluster
	// By default, the ServiceNamespace is not specified
	ServiceNamespace = "SERVICE_NAMESPACE"

	// ServicePort specifies the port of the service in the local cluster
	// By default, the ServicePort is not specified
	ServicePort = "SERVICE_PORT"
)

// TodoItems is needed to unmarshal the REST GET response
type TodoItems struct {
	ToDos []TodoItem
}

// TodoItem is needed to unmarshal the REST GET response
type TodoItem struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
}

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetricDef struct {
	metricGetDef    todo.HttpMetricDef
	metricDeleteDef todo.HttpMetricDef
}

type worker struct {
	metricDescList []prometheus.Desc
	*workerMetricDef
	client *http.Client
}

var _ spi.Worker = worker{}

func NewWorker() (spi.Worker, error) {
	w := worker{
		metricDescList: nil,
		workerMetricDef: &workerMetricDef{
			metricGetDef: todo.HttpMetricDef{
				RequestsCountTotal: metrics.MetricItem{
					Name: "get_request_count_total",
					Help: "The total number of GET requests",
					Type: prometheus.CounterValue,
				},
				RequestsSucceededCountTotal: metrics.MetricItem{
					Name: "get_request_succeeded_count_total",
					Help: "The total number of successful GET requests",
					Type: prometheus.CounterValue,
				},
				RequestsFailedCountTotal: metrics.MetricItem{
					Name: "get_request_failed_count_total",
					Help: "The total number of failed GET requests",
					Type: prometheus.CounterValue,
				},
				RequestDurationMicros: metrics.MetricItem{
					Name: "get_request_duration_micros",
					Help: "The duration of GET request round trip in microseconds",
					Type: prometheus.GaugeValue,
				},
			},
			metricDeleteDef: todo.HttpMetricDef{
				RequestsCountTotal: metrics.MetricItem{
					Name: "delete_request_count_total",
					Help: "The total number of DELETE requests",
					Type: prometheus.CounterValue,
				},
				RequestsSucceededCountTotal: metrics.MetricItem{
					Name: "delete_request_succeeded_count_total",
					Help: "The total number of successful DELETE requests",
					Type: prometheus.CounterValue,
				},
				RequestsFailedCountTotal: metrics.MetricItem{
					Name: "delete_request_failed_count_total",
					Help: "The total number of failed DELETE requests",
					Type: prometheus.CounterValue,
				},
				RequestDurationMicros: metrics.MetricItem{
					Name: "delete_request_duration_micros",
					Help: "The duration of DELETE request round trip in microseconds",
					Type: prometheus.GaugeValue,
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
		&w.metricGetDef.RequestsCountTotal,
		&w.metricGetDef.RequestsSucceededCountTotal,
		&w.metricGetDef.RequestsFailedCountTotal,
		&w.metricGetDef.RequestDurationMicros,
		&w.metricDeleteDef.RequestsCountTotal,
		&w.metricDeleteDef.RequestsSucceededCountTotal,
		&w.metricDeleteDef.RequestsFailedCountTotal,
		&w.metricDeleteDef.RequestDurationMicros,
	}, metricsLabels, w.GetWorkerDesc().MetricsPrefix)

	// Create http client
	w.client = &http.Client{}
	return w, nil
}

// GetWorkerDesc returns the WorkerDes for the worker
func (w worker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		WorkerType:    config.WorkerTypeWlsTodoDelete,
		Description:   "The get worker makes gets all entries in the TODO LIST database and deletes them all.",
		MetricsPrefix: metricsPrefix,
	}
}

func (w worker) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{
		{Key: ServiceName, DefaultVal: "", Required: true},
		{Key: ServiceNamespace, DefaultVal: "", Required: true},
		{Key: ServicePort, DefaultVal: "", Required: true},
	}
}

func (w worker) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w worker) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.metricGetDef.RequestsCountTotal.BuildMetric(),
		w.metricGetDef.RequestsSucceededCountTotal.BuildMetric(),
		w.metricGetDef.RequestsFailedCountTotal.BuildMetric(),
		w.metricGetDef.RequestDurationMicros.BuildMetric(),
		w.metricDeleteDef.RequestsCountTotal.BuildMetric(),
		w.metricDeleteDef.RequestsSucceededCountTotal.BuildMetric(),
		w.metricDeleteDef.RequestsFailedCountTotal.BuildMetric(),
		w.metricDeleteDef.RequestDurationMicros.BuildMetric(),
	}
}

func (w worker) WantLoopInfoLogged() bool {
	return false
}

func (w worker) PreconditionsMet() (bool, error) {
	return true, nil
}

func (w worker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	// Get all the items
	atomic.AddInt64(&w.workerMetricDef.metricGetDef.RequestsCountTotal.Val, 1)
	body, err := w.doGet(log)
	if err != nil {
		atomic.AddInt64(&w.metricGetDef.RequestsFailedCountTotal.Val, 1)
		return err
	}
	atomic.AddInt64(&w.metricGetDef.RequestsSucceededCountTotal.Val, 1)

	items, err := buildItemsFromJSON(log, body)
	if err != nil {
		return err
	}

	log.Progressf("GET all todo items succeeded")

	// Delete all the items that match this UUID
	for _, item := range items.ToDos {
		atomic.AddInt64(&w.workerMetricDef.metricDeleteDef.RequestsCountTotal.Val, 1)
		err := w.doDelete(log, item.ID)
		if err != nil {
			atomic.AddInt64(&w.metricDeleteDef.RequestsFailedCountTotal.Val, 1)
			return err
		}
		atomic.AddInt64(&w.metricDeleteDef.RequestsSucceededCountTotal.Val, 1)
	}

	log.Progressf("DELETE all todo items succeeded")
	return nil
}

// Get all the items
func (w worker) doGet(log vzlog.VerrazzanoLogger) ([]byte, error) {
	URL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%s/todo/rest/items",
		config.PsrEnv.GetEnv(ServiceName),
		config.PsrEnv.GetEnv(ServiceNamespace),
		config.PsrEnv.GetEnv(ServicePort))

	startTime := time.Now().UnixMicro()
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, log.ErrorfNewErr("HTTP request body NewRequest for URL %s returned error %v", URL, err)
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, log.ErrorfNewErr("HTTP GET failed for URL %s returned error %v", URL, err)
	}
	body, err := todo.HandleResponse(log, URL, resp, err, true)
	if err != nil {
		return nil, err
	}

	durationMicros := time.Now().UnixMicro() - startTime
	atomic.StoreInt64(&w.workerMetricDef.metricGetDef.RequestDurationMicros.Val, durationMicros)
	return body, nil
}

func (w worker) doDelete(log vzlog.VerrazzanoLogger, ID int) error {
	URL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%s/todo/rest/item/%v",
		config.PsrEnv.GetEnv(ServiceName),
		config.PsrEnv.GetEnv(ServiceNamespace),
		config.PsrEnv.GetEnv(ServicePort),
		ID)

	startTime := time.Now().UnixMicro()
	req, err := http.NewRequest(http.MethodDelete, URL, nil)
	if err != nil {
		return log.ErrorfNewErr("HTTP request body NewRequest for URL %s returned error %v", URL, err)
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return log.ErrorfNewErr("HTTP GET failed for URL %s returned error %v", URL, err)
	}
	_, err = todo.HandleResponse(log, URL, resp, err, false)
	if err != nil {
		return err
	}

	durationMicros := time.Now().UnixMicro() - startTime
	atomic.StoreInt64(&w.workerMetricDef.metricDeleteDef.RequestDurationMicros.Val, durationMicros)
	return nil
}

// Build the items from JSON
func buildItemsFromJSON(log vzlog.VerrazzanoLogger, body []byte) (TodoItems, error) {
	var items TodoItems
	err := json.Unmarshal(body, &items.ToDos)
	if err != nil {
		return TodoItems{}, log.ErrorfNewErr("Failed to parse response body %v: %v", body, err)
	}
	return items, nil
}
