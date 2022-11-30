// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package alerts

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/pkg/k8sclient"
	psrprom "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/prometheus"
	psrvz "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/verrazzano"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// metricsPrefix is the prefix that is automatically pre-pended to all metrics exported by this worker.
const metricsPrefix = "prom_alerts"

var funcNewPsrClient = k8sclient.NewPsrClient

type worker struct {
	metricDescList []prometheus.Desc
	*workerMetrics
}

var _ spi.Worker = worker{}

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	alertsFiringCount   metrics.MetricItem
	alertsResolvedCount metrics.MetricItem
}

func NewAlertsWorker() (spi.Worker, error) {
	w := worker{workerMetrics: &workerMetrics{
		alertsFiringCount: metrics.MetricItem{
			Name: "alerts_firing_received_count",
			Help: "The total number of alerts received from alertmanager",
			Type: prometheus.CounterValue,
		},
		alertsResolvedCount: metrics.MetricItem{
			Name: "alerts_resolved_received_count",
			Help: "The total number of alerts received from alertmanager",
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
		&w.alertsFiringCount,
		&w.alertsResolvedCount,
	}, metricsLabels, w.GetWorkerDesc().MetricsPrefix)
	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w worker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		WorkerType:    config.WorkerTypeHTTPGet,
		Description:   "The alerts receiver worker configures alertmanger and receives alerts and writes them to events",
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
		w.alertsFiringCount.BuildMetric(),
		w.alertsResolvedCount.BuildMetric(),
	}
}

func (w worker) WantLoopInfoLogged() bool {
	return false
}

func (w worker) PreconditionsMet() (bool, error) {
	return true, nil
}

func (w worker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	if err := updateVZForAlertmanager(log, conf); err != nil {
		return err
	}

	http.HandleFunc("/alerts", func(rw http.ResponseWriter, r *http.Request) {
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
		alertStatus, err := httputil.ExtractFieldFromResponseBodyOrReturnError(string(bodyRaw), "status", "unable to extract alert status from body json")
		if err != nil {
			log.Error(err)
		}
		if alertStatus == "firing" {
			atomic.AddInt64(&w.workerMetrics.alertsFiringCount.Val, 1)
		} else if alertStatus == "resolved" {
			atomic.AddInt64(&w.workerMetrics.alertsResolvedCount.Val, 1)
		} else {
			log.Errorf("alert received with unknown status: %s", alertStatus)
		}

		event := corev1.Event{
			ObjectMeta: v1.ObjectMeta{
				Name:      "psr-alert-" + alertName,
				Namespace: config.PsrEnv.GetEnv(config.PsrWorkerNamespace),
			},
			InvolvedObject: corev1.ObjectReference{
				Namespace: config.PsrEnv.GetEnv(config.PsrWorkerNamespace),
			},
			Type: "Alert",
		}
		if _, err = controllerutil.CreateOrUpdate(context.TODO(), c.CrtlRuntime, &event, func() error {
			event.LastTimestamp = v1.Time{Time: time.Now()}
			event.Message = string(bodyRaw)
			event.Reason = alertStatus
			return nil
		}); err != nil {
			log.Errorf("error generating alert event: %v", err)
		}
	})
	select {}
	return nil
}

func updateVZForAlertmanager(log vzlog.VerrazzanoLogger, conf config.CommonConfig) error {
	if err := createAlertmanagerOverridesCM(log, conf); err != nil {
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

	return psrvz.UpdateVerrazzano(c.VzInstall, cr)
}

func createAlertmanagerOverridesCM(log vzlog.VerrazzanoLogger, conf config.CommonConfig) error {
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
			psrprom.AlertmanagerCMKey: fmt.Sprintf(`alertmanager:
  alertmanagerSpec:
    podMetadata:
      annotations:
        sidecar.istio.io/inject: "false"
  config:
    receivers:
    - webhook_configs:
      - url: http://%s-%s.%s:9090/alerts
      name: webhook
    route:
      group_by:
      - alertname
      receiver: webhook
      routes:
      - match:
          alertname: Watchdog
        receiver: webhook
  enabled: true
`, conf.ReleaseName, conf.WorkerType, conf.Namespace,
			)}
		return nil
	})
	return err
}
