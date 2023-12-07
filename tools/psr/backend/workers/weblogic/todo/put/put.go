// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package put

import (
	"fmt"
	"github.com/google/uuid"
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

	// Path specifies the path in the URL
	// By default, the path is not specified
	Path = "URL_PATH"
)

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetricDef struct {
	metricDef todo.HttpMetricDef
}

type worker struct {
	metricDescList []prometheus.Desc
	*workerMetricDef
	ID     *atomic.Int64
	UUID   uuid.UUID
	client *http.Client
}

var _ spi.Worker = worker{}

func NewWorker() (spi.Worker, error) {
	w := worker{
		metricDescList: nil,
		workerMetricDef: &workerMetricDef{
			metricDef: todo.HttpMetricDef{
				RequestsCountTotal: metrics.MetricItem{
					Name: "put_request_count_total",
					Help: "The total number of PUT requests",
					Type: prometheus.CounterValue,
				},
				RequestsSucceededCountTotal: metrics.MetricItem{
					Name: "put_request_succeeded_count_total",
					Help: "The total number of successful PUT requests",
					Type: prometheus.CounterValue,
				},
				RequestsFailedCountTotal: metrics.MetricItem{
					Name: "put_request_failed_count_total",
					Help: "The total number of failed PUT requests",
					Type: prometheus.CounterValue,
				},
				RequestDurationMillis: metrics.MetricItem{
					Name: "put_request_duration_millis",
					Help: "The duration of PUT request round trip in milliseconds",
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
		&w.metricDef.RequestsCountTotal,
		&w.metricDef.RequestsSucceededCountTotal,
		&w.metricDef.RequestsFailedCountTotal,
		&w.metricDef.RequestDurationMillis,
	}, metricsLabels, w.GetWorkerDesc().MetricsPrefix)

	// Init IDs
	w.UUID = uuid.New()
	w.ID = &atomic.Int64{}

	// Create http client
	w.client = &http.Client{}
	return w, nil
}

// GetWorkerDesc returns the WorkerDes for the worker
func (w worker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		WorkerType:    config.WorkerTypeWlsTodoPut,
		Description:   "The get worker makes inserts an entry into TODO LIST database",
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
		w.metricDef.RequestsCountTotal.BuildMetric(),
		w.metricDef.RequestsSucceededCountTotal.BuildMetric(),
		w.metricDef.RequestsFailedCountTotal.BuildMetric(),
		w.metricDef.RequestDurationMillis.BuildMetric(),
	}
}

func (w worker) WantLoopInfoLogged() bool {
	return false
}

func (w worker) PreconditionsMet() (bool, error) {
	return true, nil
}

func (w worker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	atomic.AddInt64(&w.workerMetricDef.metricDef.RequestsCountTotal.Val, 1)
	err := w.doPut(conf, log)
	if err != nil {
		atomic.AddInt64(&w.metricDef.RequestsFailedCountTotal.Val, 1)
	}
	atomic.AddInt64(&w.metricDef.RequestsSucceededCountTotal.Val, 1)
	log.Progressf("PUT succeeded")
	return nil
}

func (w worker) doPut(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	newID := w.ID.Add(1)
	item := fmt.Sprintf("%v-%v", newID, w.UUID)

	URL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%s/todo/rest/item/%s",
		config.PsrEnv.GetEnv(ServiceName),
		config.PsrEnv.GetEnv(ServiceNamespace),
		config.PsrEnv.GetEnv(ServicePort),
		item)

	startTime := time.Now().UnixNano()
	req, err := http.NewRequest(http.MethodPut, URL, nil)
	durationMillis := (time.Now().UnixNano() - startTime) / 1000
	atomic.StoreInt64(&w.workerMetricDef.metricDef.RequestDurationMillis.Val, durationMillis)
	if err != nil {
		return log.ErrorfNewErr("HTTP request body NewRequest for URL %s returned error %v", URL, err)
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return log.ErrorfNewErr("HTTP PUT failed for URL %s returned error %v", URL, err)
	}
	_, err = todo.HandleResponse(log, URL, &w.workerMetricDef.metricDef, resp, err)
	if err != nil {
		return err
	}
	return nil
}
