// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package client

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/weblogic/todo"
	"io"
	"net/http"
	"strings"
	"sync"
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
	metricsPrefix = "echo_client"

	// EnvServiceName specifies the name of the service in the local cluster
	// By default, the ServiceName is not istio-ingressgateway
	EnvServiceName = "SERVICE_NAME"

	// EnvServiceNamespace specifies the namespace of the service in the local cluster
	// By default, the ServiceNamespace is istio-system
	EnvServiceNamespace = "SERVICE_NAMESPACE"

	// EnvPayload specifies the payload to PUT
	EnvPayload = "PAYLOAD"
)

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetricDef struct {
	metricDef todo.HTTPMetricDef
}

var portMutex sync.Mutex
var portIndex int
var ports = []int{31000, 31001, 31002, 31003, 31004}

type worker struct {
	metricDescList []prometheus.Desc
	*workerMetricDef
	client  *http.Client
	port    int
	host    string
	path    string
	payload string
}

var _ spi.Worker = worker{}

func NewWorker() (spi.Worker, error) {
	host := fmt.Sprintf("%s/%s.svc.cluster.local",
		config.PsrEnv.GetEnv(EnvServiceName),
		config.PsrEnv.GetEnv(EnvServiceNamespace))

	w := worker{
		host:           host,
		port:           getPort(),
		payload:        config.PsrEnv.GetEnv(EnvPayload),
		metricDescList: nil,
		workerMetricDef: &workerMetricDef{
			metricDef: todo.HTTPMetricDef{
				RequestsCountTotal: metrics.MetricItem{
					Name: "count_total",
					Help: "The total number of PUT requests",
					Type: prometheus.CounterValue,
				},
				RequestsSucceededCountTotal: metrics.MetricItem{
					Name: "succeeded_count_total",
					Help: "The total number of successful PUT requests",
					Type: prometheus.CounterValue,
				},
				RequestsFailedCountTotal: metrics.MetricItem{
					Name: "failed_count_total",
					Help: "The total number of failed PUT requests",
					Type: prometheus.CounterValue,
				},
				RequestDurationMicros: metrics.MetricItem{
					Name: "duration_micros",
					Help: "The duration of PUT request round trip in microseconds",
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
		&w.metricDef.RequestsCountTotal,
		&w.metricDef.RequestsSucceededCountTotal,
		&w.metricDef.RequestsFailedCountTotal,
		&w.metricDef.RequestDurationMicros,
	}, metricsLabels, w.GetWorkerDesc().MetricsPrefix)

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
		{Key: EnvServiceName, DefaultVal: "istio-ingressgateway", Required: false},
		{Key: EnvServiceNamespace, DefaultVal: "istio-system", Required: false},
		{Key: EnvPayload, DefaultVal: "aaa=bbb", Required: false},
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
		w.metricDef.RequestDurationMicros.BuildMetric(),
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
	startTime := time.Now().UnixMicro()

	_, err := w.putEcho()
	if err != nil {
		atomic.AddInt64(&w.metricDef.RequestsFailedCountTotal.Val, 1)
		log.Errorf("Port %d, Error %v - sleeping for a few secs", w.port, err)
		time.Sleep(5 * time.Second)
		return err
	}
	atomic.AddInt64(&w.metricDef.RequestsSucceededCountTotal.Val, 1)
	durationMicros := time.Now().UnixMicro() - startTime
	atomic.StoreInt64(&w.workerMetricDef.metricDef.RequestDurationMicros.Val, durationMicros)

	log.Progressf("PUT todo item succeeded")
	return nil
}

// getUserToken gets a user token from a secret
func (w worker) putEcho() (string, error) {
	reqURL := fmt.Sprintf("http://%s:%d/%s", w.host, w.port, w.path)
	req, err := http.NewRequest(http.MethodPut, reqURL, strings.NewReader(w.payload))
	if err != nil {
		return "", err
	}

	hostPort := fmt.Sprintf("%s:%v", w.host, w.port)
	req.Header.Add("Host", hostPort)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("accept", "*/*")
	req.Header.Add("accept-encoding", "*")
	req.Host = hostPort

	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("%s %v", resp.Status, resp.StatusCode)
	}

	defer resp.Body.Close()

	// resp.Body is consumed by the first try, and then no longer available (empty)
	// so we need to read the body and save it so we can use it in each retry
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// Get the next port, rotating through the ports
func getPort() int {
	portMutex.Lock()
	defer portMutex.Unlock()
	port := ports[portIndex]
	portIndex++
	if portIndex == len(ports) {
		portIndex = 0
	}
	return port
}
