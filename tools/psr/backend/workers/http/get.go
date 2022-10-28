// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package http

import (
   "net/http"
   "os"
)

import (
    "fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"sync/atomic"
)

const (
	totalGetRequestsCount     = "total_get_requests"
	totalGetRequestsCountHelp = "The total number of get requests logged"

	totalGetRequestsSucceededCount     = "total_get_requests_succeeded"
	totalGetRequestsSucceededCountHelp = "The total number of get requests that are successful"

	totalGetRequestsFailedCount     = "total_get_requests_failed"
	totalGetRequestsFailedCountHelp = "The total number of get requests that have failed"

)

type httpGetWorker struct {
	spi.Worker
	metricDescList []prometheus.Desc
	*httpgetMetrics
}

var _ spi.Worker = httpGetWorker{}

// httpgetMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type httpgetMetrics struct {
	totalGetRequestsCount metrics.MetricItem
	totalGetRequestsSucceededCount metrics.MetricItem
    totalGetRequestsFailedCount metrics.MetricItem
}

func NewHttpGetWorker() (spi.Worker, error) {
    constLabels := prometheus.Labels{}

    w := httpGetWorker{httpgetMetrics: &httpgetMetrics{}}

    d := prometheus.NewDesc(
        prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, totalGetRequestsCount),
        totalGetRequestsCountHelp,
        nil,
        constLabels,
    )
    w.metricDescList = append(w.metricDescList, *d)
    w.httpgetMetrics.totalGetRequestsCount.Desc = d

    d = prometheus.NewDesc(
        prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, totalGetRequestsSucceededCount),
        totalGetRequestsSucceededCountHelp,
        nil,
        constLabels,
    )
    w.metricDescList = append(w.metricDescList, *d)
    w.httpgetMetrics.totalGetRequestsSucceededCount.Desc = d

    d = prometheus.NewDesc(
        prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, totalGetRequestsFailedCount),
        totalGetRequestsFailedCountHelp,
        nil,
        constLabels,
    )
    w.metricDescList = append(w.metricDescList, *d)
    w.httpgetMetrics.totalGetRequestsFailedCount.Desc = d

    return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w httpGetWorker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		EnvName:     config.WorkerTypeHttpGet,
		Description: "Http get worker to make a GET request on the given endpoint",
		MetricsName: "httpget",
	}
}

func (w httpGetWorker) GetEnvDescList() []config.EnvVarDesc {
	return []config.EnvVarDesc{}
}

func (w httpGetWorker) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w httpGetWorker) GetMetricList() []prometheus.Metric {
    metrics := []prometheus.Metric{}

    m := prometheus.MustNewConstMetric(
        w.httpgetMetrics.totalGetRequestsCount.Desc,
        prometheus.GaugeValue,
        float64(atomic.LoadInt64(&w.httpgetMetrics.totalGetRequestsCount.Val)))
    metrics = append(metrics, m)

    m = prometheus.MustNewConstMetric(
        w.httpgetMetrics.totalGetRequestsSucceededCount.Desc,
        prometheus.GaugeValue,
        float64(atomic.LoadInt64(&w.httpgetMetrics.totalGetRequestsSucceededCount.Val)))
    metrics = append(metrics, m)

    m = prometheus.MustNewConstMetric(
        w.httpgetMetrics.totalGetRequestsFailedCount.Desc,
        prometheus.GaugeValue,
        float64(atomic.LoadInt64(&w.httpgetMetrics.totalGetRequestsFailedCount.Val)))
    metrics = append(metrics, m)

    return metrics
}

func (w httpGetWorker) WantIterationInfoLogged() bool {
	return false
}

func (w httpGetWorker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	var lc, ls, lf int64
	log.Infof("HttpGet Worker doing work")
    var u string = os.Getenv("HTTP_GET_ENDPOINT")
    log.Infof("Endpoint %s", u)
    resp, err := http.Get(u)
    if err != nil {
        lf = atomic.AddInt64(&w.httpgetMetrics.totalGetRequestsFailedCount.Val, 1)
        return err
    } else {
        fmt.Println("The status code of the get request is: ", resp.StatusCode)
        // log.Infof(logMsg)
        ls = atomic.AddInt64(&w.httpgetMetrics.totalGetRequestsSucceededCount.Val, 1)
    }
    lc = atomic.AddInt64(&w.httpgetMetrics.totalGetRequestsCount.Val, 1)
    logMsg := fmt.Sprintf("HttpGet worker total requests %v, " +
                " successful requests %v, failed requests %v",
                lc, ls,lf)
    log.Infof(logMsg)
    return nil
}