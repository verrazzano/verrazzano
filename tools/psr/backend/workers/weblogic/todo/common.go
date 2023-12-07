// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"net/http"
	"sync/atomic"
)

type httpFunc func(url string) (resp *http.Response, err error)

var httpGetFunc httpFunc = http.Get

const (
	// metricsPrefix is the prefix that is automatically pre-pended to all metrics exported by this worker.
	metricsPrefix = "http_get"

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

type HttpMetricDef struct {
	RequestsCountTotal          metrics.MetricItem
	RequestsSucceededCountTotal metrics.MetricItem
	RequestsFailedCountTotal    metrics.MetricItem
	RequestDurationMillis       metrics.MetricItem
}

// HandleResponse processes the HTTP response and updates metrics
func HandleResponse(log vzlog.VerrazzanoLogger, URL string, metricDef *HttpMetricDef, resp *http.Response, err error) error {
	if err != nil {
		atomic.AddInt64(&metricDef.RequestsFailedCountTotal.Val, 1)
		return log.ErrorfNewErr("HTTP request %s returned error %v", URL, err)
	}
	if resp == nil {
		atomic.AddInt64(&metricDef.RequestsFailedCountTotal.Val, 1)
		return log.ErrorfNewErr("HTTP request %s returned nil response", URL)
	}
	resp.Body.Read()

	if resp.StatusCode != 200 {
		atomic.AddInt64(&metricDef.RequestsFailedCountTotal.Val, 1)
		return log.ErrorfNewErr("HTTP request %s returned StatusCode &v", URL, resp.StatusCode)
	}
	// Success
	atomic.AddInt64(&metricDef.RequestsSucceededCountTotal.Val, 1)
	log.Progressf("Http request to URL %s succeeded", URL)
	return nil
}
