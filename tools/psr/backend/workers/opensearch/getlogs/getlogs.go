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

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"

	"github.com/prometheus/client_golang/prometheus"
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
	w := getLogs{workerMetrics: &workerMetrics{
		openSearchGetSuccessCountTotal: metrics.MetricItem{
			Name: "opensearch_get_success_count_total",
			Help: "The total number of successful OpenSearch GET requests",
			Type: prometheus.CounterValue,
		},
		openSearchGetFailureCountTotal: metrics.MetricItem{
			Name: "opensearch_get_failure_count_total",
			Help: "The total number of successful OpenSearch GET requests",
			Type: prometheus.CounterValue,
		},
		openSearchGetSuccessLatencyNanoSeconds: metrics.MetricItem{
			Name: "opensearch_get_success_latency_nanoseconds",
			Help: "The latency of successful OpenSearch GET requests in nanoseconds",
			Type: prometheus.GaugeValue,
		},
		openSearchGetFailureLatencyNanoSeconds: metrics.MetricItem{
			Name: "opensearch_get_failure_latency_nanoseconds",
			Help: "The latency of failed OpenSearch GET requests in nanoseconds",
			Type: prometheus.GaugeValue,
		},
		openSearchGetDataCharsTotal: metrics.MetricItem{
			Name: "opensearch_get_data_chars_total",
			Help: "The total number of characters return from OpenSearch get request",
			Type: prometheus.CounterValue,
		},
	}}

	w.metricDescList = []prometheus.Desc{
		*w.openSearchGetSuccessCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.openSearchGetFailureCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.openSearchGetSuccessLatencyNanoSeconds.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.openSearchGetFailureLatencyNanoSeconds.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.openSearchGetDataCharsTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
	}
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

func (w getLogs) WantLoopInfoLogged() bool {
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
		return fmt.Errorf("OpenSearch GET request failed, returned %v status code", resp.StatusCode)
	}
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("Error reading response body: %v", err)
	}
	atomic.AddInt64(&w.workerMetrics.openSearchGetDataCharsTotal.Val, int64(len(respBody)))
	return nil
}

func (w getLogs) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w getLogs) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.openSearchGetSuccessCountTotal.BuildMetric(),
		w.openSearchGetFailureCountTotal.BuildMetric(),
		w.openSearchGetSuccessLatencyNanoSeconds.BuildMetric(),
		w.openSearchGetFailureLatencyNanoSeconds.BuildMetric(),
		w.openSearchGetDataCharsTotal.BuildMetric(),
	}
}

func getBody() io.ReadCloser {
	body := fmt.Sprintf(`
{
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
}`, GetRandomLowerAlpha(10)...)
	return io.NopCloser(bytes.NewBuffer([]byte(body)))
}

// GetRandomLowerAlpha returns an array of len n of random lowercase letters
func GetRandomLowerAlpha(n int) []interface{} {
	var str []interface{}
	for i := 0; i < n; i++ {
		str = append(str, string(letters[rand.Intn(len(letters))])) //nolint:gosec //#gosec G404
	}
	return str
}
