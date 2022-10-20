// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
)

type Worker interface {
	GetMetricDescList() []prometheus.Desc
	GetMetricList() []prometheus.Metric
	GetEnvDescList() []config.EnvVarDesc
	Work(config.CommonConfig, vzlog.VerrazzanoLogger)
}
