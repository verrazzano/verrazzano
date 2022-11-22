// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scale

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/pkg/weblogic"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"k8s.io/client-go/dynamic"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	// DomainUID specifies the name of the domain in the local cluster
	// By default, the DomainUID is not specified
	DomainUID = "DOMAIN_UID"

	// DomainNamespace specifies the namespace of the service in the local cluster
	// By default, the DomainNamespace is not specified
	DomainNamespace = "DOMAIN_NAMESPACE"

	// MinReplicaCount specifies the minimum replicas to scale on the domain
	// By default, MinReplicaCount is set 2
	MinReplicaCount = "MIN_REPLICA_COUNT"

	// MaxReplicaCount specifies the maximum replicas to scale on the domain
	// By default, MaxReplicaCount is set to 4
	MaxReplicaCount = "MAX_REPLICA_COUNT"

	metricsPrefix = "weblogic_scaling"
)

type worker struct {
	metricDescList []prometheus.Desc
	*workerMetrics
}

var _ spi.Worker = worker{}

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	scaleDomainCountTotal          metrics.MetricItem
	scaleDomainSucceededCountTotal metrics.MetricItem
	scaleDomainFailedCountTotal    metrics.MetricItem
}

func NewScaleWorker() (spi.Worker, error) {
	w := worker{workerMetrics: &workerMetrics{
		scaleDomainCountTotal: metrics.MetricItem{
			Name: "scale_domain_count_total",
			Help: "The total number of scale domain requests",
			Type: prometheus.CounterValue,
		},
		scaleDomainSucceededCountTotal: metrics.MetricItem{
			Name: "scale_domain_succeeded_count_total",
			Help: "The total number of successful scale domain requests",
			Type: prometheus.CounterValue,
		},
		scaleDomainFailedCountTotal: metrics.MetricItem{
			Name: "scale_domain_failed_count_total",
			Help: "The total number of failed scale domain requests",
			Type: prometheus.CounterValue,
		},
	}}

	w.metricDescList = []prometheus.Desc{
		*w.scaleDomainCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsPrefix),
		*w.scaleDomainSucceededCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsPrefix),
		*w.scaleDomainFailedCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsPrefix),
	}

	if err := config.PsrEnv.LoadFromEnv(w.GetEnvDescList()); err != nil {
		return w, err
	}

	metricsLabels := map[string]string{
		config.PsrWorkerTypeMetricsName: config.PsrEnv.GetEnv(config.PsrWorkerType),
	}

	w.metricDescList = metrics.BuildMetricDescList([]*metrics.MetricItem{
		&w.scaleDomainCountTotal,
		&w.scaleDomainSucceededCountTotal,
		&w.scaleDomainFailedCountTotal,
	}, metricsLabels, w.GetWorkerDesc().MetricsPrefix)

	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w worker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		WorkerType:    config.WorkerTypeWlsScale,
		Description:   "The scale domain worker scales up and scales down the domain",
		MetricsPrefix: metricsPrefix,
	}
}

func (w worker) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{
		{Key: DomainUID, DefaultVal: "", Required: true},
		{Key: DomainNamespace, DefaultVal: "", Required: true},
		{Key: MinReplicaCount, DefaultVal: "", Required: true},
		{Key: MaxReplicaCount, DefaultVal: "", Required: true},
	}
}

func (w worker) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w worker) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.scaleDomainCountTotal.BuildMetric(),
		w.scaleDomainSucceededCountTotal.BuildMetric(),
		w.scaleDomainFailedCountTotal.BuildMetric(),
	}
}

func (w worker) WantLoopInfoLogged() bool {
	return false
}

func (w worker) PreconditionsMet() (bool, error) {
	return true, nil
}

func (w worker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {

	//increment scaleDomainCountTotal
	atomic.AddInt64(&w.workerMetrics.scaleDomainCountTotal.Val, 1)
	var replicas int64
	max, err := strconv.ParseInt(config.PsrEnv.GetEnv(MaxReplicaCount), 10, 64)
	if err != nil {
		return fmt.Errorf("MaxReplicaCount can not be parsed to an integer: %v", err)
	}
	min, err := strconv.ParseInt(config.PsrEnv.GetEnv(MinReplicaCount), 10, 64)
	if err != nil {
		return fmt.Errorf("MinReplicaCount can not be parsed to an integer: %v", err)
	}
	domainNamespace := config.PsrEnv.GetEnv(DomainNamespace)
	domainUID := config.PsrEnv.GetEnv(DomainUID)

	client, err := w.createClient()
	if err != nil {
		return err
	}

	// get current replicas at /spec/replicas
	currentReplicas, err := weblogic.GetCurrentReplicas(client, domainNamespace, domainUID)
	if err != nil {
		atomic.AddInt64(&w.workerMetrics.scaleDomainFailedCountTotal.Val, 1)
		return err
	}

	// set replicas to scale based on current replicas
	if currentReplicas > min {
		replicas = min
	} else {
		replicas = max
	}
	err = weblogic.PatchReplicas(client, domainNamespace, domainUID, replicas)
	if err != nil {
		atomic.AddInt64(&w.workerMetrics.scaleDomainFailedCountTotal.Val, 1)
		return err
	}
	success, err := w.waitForReadyReplicas(client, domainNamespace, domainUID, replicas)
	if err != nil {
		atomic.AddInt64(&w.workerMetrics.scaleDomainFailedCountTotal.Val, 1)
		return err
	}
	if success {
		atomic.AddInt64(&w.workerMetrics.scaleDomainSucceededCountTotal.Val, 1)
	} else {
		atomic.AddInt64(&w.workerMetrics.scaleDomainFailedCountTotal.Val, 1)
	}
	return nil
}

func (w worker) createClient() (dynamic.Interface, error) {
	cfg, err := controllerruntime.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("Failed to get controller-runtime config %v", err)
	}
	return dynamic.NewForConfig(cfg)
}

func (w worker) waitForReadyReplicas(client dynamic.Interface, namespace string, name string, readyReplicas int64) (bool, error) {
	for {
		rr, err := weblogic.GetReadyReplicas(client, namespace, name)
		if err != nil {
			return false, err
		}
		if rr == readyReplicas {
			break
		}
		time.Sleep(1 * time.Second)
	}
	return true, nil
}
