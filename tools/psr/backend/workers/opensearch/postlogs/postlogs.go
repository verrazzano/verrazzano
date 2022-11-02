// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package postlogs

import (
	"bytes"
	"fmt"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/opensearch/getlogs"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"

	"github.com/prometheus/client_golang/prometheus"
	"moul.io/http2curl"
)

const osIngestService = "vmi-system-es-ingest.verrazzano-system:9200"

const letters = "abcdefghijklmnopqrstuvwxyz"

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
		Description: "The log postter worker performs POST requests on the OpenSearch endpoint",
		MetricsName: "postlogs",
	}
}

func (w postLogs) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{}
}

func (w postLogs) WantIterationInfoLogged() bool {
	return false
}

func (w postLogs) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	c := http.Client{}
	body, bodyChars := getBody()
	req := http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "http",
			Host:   osIngestService,
			Path:   "/verrazzano-application-" + conf.Namespace + "/_doc",
		},
		Header: http.Header{"Content-Type": {"application/json"}},
		Body:   body,
	}
	cmd, _ := http2curl.GetCurlCommand(&req)
	log.Info(cmd.String())
	startRequest := time.Now().UnixNano()
	resp, err := c.Do(&req)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("POST request to URI %s received a nil response", req.URL.RequestURI())
	}
	log.Infof("STATUS: ", resp.Status)
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Infof("BODY: %v", string(b))
	if resp.StatusCode != 201 {
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

func getBody() (io.ReadCloser, int64) {
	body := fmt.Sprintf(`{"field1": "%s", "field2": "%s", "field3": "%s", "field4": "%s", "@timestamp": "%v"}`,
		append(getlogs.GetRandomLowerAlpha(4), getTimestamp())...)
	return io.NopCloser(bytes.NewBuffer([]byte(body))), int64(len(body))
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
