// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package alerts

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/pkg/k8sclient"
	psrprom "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/prometheus"
	psrvz "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/verrazzano"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"io"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

// metricsPrefix is the prefix that is automatically pre-pended to all metrics exported by this worker.
const metricsPrefix = "http_get"

var httpGetFunc = http.Get
var funcNewPsrClient = k8sclient.NewPsrClient

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
	if err := updateVZForAlertmanager(log); err != nil {
		return err
	}

	http.HandleFunc("/alerts", func(w http.ResponseWriter, r *http.Request) {
		c, err := funcNewPsrClient()
		if err != nil {
			log.Errorf("error creating client: %v", err)
		}
		if r.Body == nil {
			log.Errorf("Alert webhook POST request contained a nil body")
			return
		}
		bodyRaw, err := io.ReadAll(r.Body)
		if err != nil {
			log.Errorf("Unexpected error while reading request body: %v", err)
			return
		}
		r.Body.Close()
		alertName, err := httputil.ExtractFieldFromResponseBodyOrReturnError(string(bodyRaw), "alerts.0.labels.alertname", "unable to extract alertname from body json")
		if err != nil {
			log.Error(err)
		}
		event := corev1.Event{
			ObjectMeta: v1.ObjectMeta{
				Name:      "psr-alert-" + alertName,
				Namespace: config.PsrEnv.GetEnv(config.PsrWorkerNamespace),
			},
			InvolvedObject: corev1.ObjectReference{
				Namespace: config.PsrEnv.GetEnv(config.PsrWorkerNamespace),
			},
			Type:    "Alert",
			Message: string(bodyRaw),
		}
		if _, err = controllerutil.CreateOrUpdate(context.TODO(), c.CrtlRuntime, &event, func() error {
			event.LastTimestamp = v1.Time{Time: time.Now()}
			event.Message = string(bodyRaw)
			return nil
		}); err != nil {
			log.Errorf("error generating alert event: %v", err)
		}
	})
	select {}
	return nil
}

func updateVZForAlertmanager(log vzlog.VerrazzanoLogger) error {
	if err := createAlertmanagerOverridesCM(log); err != nil {
		return err
	}

	c, err := funcNewPsrClient()
	if err != nil {
		log.Errorf("error creating client: %v", err)
	}
	cr, err := psrvz.GetVerrazzano(c.VzInstall)
	if err != nil {
		return err
	}
	var m psrprom.AlertmanagerConfigModifier
	m.ModifyCR(cr)
	return psrvz.UpdateVZCR(c, log, cr, m)
}

func createAlertmanagerOverridesCM(log vzlog.VerrazzanoLogger) error {
	c, err := funcNewPsrClient()
	if err != nil {
		log.Errorf("error creating client: %v", err)
	}
	cr, err := psrvz.GetVerrazzano(c.VzInstall)
	if err != nil {
		return err
	}
	cm := corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      psrprom.AlertmanagerCMName,
			Namespace: cr.Namespace,
		},
		Data: map[string]string{},
	}
	_, err = controllerutil.CreateOrUpdate(context.TODO(), c.CrtlRuntime, &cm, func() error {
		cm.Data = map[string]string{
			psrprom.AlertmanagerCMKey: `
{
	"alertmanager": {
		"alertmanagerSpec": {
			"podMetadata": {
				"annotations": {
					"sidecar.istio.io/inject": "false"
				}
			}
		},
		"config": {
			"receivers": [
				{
					"name": "webhook",
					"webhook_configs": [
						{
							"url": "http://psr-alerts-http-alerts.psr:9090/alerts"
						}
					]
				}
			],
			"route": {
				"group_by": [
					"alertname"
				],
				"receiver": "webhook"
			}
		},
		"enabled": true
	}
}`,
		}
		return nil
	})
	return err
}
