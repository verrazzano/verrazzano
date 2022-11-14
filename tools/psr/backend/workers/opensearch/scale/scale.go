// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scale

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	"github.com/verrazzano/verrazzano/tools/psr/backend/pkg/k8sclient"
	psropensearch "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/opensearch"
	psrvz "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/verrazzano"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	er "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"

	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

const (
	openSearchTier  = "OPEN_SEARCH_TIER"
	minReplicaCount = "MIN_REPLICA_COUNT"
	maxReplicaCount = "MAX_REPLICA_COUNT"
)

type scaleWorker struct {
	metricDescList []prometheus.Desc
	*workerMetrics
	*state
	psrClient k8sclient.PsrClient
	log       vzlog.VerrazzanoLogger
}

type state struct {
	startScaleTime int64
	directionOut   bool
}

var _ spi.Worker = scaleWorker{}

// scaleMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	scaleOutCountTotal metrics.MetricItem
	scaleInCountTotal  metrics.MetricItem
	scaleOutSeconds    metrics.MetricItem
	scaleInSeconds     metrics.MetricItem
}

func NewScaleWorker() (spi.Worker, error) {
	c, err := k8sclient.NewPsrClient()
	if err != nil {
		return nil, err
	}
	w := scaleWorker{
		psrClient: c,
		log:       vzlog.DefaultLogger(),
		state:     &state{},
		workerMetrics: &workerMetrics{
			scaleOutCountTotal: metrics.MetricItem{
				Name: "opensearch_scale_out_count_total",
				Help: "The total number of times OpenSearch scaled out",
				Type: prometheus.CounterValue,
			},
			scaleInCountTotal: metrics.MetricItem{
				Name: "opensearch_scale_in_count_total",
				Help: "The total number of times OpenSearch scaled in",
				Type: prometheus.CounterValue,
			},
			scaleOutSeconds: metrics.MetricItem{
				Name: "opensearch_scale_out_seconds",
				Help: "The number of seconds taken to scale out OpenSearch",
				Type: prometheus.CounterValue,
			},
			scaleInSeconds: metrics.MetricItem{
				Name: "opensearch_scale_in_seconds",
				Help: "The number of seconds taken to scale in OpenSearch",
				Type: prometheus.CounterValue,
			},
		},
	}

	w.metricDescList = []prometheus.Desc{
		*w.scaleOutCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.scaleInCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.scaleOutSeconds.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.scaleInSeconds.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
	}

	return w, nil
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
		{Key: minReplicaCount, DefaultVal: "3", Required: false},
		{Key: maxReplicaCount, DefaultVal: "5", Required: false},
	}
}

func (w scaleWorker) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w scaleWorker) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.scaleInCountTotal.BuildMetric(),
		w.scaleOutCountTotal.BuildMetric(),
		w.scaleOutSeconds.BuildMetric(),
		w.scaleInSeconds.BuildMetric(),
	}
}

func (w scaleWorker) WantLoopInfoLogged() bool {
	return false
}

