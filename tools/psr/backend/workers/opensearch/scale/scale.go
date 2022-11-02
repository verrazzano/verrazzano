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
	scaleInCountTotal     = "total_scale_in_count"
	scaleInCountTotalHelp = "The total number of times OpenSearch has been scaled in"

	scaleOutCountTotal     = "total_scale_out_count"
	scaleOutCountTotalHelp = "The total number of times OpenSearch has been scaled out"
)

type scaleWorker struct {
	spi.Worker
	metricDescList []prometheus.Desc
	*workerMetrics
}

var _ spi.Worker = scaleWorker{}

// scaleMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	scaleInCountTotal  metrics.MetricItem
	scaleOutCountTotal metrics.MetricItem
}

func NewScaleWorker() (spi.Worker, error) {

	constLabels := prometheus.Labels{}

	w := scaleWorker{workerMetrics: &workerMetrics{}}

	d := prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, scaleInCountTotal),
		scaleInCountTotalHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.workerMetrics.scaleInCountTotal.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, scaleOutCountTotal),
		scaleOutCountTotalHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.workerMetrics.scaleOutCountTotal.Desc = d

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
