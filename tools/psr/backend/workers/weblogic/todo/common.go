// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"io"
	"net/http"
)

type HTTPMetricDef struct {
	RequestsCountTotal          metrics.MetricItem
	RequestsSucceededCountTotal metrics.MetricItem
	RequestsFailedCountTotal    metrics.MetricItem
	RequestDurationMicros       metrics.MetricItem
}

// HandleResponse processes the HTTP response and updates metrics
func HandleResponse(log vzlog.VerrazzanoLogger, URL string, resp *http.Response, err error, require200 bool) ([]byte, error) {
	if err != nil {
		return nil, log.ErrorfNewErr("HTTP request %s returned error %v", URL, err)
	}
	if resp == nil {
		return nil, log.ErrorfNewErr("HTTP request %s returned nil response", URL)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, log.ErrorfNewErr("HTTP request body ReadAll for URL %s returned error %v", URL, err)
	}

	if require200 {
		if resp.StatusCode != 200 {
			return nil, log.ErrorfNewErr("HTTP request %s returned StatusCode &v", URL, resp.StatusCode)
		}
	} else {
		if resp.StatusCode > 300 {
			return nil, log.ErrorfNewErr("HTTP request %s returned StatusCode &v", URL, resp.StatusCode)
		}
	}

	return body, nil
}
