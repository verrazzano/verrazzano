package metricsexporter

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func InitalizeMetricsEndpoint() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9100", nil)

}
