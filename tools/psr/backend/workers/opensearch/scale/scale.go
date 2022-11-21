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
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/pkg/k8sclient"
	psropensearch "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/opensearch"
	psrvz "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/verrazzano"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
)

const (
	// metricsPrefix is the prefix that is automatically pre-pended to all metrics exported by this worker.
	metricsPrefix = "opensearch_scaling"

	openSearchTier           = "OPENSEARCH_TIER"
	minReplicaCount          = "MIN_REPLICA_COUNT"
	maxReplicaCount          = "MAX_REPLICA_COUNT"
	openSearchTierMetricName = "opensearch_tier"
)

var funcNewPsrClient = k8sclient.NewPsrClient

type worker struct {
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

var _ spi.Worker = worker{}

// scaleMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	scaleOutCountTotal metrics.MetricItem
	scaleInCountTotal  metrics.MetricItem
	scaleOutSeconds    metrics.MetricItem
	scaleInSeconds     metrics.MetricItem
}

func NewScaleWorker() (spi.Worker, error) {
	c, err := funcNewPsrClient()
	if err != nil {
		return nil, err
	}
	w := worker{
		psrClient: c,
		log:       vzlog.DefaultLogger(),
		state:     &state{},
		workerMetrics: &workerMetrics{
			scaleOutCountTotal: metrics.MetricItem{
				Name: "scale_out_count_total",
				Help: "The total number of times OpenSearch scaled out",
				Type: prometheus.CounterValue,
			},
			scaleInCountTotal: metrics.MetricItem{
				Name: "scale_in_count_total",
				Help: "The total number of times OpenSearch scaled in",
				Type: prometheus.CounterValue,
			},
			scaleOutSeconds: metrics.MetricItem{
				Name: "scale_out_seconds",
				Help: "The number of seconds elapsed to scale out OpenSearch",
				Type: prometheus.GaugeValue,
			},
			scaleInSeconds: metrics.MetricItem{
				Name: "scale_in_seconds",
				Help: "The number of seconds elapsed to scale in OpenSearch",
				Type: prometheus.GaugeValue,
			},
		},
	}

	if err = config.PsrEnv.LoadFromEnv(w.GetEnvDescList()); err != nil {
		return w, err
	}

	tier, err := psropensearch.ValidateOpenSeachTier(openSearchTier)
	if err != nil {
		return w, err
	}

	metricsLabels := map[string]string{
		openSearchTierMetricName:        tier,
		config.PsrWorkerTypeMetricsName: config.PsrEnv.GetEnv(config.PsrWorkerType),
	}

	w.metricDescList = metrics.BuildMetricDescList([]*metrics.MetricItem{
		&w.scaleOutCountTotal,
		&w.scaleInCountTotal,
		&w.scaleOutSeconds,
		&w.scaleInSeconds,
	}, metricsLabels, w.GetWorkerDesc().MetricsPrefix)

	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w worker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		WorkerType:    config.WorkerTypeOpsScale,
		Description:   "The OpenSearch scale worker scales an OpenSearch tier in and out continuously",
		MetricsPrefix: metricsPrefix,
	}
}

func (w worker) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{
		{Key: openSearchTier, DefaultVal: "", Required: true},
		{Key: minReplicaCount, DefaultVal: "3", Required: false},
		{Key: maxReplicaCount, DefaultVal: "5", Required: false},
	}
}

func (w worker) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w worker) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.scaleInCountTotal.BuildMetric(),
		w.scaleOutCountTotal.BuildMetric(),
		w.scaleOutSeconds.BuildMetric(),
		w.scaleInSeconds.BuildMetric(),
	}
}

func (w worker) WantLoopInfoLogged() bool {
	return false
}

func (w worker) PreconditionsMet() (bool, error) {
	return true, nil
}

// DoWork continuously scales a specified OpenSearch out and in by modifying the VZ CR OpenSearch component
func (w worker) DoWork(_ config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	// validate OS tier
	tier, err := psropensearch.ValidateOpenSeachTier(openSearchTier)
	if err != nil {
		return err
	}

	// Wait until VZ is ready
	cr, err := w.waitReady(true)
	if err != nil {
		return log.ErrorfNewErr("Failed to wait for Verrazzano to be ready after update.  The test results are not valid %v", err)
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
		return log.ErrorfNewErr("Failed to get the pods for tier %s: %v", tier, err)
	}
	existingReplicas := len(pods)
	if err != nil {
		return log.ErrorfNewErr("Failed to get the pods for tier %s: %v", tier, err)
	}
	if existingReplicas == 0 {
		return log.ErrorfNewErr("Failed, no pods exist for tier %s", tier)
	}
	// Create a modifier that is used to update the Verrazzno CR opensearch replica field
	m, desiredReplicas, err := w.getUpdateModifier(tier, existingReplicas)
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
		return log.ErrorfNewErr("Failed to wait for Verrazzano to be NOT ready after update.  The test results are not valid %v", err)
	}

	return nil
}

func (w worker) getUpdateModifier(tier string, currentReplicas int) (update.CRModifier, int, error) {
	max, err := strconv.ParseInt(config.PsrEnv.GetEnv(maxReplicaCount), 10, 32)
	if err != nil {
		return nil, 0, fmt.Errorf("maxReplicaCount can not be parsed to an integer: %v", err)
	}
	min, err := strconv.ParseInt(config.PsrEnv.GetEnv(minReplicaCount), 10, 32)
	if err != nil {
		return nil, 0, fmt.Errorf("minReplicaCount can not be parsed to an integer: %v", err)
	}
	if min < 3 {
		return nil, 0, fmt.Errorf("minReplicaCount can not be less than 3")
	}
	var desiredReplicas int32
	if currentReplicas != int(min) {
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
func (w worker) updateCr(cr *vzv1alpha1.Verrazzano, m update.CRModifier) error {
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
		w.log.Info("OpenSearch scaling, Verrazzano CR conflict error, retrying")

		cr, err = psrvz.GetVerrazzano(w.psrClient.VzInstall)
		if err != nil {
			return err
		}
	}
	w.log.Info("Updated Verrazzano CR")
	return nil
}

// Wait until Verrazzano is ready or not ready
func (w worker) waitReady(desiredReady bool) (cr *vzv1alpha1.Verrazzano, err error) {
	for {
		cr, err = psrvz.GetVerrazzano(w.psrClient.VzInstall)
		if err != nil {
			return nil, err
		}
		ready := psrvz.IsReady(cr)
		if ready == desiredReady {
			break
		}
		w.log.Progressf("Waiting for Verrazzano CR ready state to be %v", desiredReady)
		time.Sleep(1 * time.Second)
	}
	return cr, err
}
