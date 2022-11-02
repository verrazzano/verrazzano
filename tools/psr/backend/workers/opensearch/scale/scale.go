// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scale

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
)

const (
	totalScaleInCount     = "total_scale_in_count"
	totalScaleInCountHelp = "The total number of times OpenSearch has been scaled in"

	totalScaleOutCount     = "total_scale_out_count"
	totalScaleOutCountHelp = "The total number of times OpenSearch has been scaled out"
)

type scaleWorker struct {
	spi.Worker
	metricDescList []prometheus.Desc
	*scaleMetrics
}

var _ spi.Worker = scaleWorker{}

// scaleMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type scaleMetrics struct {
	totalScaleInCount           metrics.MetricItem
	totalScaleOutCount          metrics.MetricItem
	totalGetRequestsFailedCount metrics.MetricItem
}

func NewScaleWorker() (spi.Worker, error) {
	return nil, nil
}

func (w scaleWorker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	return nil
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
	return []osenv.EnvVarDesc{}
}

func (w scaleWorker) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w scaleWorker) GetMetricList() []prometheus.Metric {
	return nil
}

func (w scaleWorker) WantIterationInfoLogged() bool {
	return false
}
