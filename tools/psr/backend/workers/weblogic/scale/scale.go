// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scale

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/psr/backend/pkg/k8sclient"
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

var funcNewPsrClient = k8sclient.NewPsrClient

//var funcNewDynClient = k8sclient.NewDynamicClient

type worker struct {
	metricDescList []prometheus.Desc
	*workerMetrics
	psrClient k8sclient.PsrClient
	//dynClient k8sclient.DynamicClient
	*state
	log vzlog.VerrazzanoLogger
}

type state struct {
	startScaleTime int64
	directionOut   bool
}

var _ spi.Worker = worker{}

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	scaleUpDomainCountTotal   metrics.MetricItem
	scaleDownDomainCountTotal metrics.MetricItem
	scaleUpSeconds            metrics.MetricItem
	scaleDownSeconds          metrics.MetricItem
}

func NewScaleWorker() (spi.Worker, error) {
	c, err := funcNewPsrClient()
	if err != nil {
		return nil, err
	}
	//d, err := funcNewDynClient()
	if err != nil {
		return nil, err
	}
	w := worker{
		psrClient: c,
		log:       vzlog.DefaultLogger(),
		state:     &state{},
		//dynClient: d,
		workerMetrics: &workerMetrics{
			scaleUpDomainCountTotal: metrics.MetricItem{
				Name: "scale_up_domain_count_total",
				Help: "The total number of successful scale up domain requests",
				Type: prometheus.CounterValue,
			},
			scaleDownDomainCountTotal: metrics.MetricItem{
				Name: "scale_down_domain_count_total",
				Help: "The total number of failed scale down domain requests",
				Type: prometheus.CounterValue,
			},
			scaleUpSeconds: metrics.MetricItem{
				Name: "scale_up_seconds",
				Help: "The total number of seconds elapsed to scale up the domain",
				Type: prometheus.GaugeValue,
			},
			scaleDownSeconds: metrics.MetricItem{
				Name: "scale_down_seconds",
				Help: "The total number of seconds elapsed to scale down the domain",
				Type: prometheus.GaugeValue,
			},
		}}

	w.metricDescList = []prometheus.Desc{
		*w.scaleUpDomainCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsPrefix),
		*w.scaleDownDomainCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsPrefix),
		*w.scaleUpSeconds.BuildMetricDesc(w.GetWorkerDesc().MetricsPrefix),
		*w.scaleDownSeconds.BuildMetricDesc(w.GetWorkerDesc().MetricsPrefix),
	}

	if err := config.PsrEnv.LoadFromEnv(w.GetEnvDescList()); err != nil {
		return w, err
	}

	metricsLabels := map[string]string{
		config.PsrWorkerTypeMetricsName: config.PsrEnv.GetEnv(config.PsrWorkerType),
	}

	w.metricDescList = metrics.BuildMetricDescList([]*metrics.MetricItem{
		&w.scaleUpDomainCountTotal,
		&w.scaleDownDomainCountTotal,
		&w.scaleUpSeconds,
		&w.scaleDownSeconds,
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
		{Key: MinReplicaCount, DefaultVal: "2", Required: true},
		{Key: MaxReplicaCount, DefaultVal: "4", Required: true},
	}
}

func (w worker) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w worker) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.scaleUpDomainCountTotal.BuildMetric(),
		w.scaleDownDomainCountTotal.BuildMetric(),
		w.scaleUpSeconds.BuildMetric(),
		w.scaleDownSeconds.BuildMetric(),
	}
}

func (w worker) WantLoopInfoLogged() bool {
	return false
}

func (w worker) PreconditionsMet() (bool, error) {
	return true, nil
}

func (w worker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
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

	client := w.psrClient.DynClient
	// client := w.dynClient.DynClient
	if err != nil {
		return log.ErrorfNewErr("Failed to get client: %v", err)
	}

	// get current replicas at /spec/replicas
	currentReplicas, err := weblogic.GetCurrentReplicas(client, domainNamespace, domainUID)
	if err != nil {
		return log.ErrorfNewErr("Failed to get current replicas: %v", err)
	}

	// set replicas to scale based on current replicas
	if currentReplicas > min {
		replicas = min
		w.state.directionOut = false
	} else {
		replicas = max
		w.state.directionOut = true
	}
	w.state.startScaleTime = time.Now().UnixNano()
	err = weblogic.PatchReplicas(client, domainNamespace, domainUID, replicas)
	if err != nil {
		return log.ErrorfNewErr("Failed to patch the replicas: %v", err)
	}
	err = w.waitForReadyReplicas(client, domainNamespace, domainUID, replicas)
	elapsedSecs := time.Now().UnixNano() - w.state.startScaleTime
	if err != nil {
		return log.ErrorfNewErr("Failed to get the ready replicas: %v", err)
	}
	if w.state.directionOut {
		atomic.StoreInt64(&w.workerMetrics.scaleUpSeconds.Val, elapsedSecs)
		atomic.AddInt64(&w.workerMetrics.scaleUpDomainCountTotal.Val, 1)
	} else {
		atomic.StoreInt64(&w.workerMetrics.scaleDownSeconds.Val, elapsedSecs)
		atomic.AddInt64(&w.workerMetrics.scaleDownDomainCountTotal.Val, 1)
	}

	return nil
}

func (w worker) waitForReadyReplicas(client dynamic.Interface, namespace string, name string, readyReplicas int64) error {
	for {
		rr, err := weblogic.GetReadyReplicas(client, namespace, name)
		if err != nil {
			return err
		}
		if rr == readyReplicas {
			break
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}
