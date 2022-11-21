// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package alerts

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/pkg/k8sclient"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

var httpGetFunc = http.Get
var funcNewPsrClient = k8sclient.NewPsrClient

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
	Path = "PATH"
)

type worker struct {
	metricDescList []prometheus.Desc
	*workerMetrics
}

var _ spi.Worker = worker{}

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	getRequestsCountTotal          metrics.MetricItem
	getRequestsSucceededCountTotal metrics.MetricItem
	getRequestsFailedCountTotal    metrics.MetricItem
}

func NewReceiveAlertsWorker() (spi.Worker, error) {
	w := worker{workerMetrics: &workerMetrics{
		getRequestsCountTotal: metrics.MetricItem{
			Name: "request_count_total",
			Help: "The total number of GET requests",
			Type: prometheus.CounterValue,
		},
		getRequestsSucceededCountTotal: metrics.MetricItem{
			Name: "request_succeeded_count_total",
			Help: "The total number of successful GET requests",
			Type: prometheus.CounterValue,
		},
		getRequestsFailedCountTotal: metrics.MetricItem{
			Name: "request_failed_count_total",
			Help: "The total number of failed GET requests",
			Type: prometheus.CounterValue,
		},
	}}

	if err := config.PsrEnv.LoadFromEnv(w.GetEnvDescList()); err != nil {
		return w, err
	}

	metricsLabels := map[string]string{
		config.PsrWorkerTypeMetricsName: config.PsrEnv.GetEnv(config.PsrWorkerType),
	}

	w.metricDescList = metrics.BuildMetricDescList([]*metrics.MetricItem{
		&w.getRequestsCountTotal,
		&w.getRequestsSucceededCountTotal,
		&w.getRequestsFailedCountTotal,
	}, metricsLabels, w.GetWorkerDesc().MetricsPrefix)
	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w worker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		WorkerType:    config.WorkerTypeHTTPGet,
		Description:   "The get worker makes GET request on the given endpoint",
		MetricsPrefix: metricsPrefix,
	}
}

func (w worker) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{}
}

func (w worker) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w worker) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.getRequestsCountTotal.BuildMetric(),
		w.getRequestsSucceededCountTotal.BuildMetric(),
		w.getRequestsFailedCountTotal.BuildMetric(),
	}
}

func (w worker) WantLoopInfoLogged() bool {
	return false
}

func (w worker) PreconditionsMet() (bool, error) {
	return true, nil
}

func (w worker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Infof("Received POST")
		c, err := funcNewPsrClient()
		if err != nil {
			log.Errorf("error creating client: %v", err)
		}
		event := corev1.Event{
			ObjectMeta: v1.ObjectMeta{
				Namespace: config.PsrEnv.GetEnv(config.PsrWorkerNamespace),
			},
			Type:    "Alert",
			Message: "Alert received",
		}
		if err = c.CrtlRuntime.Create(context.TODO(), &event); err != nil {
			log.Errorf("error generating event: %v", err)
		}
	})
	select {}
	return nil
}
