// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scale

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	er "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/k8s/update"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	oscomp "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearch"
	spicomponent "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
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
	scaleOutCountTotal metrics.MetricItem
	scaleInCountTotal  metrics.MetricItem
}

func NewScaleWorker() (spi.Worker, error) {

	w := scaleWorker{
		workerMetrics: &workerMetrics{
			scaleOutCountTotal: metrics.MetricItem{
				Name: "scale_out_count_total",
				Help: "The total number of times OpenSearch has been scaled out",
				Type: prometheus.CounterValue,
			},
			scaleInCountTotal: metrics.MetricItem{
				Name: "scale_in_count_total",
				Help: "The total number of times OpenSearch has been scaled in",
				Type: prometheus.CounterValue,
			},
		},
		nextScale: &nextScale{
			val: OUT,
		},
	}

	w.metricDescList = []prometheus.Desc{
		*w.scaleOutCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.scaleInCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
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

// DoWork continuously scales a specified OpenSearch out and in by modifying the VZ CR
// It uses the nextScale value to determine which direction OpenSearch should be scaled next
// This worker is blocking until the current scaling of replicas has completed and OpenSearch reaches a "ready" state
// Verrazzano installed using the v1beta1 API is assumed
func (w scaleWorker) DoWork(_ config.CommonConfig, log vzlog.VerrazzanoLogger) error {
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

	var m update.CRModifierV1beta1

	switch tier {
	case masterTier:
		m = opensearch.OpensearchMasterNodeGroupModifier{NodeReplicas: replicas}
	case dataTier:
		m = opensearch.OpensearchDataNodeGroupModifier{NodeReplicas: replicas}
	case ingestTier:
		m = opensearch.OpensearchIngestNodeGroupModifier{NodeReplicas: replicas}
	}

	for {
		err := update.UpdateCRV1beta1(m)
		if err != nil {
			if er.IsUpdateConflict(err) {
				continue
			} else {
				return fmt.Errorf("failed to scale OpenSearch %s replicas: %f", tier, err)
			}
		} else {
			break
		}
	}

	for {
		// get the VZ CR
		vz, err := pkg.GetVerrazzanoV1beta1()
		if err != nil {
			return err
		}

		// get controller runtime client
		client, err := pkg.GetPromOperatorClient()
		if err != nil {
			return err
		}

		// check if actual number of replicas is equal to the expected number
		ctx, err := spicomponent.NewContext(log, client, nil, vz, false)
		if err != nil {
			return err
		}
		if oscomp.IsOSReady(ctx) {
			finishWork(nextScale, metric, delta)
			logMsg := fmt.Sprintf("OpenSearch %s tier scaled to %b replicas", tier, replicas)
			log.Infof(logMsg)
			break
		}
		time.Sleep(100 * time.Millisecond)
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

// finishWork switches the nextScale value and pushes the scale metric
// once OpenSearch has finished scaling to the desired replica count
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
