// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package postlogs

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

const (
	openSearchPostSuccessCountTotal     = "openSearch_post_success_count_total"
	openSearchPostSuccessCountTotalHelp = "The total number of successful openSearch POST requests"

	openSearchPostFailureCountTotal     = "openSearch_post_failure_count_total"
	openSearchPostFailureCountTotalHelp = "The total number of failed openSearch POST requests"

	openSearchPostSuccessLatencyNanoSeconds     = "openSearch_post_success_latency_nanoseconds"
	openSearchPostSuccessLatencyNanoSecondsHelp = "The latency of successful openSearch POST requests in nanoseconds"

	openSearchPostFailureLatencyNanoSeconds     = "openSearch_post_failure_latency_nanoseconds"
	openSearchPostFailureLatencyNanoSecondsHelp = "The latency of failed openSearch POST requests in nanoseconds"

	openSearchPostDataCharsTotal     = "openSearch_post_data_chars_total"
	openSearchPostDataCharsTotalHelp = "The total number of characters return from openSearch post request"
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
	constLabels := prometheus.Labels{}

	w := postLogs{workerMetrics: &workerMetrics{}}

	d := prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchPostSuccessCountTotal),
		openSearchPostSuccessCountTotalHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.workerMetrics.openSearchPostSuccessCountTotal.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchPostFailureCountTotal),
		openSearchPostFailureCountTotalHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.workerMetrics.openSearchPostFailureCountTotal.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchPostSuccessLatencyNanoSeconds),
		openSearchPostSuccessLatencyNanoSecondsHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.workerMetrics.openSearchPostSuccessLatencyNanoSeconds.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchPostFailureLatencyNanoSeconds),
		openSearchPostFailureLatencyNanoSecondsHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.workerMetrics.openSearchPostFailureLatencyNanoSeconds.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, openSearchPostDataCharsTotal),
		openSearchPostDataCharsTotalHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.workerMetrics.openSearchPostDataCharsTotal.Desc = d

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
	req := http.Request{
		URL: &url.URL{
			Scheme: "http",
			Host:   osIngestService,
			Path:   "/" + conf.DataStreamTemplate,
		},
		Header: http.Header{"Content-Type": {"application/json"}},
		Body:   postBody(),
	}
	startRequest := time.Now().UnixNano()
	resp, err := c.Do(&req)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("POST request to URI %s received a nil response", req.URL.RequestURI())
	}
	if resp.StatusCode == 200 {
		atomic.StoreInt64(&w.workerMetrics.openSearchPostSuccessLatencyNanoSeconds.Val, time.Now().UnixNano()-startRequest)
		atomic.AddInt64(&w.workerMetrics.openSearchPostSuccessCountTotal.Val, 1)
	} else {
		atomic.StoreInt64(&w.workerMetrics.openSearchPostFailureLatencyNanoSeconds.Val, time.Now().UnixNano()-startRequest)
		atomic.AddInt64(&w.workerMetrics.openSearchPostFailureCountTotal.Val, 1)
	}
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("Error reading response body: %v", err)
	}
	atomic.AddInt64(&w.workerMetrics.openSearchPostDataCharsTotal.Val, int64(len(respBody)))

	if resp.StatusCode == 200 {
		return nil
	}
	return fmt.Errorf("Failed, POST response error status code %v", resp.StatusCode)
}

func (w postLogs) PostMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w postLogs) PostMetricList() []prometheus.Metric {
	metrics := []prometheus.Metric{}

	m := prometheus.MustNewConstMetric(
		w.workerMetrics.openSearchPostSuccessCountTotal.Desc,
		prometheus.CounterValue,
		float64(atomic.LoadInt64(&w.workerMetrics.openSearchPostSuccessCountTotal.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.workerMetrics.openSearchPostFailureCountTotal.Desc,
		prometheus.CounterValue,
		float64(atomic.LoadInt64(&w.workerMetrics.openSearchPostFailureCountTotal.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.workerMetrics.openSearchPostSuccessLatencyNanoSeconds.Desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&w.workerMetrics.openSearchPostSuccessLatencyNanoSeconds.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.workerMetrics.openSearchPostFailureLatencyNanoSeconds.Desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&w.workerMetrics.openSearchPostFailureLatencyNanoSeconds.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.workerMetrics.openSearchPostDataCharsTotal.Desc,
		prometheus.CounterValue,
		float64(atomic.LoadInt64(&w.workerMetrics.openSearchPostDataCharsTotal.Val)))
	metrics = append(metrics, m)

	return metrics
}

func postBody() io.ReadCloser {
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
		PostRandomLowerAlpha(),
		PostRandomLowerAlpha(),
		PostRandomLowerAlpha(),
		PostRandomLowerAlpha(),
		PostRandomLowerAlpha(),
		PostRandomLowerAlpha(),
		PostRandomLowerAlpha(),
		PostRandomLowerAlpha(),
		PostRandomLowerAlpha(),
		PostRandomLowerAlpha(),
	)
	return io.NopCloser(bytes.NewBuffer([]byte(body)))
}

func PostRandomLowerAlpha() string {
	rand.Seed(time.Now().UnixNano())
	return string(letters[rand.Intn(len(letters))])
}
