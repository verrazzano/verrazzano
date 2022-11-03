// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scale

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/k8s/update"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tests/e2e/update/opensearch"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
)

const (
	openSearchTier    = "OPEN_SEARCH_TIER"
	scaleDelayPerTier = "SCALE_DELAY_PER_TIER"
	minReplicaCount   = "MIN_REPLICA_COUNT"
	maxReplicaCount   = "MAX_REPLICA_COUNT"

	masterTier = "master"
	dataTier   = "data"
	ingestTier = "ingest"
)

type scaleWorker struct {
	spi.Worker
	metricDescList []prometheus.Desc
	*workerMetrics
	*nextScale
}

type nextScale struct {
	val string
}

const (
	OUT string = "OUT"
	IN  string = "IN"
)

var _ spi.Worker = scaleWorker{}

// scaleMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	scaleInCountTotal  metrics.MetricItem
	scaleOutCountTotal metrics.MetricItem
}

func NewScaleWorker() (spi.Worker, error) {

	w := scaleWorker{
		workerMetrics: &workerMetrics{
			scaleInCountTotal: metrics.MetricItem{
				Name: "scale_in_count_total",
				Help: "The total number of times OpenSearch has been scaled in",
				Type: prometheus.CounterValue,
			},
			scaleOutCountTotal: metrics.MetricItem{
				Name: "scale_out_count_total",
				Help: "The total number of times OpenSearch has been scaled out",
				Type: prometheus.CounterValue,
			},
		},
		nextScale: &nextScale{
			val: OUT,
		},
	}

	w.metricDescList = []prometheus.Desc{
		*w.scaleInCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.scaleOutCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
	}

	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w scaleWorker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		EnvName:     config.WorkerTypeScale,
		Description: "Worker to scale the number of specified OpenSearch tiers",
		MetricsName: "scale",
	}
}

func (w scaleWorker) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{
		{Key: openSearchTier, DefaultVal: "", Required: true},
		{Key: scaleDelayPerTier, DefaultVal: "5s", Required: false},
		{Key: minReplicaCount, DefaultVal: "3", Required: false},
		{Key: maxReplicaCount, DefaultVal: "5", Required: false},
	}
}

func (w scaleWorker) WantIterationInfoLogged() bool {
	return false
}

func (w scaleWorker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {

	nextScale := &w.nextScale.val
	tier := config.PsrEnv.GetEnv(openSearchTier)

	// validate OS tier
	if tier != masterTier && tier != dataTier && tier != ingestTier {
		return fmt.Errorf("error, not a valid OpenSearch tier to scale")
	}

	var replicas int32
	var delta int64
	var metric *metrics.MetricItem
	// validate replicas
	if *nextScale == OUT {
		max, err := strconv.ParseInt(config.PsrEnv.GetEnv(maxReplicaCount), 10, 32)
		if err != nil {
			return fmt.Errorf("maxReplicaCount can not be parsed to an integer: %f", err)
		}
		replicas = int32(max)
		metric = &w.workerMetrics.scaleOutCountTotal
		delta = 1
	} else {
		min, err := strconv.ParseInt(config.PsrEnv.GetEnv(minReplicaCount), 10, 32)
		if err != nil {
			return fmt.Errorf("minReplicaCount can not be parsed to an integer: %f", err)
		}
		if replicas < 1 {
			return fmt.Errorf("minReplicaCount can not be less than 1")
		}
		replicas = int32(min)
		metric = &w.workerMetrics.scaleInCountTotal
		delta = -1
	}

	// TODO check if OpenSearch is ready

	switch tier {
	case masterTier:
		m := opensearch.OpensearchMasterNodeGroupModifier{NodeReplicas: replicas}
		err := update.UpdateCRV1beta1(m)
		if err != nil {
			return fmt.Errorf("failed to scale OpenSearch %s replicas: %f", tier, err)
		}
	case dataTier:
		m := opensearch.OpensearchDataNodeGroupModifier{NodeReplicas: replicas}
		err := update.UpdateCRV1beta1(m)
		if err != nil {
			return fmt.Errorf("failed to scale OpenSearch %s replicas: %f", tier, err)
		}
	case ingestTier:
		m := opensearch.OpensearchIngestNodeGroupModifier{NodeReplicas: replicas}
		err := update.UpdateCRV1beta1(m)
		if err != nil {
			return fmt.Errorf("failed to scale OpenSearch %s replicas: %f", tier, err)
		}
	}
	atomic.AddInt64(&metric.Val, delta)

	// TODO check replica count == expected
	if true {
		finishWork(nextScale, metric, delta)
	}
	return nil
}

func (w scaleWorker) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w scaleWorker) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.scaleInCountTotal.BuildMetric(),
		w.scaleOutCountTotal.BuildMetric(),
	}
}

func finishWork(next *string, metric *metrics.MetricItem, delta int64) {
	// Alternate between scale out and in
	if *next == OUT {
		*next = IN
	} else {
		*next = OUT
	}
	// Add metric once work has finished
	atomic.AddInt64(&metric.Val, delta)
}
