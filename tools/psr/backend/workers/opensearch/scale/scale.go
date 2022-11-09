// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scale

import (
	"fmt"
	update "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/opensearch"
	psrvz "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/verrazzano"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
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
	// validate OS tier
	tier := config.PsrEnv.GetEnv(openSearchTier)
	if tier != masterTier && tier != dataTier && tier != ingestTier {
		return fmt.Errorf("error, not a valid OpenSearch tier to scale")
	}

	// Wait until VZ is ready
	cr, err := waitReady(log, true)
	if err != nil {
		log.Progress("Failed to wait for Verrazzano to be ready after update.  The test results are not valid %v", err)
		return err
	}

	// Create a modifier that is used to update the Verrazzno CR opensearch replica field
	m, desiredReplicas, err := getUpdateModifier(cr, tier)
	if err != nil {
		return err
	}

	log.Infof("Updating Verrazzano CR OpenSearch %s tier, scaling to %v replicas", tier, desiredReplicas)

	// Update the CR to change the replica count
	err = updateCr(log, w.ctrlRuntimeClient, m)
	if err != nil {
		return err
	}

	// Wait until VZ is NOT ready, this means it started working on the change
	_, err = waitReady(log, false)
	if err != nil {
		log.Progress("Failed to wait for Verrazzano to be NOT ready after update.  The test results are not valid %v", err)
		return err
	}
	//
	//// Wait until OpenSearch is NOT ready
	//_, err = waitOpenSearchReady(log, w.ctrlRuntimeClient, false)
	//if err != nil {
	//	log.Progress("Failed to wait for OpenSearch to be NOT ready after update.  The test results are not valid %v", err)
	//	return err
	//}

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

func getUpdateModifier(cr *vzv1alpha1.Verrazzano, tier string) (*update.CRModifier, int, error) {
	// Get the current opensearch replica field for the tier in the VZ CR
	currentReplicas := getCurrentReplicas(cr, tier)

	max, err := strconv.Atoi(config.PsrEnv.GetEnv(maxReplicaCount))
	if err != nil {
		return nil, 0, fmt.Errorf("maxReplicaCount can not be parsed to an integer: %f", err)
	}
	min, err := strconv.Atoi(config.PsrEnv.GetEnv(minReplicaCount))
	if err != nil {
		return nil, 0, fmt.Errorf("minReplicaCount can not be parsed to an integer: %f", err)
	}
	if min < 1 {
		return nil, 0, fmt.Errorf("minReplicaCount can not be less than 1")
	}
	var desiredReplicas int32
	if currentReplicas == min {
		desiredReplicas = int32(max)
	} else {
		desiredReplicas = int32(min)
	}

	var m update.CRModifier

	switch tier {
	case masterTier:
		//if len(cr.Spec.Components.Elasticsearch.Nodes) == 1 {
		//	m = update.OpensearchAllNodeRolesModifier{NodeReplicas: desiredReplicas}
		//} else {
		//	m = update.OpensearchMasterNodeGroupModifier{NodeReplicas: desiredReplicas}
		//}
		m = update.OpensearchMasterNodeGroupModifier{NodeReplicas: desiredReplicas}
	case dataTier:
		m = update.OpensearchDataNodeGroupModifier{NodeReplicas: desiredReplicas}
	case ingestTier:
		m = update.OpensearchIngestNodeGroupModifier{NodeReplicas: desiredReplicas}
	}
	return &m, int(desiredReplicas), nil
}

// Get current replicas in the VZ CR elasticesearch component for the current tier
func getCurrentReplicas(cr *vzv1alpha1.Verrazzano, tier string) int {
	if cr.Spec.Components.Elasticsearch == nil || cr.Spec.Components.Elasticsearch.Nodes == nil {
		return 1
	}

	for _, node := range cr.Spec.Components.Elasticsearch.Nodes {
		for _, nodeRole := range node.Roles {
			if string(nodeRole) == tier {
				return int(node.Replicas)
			}
		}
	}
	return 1
}

// updateCr updates the Verrazzano CR and retries if there is a conflict error
func updateCr(log vzlog.VerrazzanoLogger, ctrlRuntimeClient client.Client, m *update.CRModifier) error {
	for {
		err := psrvz.UpdateCR(ctrlRuntimeClient, *m)
		if err != nil {
			if er.IsUpdateConflict(err) {
				time.Sleep(3 * time.Second)
				logMsg := fmt.Sprintf("VZ conflict error, retrying")
				log.Infof(logMsg)
				continue
			} else {
				return fmt.Errorf("Failed to scale update Verrazzano cr: %v", err)
			}
		}
		break
	}
	return nil
}

// Wait until Verrazzano is ready or not ready
func waitReady(log vzlog.VerrazzanoLogger, desiredReady bool) (cr *vzv1alpha1.Verrazzano, err error) {
	for {
		cr, err = psrvz.GetVzCr()
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
		log.Progressf("Waiting for Verrazzano CR ready state to be %v", desiredReady)
		time.Sleep(3 * time.Second)
	}
	return cr, err
}

// waitOpenSearchReady waits until OpenSearch is ready or not ready
func waitOpenSearchReady(log vzlog.VerrazzanoLogger, ctrlRuntimeClient client.Client, desiredReady bool) (cr *vzv1alpha1.Verrazzano, err error) {
	for {
		cr, err = psrvz.GetVzCr()
		if err != nil {
			return nil, err
		}
		ready := update.IsOSReady(ctrlRuntimeClient, cr)
		if err != nil {
			return nil, err
		}
		if ready == desiredReady {
			break
		}
		log.Progressf("Waiting for OpenSearch  ready state to be %v", desiredReady)
		time.Sleep(3 * time.Second)
	}
	return cr, err
}
