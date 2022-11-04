// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package postlogs

import (
	"bytes"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/security/password"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"

	"github.com/prometheus/client_golang/prometheus"
)

const LogEntries = "LOG_ENTRIES"
const LogLength = "LOG_LENGTH"

const osIngestService = "vmi-system-es-ingest.verrazzano-system:9200"

type postLogs struct {
	spi.Worker
	metricDescList []prometheus.Desc
	*workerMetrics
}

var _ spi.Worker = postLogs{}

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	openSearchPostSuccessCountTotal         metrics.MetricItem
	openSearchPostFailureCountTotal         metrics.MetricItem
	openSearchPostSuccessLatencyNanoSeconds metrics.MetricItem
	openSearchPostFailureLatencyNanoSeconds metrics.MetricItem
	openSearchPostDataCharsTotal            metrics.MetricItem
}

func NewPostLogsWorker() (spi.Worker, error) {
	w := postLogs{workerMetrics: &workerMetrics{
		openSearchPostSuccessCountTotal: metrics.MetricItem{
			Name: "opensearch_post_success_count_total",
			Help: "The total number of successful OpenSearch POST requests",
			Type: prometheus.CounterValue,
		},
		openSearchPostFailureCountTotal: metrics.MetricItem{
			Name: "opensearch_post_failure_count_total",
			Help: "The total number of successful OpenSearch POST requests",
			Type: prometheus.CounterValue,
		},
		openSearchPostSuccessLatencyNanoSeconds: metrics.MetricItem{
			Name: "opensearch_post_success_latency_nanoseconds",
			Help: "The latency of successful OpenSearch POST requests in nanoseconds",
			Type: prometheus.GaugeValue,
		},
		openSearchPostFailureLatencyNanoSeconds: metrics.MetricItem{
			Name: "opensearch_post_failure_latency_nanoseconds",
			Help: "The latency of failed OpenSearch POST requests in nanoseconds",
			Type: prometheus.GaugeValue,
		},
		openSearchPostDataCharsTotal: metrics.MetricItem{
			Name: "opensearch_post_data_chars_total",
			Help: "The total number of characters posted to OpenSearch",
			Type: prometheus.CounterValue,
		},
	}}

	w.metricDescList = []prometheus.Desc{
		*w.openSearchPostSuccessCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.openSearchPostFailureCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.openSearchPostSuccessLatencyNanoSeconds.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.openSearchPostFailureLatencyNanoSeconds.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.openSearchPostDataCharsTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
	}
	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w postLogs) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		EnvName:     config.WorkerTypePostLogs,
		Description: "The postlogs worker performs POST requests on the OpenSearch endpoint",
		MetricsName: "postlogs",
	}
}

func (w postLogs) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{
		{Key: LogEntries, DefaultVal: "1", Required: false},
		{Key: LogLength, DefaultVal: "1", Required: false},
	}
}

func (w postLogs) WantLoopInfoLogged() bool {
	return false
}

func (w postLogs) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	c := http.Client{}
	logEntries, err := strconv.Atoi(config.PsrEnv.GetEnv(LogEntries))
	if err != nil {
		return err
	}
	logLength, err := strconv.Atoi(config.PsrEnv.GetEnv(LogLength))
	if err != nil {
		return err
	}

	body, bodyChars, err := getBody(logEntries, logLength)
	if err != nil {
		return err
	}

	req := http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "http",
			Host:   osIngestService,
			Path:   fmt.Sprintf("/verrazzano-application-%s/_bulk", conf.Namespace),
		},
		Header: http.Header{"Content-Type": {"application/json"}},
		Body:   body,
	}

	startRequest := time.Now().UnixNano()
	resp, err := c.Do(&req)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("POST request to URI %s received a nil response", req.URL.RequestURI())
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		atomic.StoreInt64(&w.workerMetrics.openSearchPostFailureLatencyNanoSeconds.Val, time.Now().UnixNano()-startRequest)
		atomic.AddInt64(&w.workerMetrics.openSearchPostFailureCountTotal.Val, 1)
		return fmt.Errorf("OpenSearch POST request failed, returned %v status code with status: %s", resp.StatusCode, resp.Status)
	}
	atomic.StoreInt64(&w.workerMetrics.openSearchPostSuccessLatencyNanoSeconds.Val, time.Now().UnixNano()-startRequest)
	atomic.AddInt64(&w.workerMetrics.openSearchPostSuccessCountTotal.Val, 1)
	atomic.AddInt64(&w.workerMetrics.openSearchPostDataCharsTotal.Val, bodyChars)
	return nil
}

func (w postLogs) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w postLogs) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.openSearchPostSuccessCountTotal.BuildMetric(),
		w.openSearchPostFailureCountTotal.BuildMetric(),
		w.openSearchPostSuccessLatencyNanoSeconds.BuildMetric(),
		w.openSearchPostFailureLatencyNanoSeconds.BuildMetric(),
		w.openSearchPostDataCharsTotal.BuildMetric(),
	}
}

func getBody(logCount int, dataLength int) (io.ReadCloser, int64, error) {
	var body string
	for i := 0; i < logCount; i++ {
		data, err := password.GeneratePassword(dataLength)
		if err != nil {
			return nil, 0, err
		}
		body = body + "{\"create\": {}}\n" + fmt.Sprintf("{\"postlogs-data\":\"%s\",\"@timestamp\":\"%v\"}\n", data, getTimestamp())
	}
	return io.NopCloser(bytes.NewBuffer([]byte(body))), int64(len(body)), nil
}

func getTimestamp() interface{} {
	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d",
		time.Now().Year(),
		int(time.Now().Month()),
		time.Now().Day(),
		time.Now().Hour(),
		time.Now().Minute(),
		time.Now().Second(),
	)
}