// DoWork continuously scales a specified OpenSearch out and in by modifying the VZ CR OpenSearch component
func (w scaleWorker) DoWork(_ config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	// validate OS tier
	tier := config.PsrEnv.GetEnv(openSearchTier)
	if tier != psropensearch.MasterTier && tier != psropensearch.DataTier && tier != psropensearch.IngestTier {
		return fmt.Errorf("error, not a valid OpenSearch tier to scale")
	}
	// Wait until VZ is ready
	cr, err := w.waitReady(true)
	if err != nil {
		log.Progress("Failed to wait for Verrazzano to be ready after update.  The test results are not valid %v", err)
		return err
	}
	// Update the elapsed time of the scale operation
	if w.state.startScaleTime > 0 {
		elapsedSecs := time.Now().UnixNano() - w.state.startScaleTime
		if w.state.directionOut {
			atomic.StoreInt64(&w.workerMetrics.scaleOutSeconds.Val, elapsedSecs)
		} else {
			atomic.StoreInt64(&w.workerMetrics.scaleInSeconds.Val, elapsedSecs)
		}
	}
	// Get the current number OpenSearch pods that exist for the given tier
	pods, err := psropensearch.GetPodsForTier(w.psrClient.CrtlRuntime, tier)
	if err != nil {
		log.Progress("Failed to get the pods for tier %s: %v", tier, err)
		return err
	}
	// Create a modifier that is used to update the Verrazzno CR opensearch replica field
	m, desiredReplicas, err := w.getUpdateModifier(tier, len(pods))
	if err != nil {
		return err
	}
	log.Infof("Updating Verrazzano CR OpenSearch %s tier, scaling to %v replicas", tier, desiredReplicas)

	// Update metrics
	if desiredReplicas > len(pods) {
		w.state.directionOut = true
		atomic.AddInt64(&w.workerMetrics.scaleOutCountTotal.Val, 1)
	} else {
		w.state.directionOut = false
		atomic.AddInt64(&w.workerMetrics.scaleInCountTotal.Val, 1)
	}
	w.state.startScaleTime = time.Now().UnixNano()

	// Update the CR to change the replica count
	err = w.updateCr(cr, m)
	if err != nil {
		return err
	}

	// Wait until VZ is NOT ready, this means it started working on the change
	_, err = w.waitReady(false)
	if err != nil {
		log.Progress("Failed to wait for Verrazzano to be NOT ready after update.  The test results are not valid %v", err)
		return err
	}

	return nil
}

func (w scaleWorker) getUpdateModifier(tier string, currentReplicas int) (update.CRModifier, int, error) {
	max, err := strconv.Atoi(config.PsrEnv.GetEnv(maxReplicaCount))
	if err != nil {
		return nil, 0, fmt.Errorf("maxReplicaCount can not be parsed to an integer: %f", err)
	}
	min, err := strconv.Atoi(config.PsrEnv.GetEnv(minReplicaCount))
	if err != nil {
		return nil, 0, fmt.Errorf("minReplicaCount can not be parsed to an integer: %f", err)
	}
	if min < 3 {
		return nil, 0, fmt.Errorf("minReplicaCount can not be less than 3")
	}
	var desiredReplicas int32
	if currentReplicas != min {
		desiredReplicas = int32(min)
	} else {
		desiredReplicas = int32(max)
	}

	var m update.CRModifier

	switch tier {
	case psropensearch.MasterTier:
		m = psropensearch.OpensearchMasterNodeGroupModifier{NodeReplicas: desiredReplicas}
	case psropensearch.DataTier:
		m = psropensearch.OpensearchDataNodeGroupModifier{NodeReplicas: desiredReplicas}
	case psropensearch.IngestTier:
		m = psropensearch.OpensearchIngestNodeGroupModifier{NodeReplicas: desiredReplicas}
	}
	return m, int(desiredReplicas), nil
}

// updateCr updates the Verrazzano CR and retries if there is a conflict error
func (w scaleWorker) updateCr(cr *vzv1alpha1.Verrazzano, m update.CRModifier) error {
	for {
		// Modify the CR
		m.ModifyCR(cr)

		err := psrvz.UpdateVerrazzano(w.psrClient.VzInstall, cr)
		if err == nil {
			break
		}
		if !er.IsUpdateConflict(err) {
			return fmt.Errorf("Failed to scale update Verrazzano cr: %v", err)
		}
		// Conflict error, get latest vz cr
		time.Sleep(1 * time.Second)
		logMsg := fmt.Sprintf("VZ conflict error, retrying")
		w.log.Infof(logMsg)

		cr, err = psrvz.GetVerrazzano(w.psrClient.VzInstall)
		if err != nil {
			return err
		}
	}
	w.log.Info("Updated Verrazzano CR")
	return nil
}

// Wait until Verrazzano is ready or not ready
func (w scaleWorker) waitReady(desiredReady bool) (cr *vzv1alpha1.Verrazzano, err error) {
	for {
		cr, err = psrvz.GetVerrazzano(w.psrClient.VzInstall)
		if err != nil {
			return nil, err
		}
		ready := psrvz.IsReady(cr)
		if err != nil {
			return nil, err
		}
		if ready == desiredReady {
			break
		}
		w.log.Progressf("Waiting for Verrazzano CR ready state to be %v", desiredReady)
		time.Sleep(1 * time.Second)
	}
	return cr, err
}
