// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func InitalizeMetricsEndpoint() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9100", nil)

}
