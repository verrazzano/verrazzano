// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package logget

import (
	"bytes"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

const (
	openSearchGetSuccess     = "openSearch_get_success_count"
	openSearchGetSuccessHelp = "The total number of sucessful openSearch GET requests"

	openSearchGetFailure     = "openSearch_get_failure_count"
	openSearchGetFailureHelp = "The total number of failed openSearch GET requests"

	openSearchGetSuccessNanoSeconds     = "openSearch_get_success_latency_nanoseconds"
	openSearchGetSuccessNanoSecondsHelp = "The latency of sucessful openSearch GET requests in nanoseconds"

	openSearchGetFailureNanoSeconds     = "openSearch_get_failure_latency_nanoseconds"
	openSearchGetFailureNanoSecondsHelp = "The latency of failed openSearch GET requests in nanoseconds"

	openSearchGetDataTotalChars     = "openSearch_get_data_total_chars"
	openSearchGetDataTotalCharsHelp = "The total number of characters return from openSearch get request"
)

const osIngestService = "vmi-system-es-ingest.verrazzano-system:9200"

var bodyString = "{\"query\":{\"bool\":{\"filter\":[{\"match_phrase\":{\"kubernetes.container_name\":\"istio-proxy\"}}]}}}"
var body = io.NopCloser(bytes.NewBuffer([]byte(bodyString)))

type logGetter struct {
	spi.Worker
	metricDescList []prometheus.Desc
	*loggetMetrics
}

var _ spi.Worker = logGetter{}

// loggetMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type loggetMetrics struct {
	openSearchGetSuccess            metrics.MetricItem
	openSearchGetFailure            metrics.MetricItem
	openSearchGetSuccessNanoSeconds metrics.MetricItem
	openSearchGetFailureNanoSeconds metrics.MetricItem
	openSearchGetDataTotalChars     metrics.MetricItem
}

func NewLogGetter() (spi.Worker, error) {
	constLabels := prometheus.Labels{}

	w := logGetter{loggetMetrics: &loggetMetrics{}}

	d := prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetSuccess),
		openSearchGetSuccessHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.loggetMetrics.openSearchGetSuccess.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetFailure),
		openSearchGetFailureHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.loggetMetrics.openSearchGetFailure.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetSuccessNanoSeconds),
		openSearchGetSuccessNanoSecondsHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.loggetMetrics.openSearchGetSuccessNanoSeconds.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetFailureNanoSeconds),
		openSearchGetFailureNanoSecondsHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.loggetMetrics.openSearchGetFailureNanoSeconds.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetDataTotalChars),
		openSearchGetDataTotalCharsHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.loggetMetrics.openSearchGetDataTotalChars.Desc = d

	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w logGetter) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		EnvName:     config.WorkerTypeLogGet,
		Description: "The log getter worker performs GET requests on the OpenSearch endpoint",
		MetricsName: "LogGet",
	}
}

func (w logGetter) GetEnvDescList() []config.EnvVarDesc {
	return []config.EnvVarDesc{}
}

func (w logGetter) WantIterationInfoLogged() bool {
	return false
}

func (w logGetter) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
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
		atomic.StoreInt64(&w.loggetMetrics.openSearchGetSuccessNanoSeconds.Val, time.Now().UnixNano()-startRequest)
		atomic.AddInt64(&w.loggetMetrics.openSearchGetSuccess.Val, 1)
	} else {
		atomic.StoreInt64(&w.loggetMetrics.openSearchGetFailureNanoSeconds.Val, time.Now().UnixNano()-startRequest)
		atomic.AddInt64(&w.loggetMetrics.openSearchGetFailure.Val, 1)
	}
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("Error reading response body: %v", err)
	}
	atomic.AddInt64(&w.loggetMetrics.openSearchGetDataTotalChars.Val, int64(len(respBody)))
	return nil
}

func (w logGetter) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w logGetter) GetMetricList() []prometheus.Metric {
	metrics := []prometheus.Metric{}

	m := prometheus.MustNewConstMetric(
		w.loggetMetrics.openSearchGetSuccess.Desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&w.loggetMetrics.openSearchGetSuccess.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.loggetMetrics.openSearchGetFailure.Desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&w.loggetMetrics.openSearchGetFailure.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.loggetMetrics.openSearchGetSuccessNanoSeconds.Desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&w.loggetMetrics.openSearchGetSuccessNanoSeconds.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.loggetMetrics.openSearchGetFailureNanoSeconds.Desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&w.loggetMetrics.openSearchGetFailureNanoSeconds.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.loggetMetrics.openSearchGetDataTotalChars.Desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&w.loggetMetrics.openSearchGetDataTotalChars.Val)))
	metrics = append(metrics, m)

	return metrics
}
