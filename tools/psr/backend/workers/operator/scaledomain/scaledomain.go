// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scaledomain

import (
	"fmt"
	"k8s.io/client-go/dynamic"
	controllerruntime "sigs.k8s.io/controller-runtime"
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
	specField       = "spec"
)

type scaleDomain struct {
	spi.Worker
	metricDescList []prometheus.Desc
	*workerMetrics
}

var _ spi.Worker = scaleDomain{}

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	scaleDomainCountTotal          metrics.MetricItem
	scaleDomainSucceededCountTotal metrics.MetricItem
	scaleDomainFailedCountTotal    metrics.MetricItem
}

func NewScaleDomainWorker() (spi.Worker, error) {
	w := scaleDomain{workerMetrics: &workerMetrics{
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
		*w.scaleDomainCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.scaleDomainSucceededCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.scaleDomainFailedCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
	}
	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w scaleDomain) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		WorkerType:  config.WorkerTypeScaleDomain,
		Description: "The scale domain worker scales up and scales down the domain",
		MetricsName: "scaleDomain",
	}
}

func (w scaleDomain) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{
		{Key: DomainUID, DefaultVal: "", Required: true},
		{Key: DomainNamespace, DefaultVal: "", Required: true},
		{Key: MinReplicaCount, DefaultVal: "", Required: true},
		{Key: MaxReplicaCount, DefaultVal: "", Required: true},
	}
}

func (w scaleDomain) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w scaleDomain) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.scaleDomainCountTotal.BuildMetric(),
		w.scaleDomainSucceededCountTotal.BuildMetric(),
		w.scaleDomainFailedCountTotal.BuildMetric(),
	}
}

func (w scaleDomain) WantLoopInfoLogged() bool {
	return false
}

func (w scaleDomain) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {

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
	log.Infof("current replicas %v", currentReplicas)

	// set replicas to scale based on current replicas
	if currentReplicas > min {
		replicas = min
	} else {
		replicas = max
	}
	log.Infof("replicas to scale %v", replicas)
	err = weblogic.PatchReplicas(client, domainNamespace, domainUID, replicas)
	if err != nil {
		log.Infof("patchReplicas failed")
		atomic.AddInt64(&w.workerMetrics.scaleDomainFailedCountTotal.Val, 1)
		return err
	}
	success, err := w.waitForReadyReplicas(client, domainNamespace, domainUID, replicas)
	if err != nil {
		log.Infof("wait for readyReplicas failed")
		atomic.AddInt64(&w.workerMetrics.scaleDomainFailedCountTotal.Val, 1)
		return err
	}
	if success {
		log.Infof("readyReplicas success")
		atomic.AddInt64(&w.workerMetrics.scaleDomainSucceededCountTotal.Val, 1)
	} else {
		log.Infof("readyReplicas failed")
		atomic.AddInt64(&w.workerMetrics.scaleDomainFailedCountTotal.Val, 1)
	}
	return nil
}

func (w scaleDomain) createClient() (dynamic.Interface, error) {
	cfg, err := controllerruntime.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("Failed to get controller-runtime config %v", err)
	}
	return dynamic.NewForConfig(cfg)
}

func (w scaleDomain) waitForReadyReplicas(client dynamic.Interface, namespace string, name string, readyReplicas int64) (bool, error) {
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
