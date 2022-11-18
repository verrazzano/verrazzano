// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restart

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/tools/psr/backend/pkg/k8sclient"
	psropensearch "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/opensearch"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"math/big"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
)

const (
	openSearchTier           = "OPENSEARCH_TIER"
	openSearchTierMetricName = "opensearch_tier"
)

var funcNewPsrClient = k8sclient.NewPsrClient

type worker struct {
	metricDescList []prometheus.Desc
	*workerMetrics
	psrClient k8sclient.PsrClient
	log       vzlog.VerrazzanoLogger
	*restartData
}

type restartData struct {
	restartStartTime int64
	restartedPodUID  types.UID
}

var _ spi.Worker = worker{}

// restartMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	restartCount metrics.MetricItem
	restartTime  metrics.MetricItem
}

func NewRestartWorker() (spi.Worker, error) {
	c, err := funcNewPsrClient()
	if err != nil {
		return nil, err
	}
	w := worker{
		psrClient:   c,
		log:         vzlog.DefaultLogger(),
		restartData: &restartData{},
		workerMetrics: &workerMetrics{
			restartCount: metrics.MetricItem{
				Name: "opensearch_pod_restart_count",
				Help: "The total number of OpenSearch pod restarts",
				Type: prometheus.CounterValue,
			},
			restartTime: metrics.MetricItem{
				Name: "opensearch_pod_restart_time_nanoseconds",
				Help: "The number of nanoseconds elapsed to restart the OpenSearch pod",
				Type: prometheus.GaugeValue,
			},
		},
	}

	// add the worker config
	if err = config.PsrEnv.LoadFromEnv(w.GetEnvDescList()); err != nil {
		return w, err
	}

	tier, err := psropensearch.ValidateOpenSeachTier(openSearchTier)
	if err != nil {
		return w, err
	}

	workerType := config.PsrEnv.GetEnv(config.PsrWorkerType)

	metricsLabels := map[string]string{
		openSearchTierMetricName:        tier,
		config.PsrWorkerTypeMetricsName: workerType,
	}
	w.restartCount.ConstLabels = metricsLabels
	w.restartTime.ConstLabels = metricsLabels

	w.metricDescList = metrics.BuildMetricDescList([]*metrics.MetricItem{
		&w.restartCount,
		&w.restartTime,
	}, metricsLabels, w.GetWorkerDesc().MetricsName)

	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w worker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		WorkerType:  config.WorkerTypeRestart,
		Description: "Worker to restart pods in the specified OpenSearch tier",
		MetricsName: "restart",
	}
}

func (w worker) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{
		{Key: openSearchTier, DefaultVal: "", Required: true},
	}
}

func (w worker) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w worker) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.restartCount.BuildMetric(),
		w.restartTime.BuildMetric(),
	}
}

func (w worker) WantLoopInfoLogged() bool {
	return false
}

// DoWork restarts a pod in the specified OpenSearch tier
func (w worker) DoWork(_ config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	// validate OS tier
	tier, err := psropensearch.ValidateOpenSeachTier(openSearchTier)
	if err != nil {
		return err
	}

	// Wait for restarted pod to be ready
	if err = w.podsReady(tier); err != nil {
		return err
	}

	// Update the elapsed time of the restart operation
	if w.restartStartTime > 0 {
		atomic.StoreInt64(&w.workerMetrics.restartTime.Val, time.Now().UnixNano()-w.restartStartTime)
	}

	w.restartStartTime = time.Now().UnixNano()
	if err = w.restartPod(tier); err != nil {
		// reset restartStartTime to 0 so we don't emit a bogus metric on the next time through
		w.restartStartTime = 0
		return err
	}
	atomic.AddInt64(&w.workerMetrics.restartCount.Val, 1)

	return nil
}

func (w worker) PreconditionsMet() (bool, error) {
	return true, nil
}

func (w worker) podsReady(tier string) error {
	var label string
	var err error
	switch tier {
	case psropensearch.MasterTier:
		//err = ready.StatefulSetsAreAvailable(w.psrClient.CrtlRuntime, []types.NamespacedName{{
		//	Name:      "vmi-system-es-master",
		//	Namespace: constants.VerrazzanoSystemNamespace,
		//}})

		// there's no opensearch.verrazzano.io/role-master label on the master statefulset
		// however master is the only tier that's deployed as a statefulset, so any opensearch sts must be master
		label = "verrazzano-component"
		err = ready.StatefulSetsAreAvailableBySelector(w.psrClient.CrtlRuntime, getSelectortForLabel(label, "opensearch"))
	case psropensearch.DataTier:
		label = "opensearch.verrazzano.io/role-data"
		err = ready.DeploymentsAreAvailableBySelector(w.psrClient.CrtlRuntime, getSelectortForLabel(label, "true"))
	case psropensearch.IngestTier:
		label = "opensearch.verrazzano.io/role-ingest"
		err = ready.DeploymentsAreAvailableBySelector(w.psrClient.CrtlRuntime, getSelectortForLabel(label, "true"))
	}
	if err != nil {
		return err
	}
	pods, err := psropensearch.GetPodsForTier(w.psrClient.CrtlRuntime, tier)
	if err != nil {
		return err
	}
	for _, pod := range pods {
		if pod.GetUID() == w.restartedPodUID {
			return fmt.Errorf("restarted pod still found in cluster, requeuing")
		}
	}
	return nil
}

func (w worker) restartPod(tier string) error {
	pods, err := psropensearch.GetPodsForTier(w.psrClient.CrtlRuntime, tier)
	if err != nil {
		return err
	}
	if len(pods) == 0 {
		return fmt.Errorf("Failed, no pods found for tier %s", tier)
	}
	i, err := rand.Int(rand.Reader, big.NewInt(int64(len(pods))))
	if err != nil {
		return err
	}
	w.restartedPodUID = pods[i.Int64()].UID
	return w.psrClient.CrtlRuntime.Delete(context.TODO(), &pods[i.Int64()])
}

func getSelectortForLabel(key, val string) []client.ListOption {
	req, _ := labels.NewRequirement(key, selection.Equals, []string{val})
	selector := labels.NewSelector().Add(*req)
	return []client.ListOption{&client.ListOptions{
		Namespace:     constants.VerrazzanoSystemNamespace,
		LabelSelector: selector,
	}}
}
