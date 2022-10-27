// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package getlogs

import (
	"bytes"
	"fmt"
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
)

const (
	openSearchGetSuccessCount     = "openSearch_get_success_count"
	openSearchGetSuccessCountHelp = "The total number of successful openSearch GET requests"

	openSearchGetFailureCount     = "openSearch_get_failure_count"
	openSearchGetFailureCountHelp = "The total number of failed openSearch GET requests"

	openSearchGetSuccessLatencyNanoSeconds     = "openSearch_get_success_latency_nanoseconds"
	openSearchGetSuccessLatencyNanoSecondsHelp = "The latency of sucessful openSearch GET requests in nanoseconds"

	openSearchGetFailureLatencyNanoSeconds     = "openSearch_get_failure_latency_nanoseconds"
	openSearchGetFailureLatencyNanoSecondsHelp = "The latency of failed openSearch GET requests in nanoseconds"

	openSearchGetDataTotalChars     = "openSearch_get_data_total_chars"
	openSearchGetDataTotalCharsHelp = "The total number of characters return from openSearch get request"
)

const osIngestService = "vmi-system-es-ingest.verrazzano-system:9200"

var bodyString = "{\"query\":{\"bool\":{\"filter\":[{\"match_phrase\":{\"kubernetes.container_name\":\"istio-proxy\"}}]}}}"
var body = io.NopCloser(bytes.NewBuffer([]byte(bodyString)))

type getLogs struct {
	spi.Worker
	metricDescList []prometheus.Desc
	*getLogsMetrics
}

var _ spi.Worker = getLogs{}

// getLogsMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type getLogsMetrics struct {
	openSearchGetSuccessCount              metrics.MetricItem
	openSearchGetFailureCount              metrics.MetricItem
	openSearchGetSuccessLatencyNanoSeconds metrics.MetricItem
	openSearchGetFailureLatencyNanoSeconds metrics.MetricItem
	openSearchGetDataTotalChars            metrics.MetricItem
}

func NewGetLogs() (spi.Worker, error) {
	constLabels := prometheus.Labels{}

	w := getLogs{getLogsMetrics: &getLogsMetrics{}}

	d := prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetSuccessCount),
		openSearchGetSuccessCountHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.getLogsMetrics.openSearchGetSuccessCount.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetFailureCount),
		openSearchGetFailureCountHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.getLogsMetrics.openSearchGetFailureCount.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetSuccessLatencyNanoSeconds),
		openSearchGetSuccessLatencyNanoSecondsHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.getLogsMetrics.openSearchGetSuccessLatencyNanoSeconds.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetFailureLatencyNanoSeconds),
		openSearchGetFailureLatencyNanoSecondsHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.getLogsMetrics.openSearchGetFailureLatencyNanoSeconds.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetDataTotalChars),
		openSearchGetDataTotalCharsHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.getLogsMetrics.openSearchGetDataTotalChars.Desc = d

	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w getLogs) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		EnvName:     config.WorkerTypeGetLogs,
		Description: "The log getter worker performs GET requests on the OpenSearch endpoint",
		MetricsName: "GetLogs",
	}
}

func (w getLogs) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{}
}

func (w getLogs) WantIterationInfoLogged() bool {
	return false
}

func (w getLogs) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	c := http.Client{}
	req := http.Request{
		URL: &url.URL{
			Scheme: "http",
			Host:   osIngestService,
			Path:   "/verrazzano-system",
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
		return fmt.Errorf("GET request to URI %s received a nil response", req.URL.RequestURI())
	}
	if resp.StatusCode == 200 {
		atomic.StoreInt64(&w.getLogsMetrics.openSearchGetSuccessLatencyNanoSeconds.Val, time.Now().UnixNano()-startRequest)
		atomic.AddInt64(&w.getLogsMetrics.openSearchGetSuccessCount.Val, 1)
	} else {
		atomic.StoreInt64(&w.getLogsMetrics.openSearchGetFailureLatencyNanoSeconds.Val, time.Now().UnixNano()-startRequest)
		atomic.AddInt64(&w.getLogsMetrics.openSearchGetFailureCount.Val, 1)
	}
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("Error reading response body: %v", err)
	}
	atomic.AddInt64(&w.getLogsMetrics.openSearchGetDataTotalChars.Val, int64(len(respBody)))
	return nil
}

func (w getLogs) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w getLogs) GetMetricList() []prometheus.Metric {
	metrics := []prometheus.Metric{}

	m := prometheus.MustNewConstMetric(
		w.getLogsMetrics.openSearchGetSuccessCount.Desc,
		prometheus.CounterValue,
		float64(atomic.LoadInt64(&w.getLogsMetrics.openSearchGetSuccessCount.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.getLogsMetrics.openSearchGetFailureCount.Desc,
		prometheus.CounterValue,
		float64(atomic.LoadInt64(&w.getLogsMetrics.openSearchGetFailureCount.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.getLogsMetrics.openSearchGetSuccessLatencyNanoSeconds.Desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&w.getLogsMetrics.openSearchGetSuccessLatencyNanoSeconds.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.getLogsMetrics.openSearchGetFailureLatencyNanoSeconds.Desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&w.getLogsMetrics.openSearchGetFailureLatencyNanoSeconds.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.getLogsMetrics.openSearchGetDataTotalChars.Desc,
		prometheus.CounterValue,
		float64(atomic.LoadInt64(&w.getLogsMetrics.openSearchGetDataTotalChars.Val)))
	metrics = append(metrics, m)

	return metrics
}
