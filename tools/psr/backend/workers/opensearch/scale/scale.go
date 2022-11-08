// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scale

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	psrvz "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/verrazzano"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	er "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update/opensearch"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	psropensearch "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/opensearch"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"

	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
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
	metricDescList []prometheus.Desc
	*workerMetrics
	*nextScale

	// ctrlRuntimeClient is a controller-runtime client
	ctrlRuntimeClient client.Client
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

	c, err := createClient()
	if err != nil {
		return nil, err
	}

	w.ctrlRuntimeClient = c

	return w, nil
}

func createClient() (client.Client, error) {
	cfg, err := controllerruntime.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("Failed to get controller-runtime config %v", err)
	}
	c, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("Failed to create a controller-runtime client %v", err)
	}
	_ = vzv1alpha1.AddToScheme(c.Scheme())
	return c, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w scaleWorker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		WorkerType:  config.WorkerTypeScale,
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

func (w scaleWorker) WantLoopInfoLogged() bool {
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

	var m update.CRModifier

	switch tier {
	case masterTier:
		m = opensearch.OpensearchMasterNodeGroupModifier{NodeReplicas: replicas}
	case dataTier:
		m = opensearch.OpensearchDataNodeGroupModifier{NodeReplicas: replicas}
	case ingestTier:
		m = opensearch.OpensearchIngestNodeGroupModifier{NodeReplicas: replicas}
	}

	for {
		err := psrvz.UpdateCR(w.ctrlRuntimeClient, m)
		if err != nil {
			if er.IsUpdateConflict(err) {
				time.Sleep(3 * time.Second)
				logMsg := fmt.Sprintf("VZ conflict error, retrying")
				log.Infof(logMsg)
				continue
				//} else if er.IsUpdateConflict(err) {
				//	logMsg := fmt.Sprintf("VZ CR in state Reconciling, not Ready yet")
				//	log.Infof(logMsg)
				//	break
			} else {
				return fmt.Errorf("failed to scale OpenSearch %s replicas: %f", tier, err)
			}
		}
		break
	}

	for {
		// get the VZ CR
		vz, err := pkg.GetVerrazzano()
		if err != nil {
			return err
		}

		if psropensearch.IsOSReady(w.ctrlRuntimeClient, vz) {
			finishWork(nextScale, metric, delta)
			logMsg := fmt.Sprintf("OpenSearch %s tier scaled to %b replicas", tier, replicas)
			log.Infof(logMsg)
			break
		}
		logMsg := fmt.Sprintf("Waiting for OpenSearch to enter Ready state")
		log.Infof(logMsg)
		time.Sleep(3 * time.Second)
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
