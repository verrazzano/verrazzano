// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scale

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
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
	nextScale string
}

const (
	UP   string = "UP"
	DOWN string = "DOWN"
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
			nextScale: UP,
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

	nextScale := w.nextScale.nextScale
	tier := config.PsrEnv.GetEnv(openSearchTier)

	// validate tier
	if tier != masterTier && tier != dataTier && tier != ingestTier {
		return fmt.Errorf("error, not a valid OpenSearch tier to scale")
	}

	// validate replicas
	var replicas int
	var err error
	if nextScale == UP {
		replicas, err = strconv.Atoi(config.PsrEnv.GetEnv(maxReplicaCount))
		if err != nil {
			return fmt.Errorf("maxReplicaCount can not be parsed to an integer: %f", err)
		}
	} else {
		replicas, err = strconv.Atoi(config.PsrEnv.GetEnv(minReplicaCount))
		if err != nil {
			return fmt.Errorf("minReplicaCount can not be parsed to an integer: %f", err)
		}
	}

	// check OpenSearch is ready

	// switch on tier
	switch tier {
	case masterTier:
		m := opensearch.OpensearchMasterNodeGroupModifier{NodeReplicas: int32(replicas)}
		err := update.UpdateCR(m)
		if err != nil {
			return fmt.Errorf("failed to scale OpenSearch replicas: %f", err)
		}
	case dataTier:
		m := opensearch.OpensearchDataNodeGroupModifier{NodeReplicas: int32(replicas)}
		err := update.UpdateCR(m)
		if err != nil {
			return fmt.Errorf("failed to scale OpenSearch replicas: %f", err)
		}
	case ingestTier:
		m := opensearch.OpensearchIngestNodeGroupModifier{NodeReplicas: int32(replicas)}
		err := update.UpdateCR(m)
		if err != nil {
			return fmt.Errorf("failed to scale OpenSearch replicas: %f", err)
		}
	}
	return nil
}

func (w scaleWorker) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w scaleWorker) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.scaleOutCountTotal.BuildMetric(),
		w.scaleOutCountTotal.BuildMetric(),
	}
}
