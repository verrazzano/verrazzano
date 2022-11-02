// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package getlogs

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
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
	openSearchGetSuccessCountTotal     = "openSearch_get_success_count_total"
	openSearchGetSuccessCountTotalHelp = "The total number of successful openSearch GET requests"

	openSearchGetFailureCountTotal     = "openSearch_get_failure_count_total"
	openSearchGetFailureCountTotalHelp = "The total number of failed openSearch GET requests"

	openSearchGetSuccessLatencyNanoSeconds     = "openSearch_get_success_latency_nanoseconds"
	openSearchGetSuccessLatencyNanoSecondsHelp = "The latency of successful openSearch GET requests in nanoseconds"

	openSearchGetFailureLatencyNanoSeconds     = "openSearch_get_failure_latency_nanoseconds"
	openSearchGetFailureLatencyNanoSecondsHelp = "The latency of failed openSearch GET requests in nanoseconds"

	openSearchGetDataCharsTotal     = "openSearch_get_data_chars_total"
	openSearchGetDataCharsTotalHelp = "The total number of characters return from openSearch get request"
)

const osIngestService = "vmi-system-es-ingest.verrazzano-system:9200"

const letters = "abcdefghijklmnopqrstuvwxyz"

type getLogs struct {
	spi.Worker
	metricDescList []prometheus.Desc
	*workerMetrics
}

var _ spi.Worker = getLogs{}

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	openSearchGetSuccessCountTotal         metrics.MetricItem
	openSearchGetFailureCountTotal         metrics.MetricItem
	openSearchGetSuccessLatencyNanoSeconds metrics.MetricItem
	openSearchGetFailureLatencyNanoSeconds metrics.MetricItem
	openSearchGetDataCharsTotal            metrics.MetricItem
}

func NewGetLogsWorker() (spi.Worker, error) {
	constLabels := prometheus.Labels{}

	w := getLogs{workerMetrics: &workerMetrics{}}

	d := prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetSuccessCountTotal),
		openSearchGetSuccessCountTotalHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.workerMetrics.openSearchGetSuccessCountTotal.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetFailureCountTotal),
		openSearchGetFailureCountTotalHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.workerMetrics.openSearchGetFailureCountTotal.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetSuccessLatencyNanoSeconds),
		openSearchGetSuccessLatencyNanoSecondsHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.workerMetrics.openSearchGetSuccessLatencyNanoSeconds.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetFailureLatencyNanoSeconds),
		openSearchGetFailureLatencyNanoSecondsHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.workerMetrics.openSearchGetFailureLatencyNanoSeconds.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchGetDataCharsTotal),
		openSearchGetDataCharsTotalHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.workerMetrics.openSearchGetDataCharsTotal.Desc = d

	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w getLogs) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		EnvName:     config.WorkerTypeGetLogs,
		Description: "The log getter worker performs GET requests on the OpenSearch endpoint",
		MetricsName: "getlogs",
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
			Path:   "/_search",
		},
		Header: http.Header{"Content-Type": {"application/json"}},
		Body:   getBody(),
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
		atomic.StoreInt64(&w.workerMetrics.openSearchGetSuccessLatencyNanoSeconds.Val, time.Now().UnixNano()-startRequest)
		atomic.AddInt64(&w.workerMetrics.openSearchGetSuccessCountTotal.Val, 1)
	} else {
		atomic.StoreInt64(&w.workerMetrics.openSearchGetFailureLatencyNanoSeconds.Val, time.Now().UnixNano()-startRequest)
		atomic.AddInt64(&w.workerMetrics.openSearchGetFailureCountTotal.Val, 1)
	}
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("Error reading response body: %v", err)
	}
	atomic.AddInt64(&w.workerMetrics.openSearchGetDataCharsTotal.Val, int64(len(respBody)))

	if resp.StatusCode == 200 {
		return nil
	}
	return fmt.Errorf("Failed, GET response error status code %v", resp.StatusCode)
}

func (w getLogs) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w getLogs) GetMetricList() []prometheus.Metric {
	metrics := []prometheus.Metric{}

	m := prometheus.MustNewConstMetric(
		w.workerMetrics.openSearchGetSuccessCountTotal.Desc,
		prometheus.CounterValue,
		float64(atomic.LoadInt64(&w.workerMetrics.openSearchGetSuccessCountTotal.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.workerMetrics.openSearchGetFailureCountTotal.Desc,
		prometheus.CounterValue,
		float64(atomic.LoadInt64(&w.workerMetrics.openSearchGetFailureCountTotal.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.workerMetrics.openSearchGetSuccessLatencyNanoSeconds.Desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&w.workerMetrics.openSearchGetSuccessLatencyNanoSeconds.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.workerMetrics.openSearchGetFailureLatencyNanoSeconds.Desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&w.workerMetrics.openSearchGetFailureLatencyNanoSeconds.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.workerMetrics.openSearchGetDataCharsTotal.Desc,
		prometheus.CounterValue,
		float64(atomic.LoadInt64(&w.workerMetrics.openSearchGetDataCharsTotal.Val)))
	metrics = append(metrics, m)

	return metrics
}

func getBody() io.ReadCloser {
	body := fmt.Sprintf(`{
  "query": {
    "bool": {
      "should": [
        {
          "match": {
            "message": "%s"
          }
        },
        {
          "match": {
            "message": "%s"
          }
        },
        {
          "match": {
            "message": "%s"
          }
        },
        {
          "match": {
            "message": "%s"
          }
        },
                {
          "match": {
            "message": "%s"
          }
        },
        {
          "match": {
            "message": "%s"
          }
        },
        {
          "match": {
            "message": "%s"
          }
        },
        {
          "match": {
            "message": "%s"
          }
        },
        {
          "match": {
            "message": "%s"
          }
        },
                {
          "match": {
            "message": "%s"
          }
        }
      ]
    }
  }
}`,
		GetRandomLowerAlpha(),
		GetRandomLowerAlpha(),
		GetRandomLowerAlpha(),
		GetRandomLowerAlpha(),
		GetRandomLowerAlpha(),
		GetRandomLowerAlpha(),
		GetRandomLowerAlpha(),
		GetRandomLowerAlpha(),
		GetRandomLowerAlpha(),
		GetRandomLowerAlpha(),
	)
	return io.NopCloser(bytes.NewBuffer([]byte(body)))
}

func GetRandomLowerAlpha() string {
	rand.Seed(time.Now().UnixNano())
	return string(letters[rand.Intn(len(letters))]) //nolint:gosec //#gosec G404
}
