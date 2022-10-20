// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"time"
)

const (
	psrMsgSize = "PSR_MSG_SIZE"
)

type LogGenerator struct{}

var _ spi.Worker = LogGenerator{}

func (w LogGenerator) GetEnvDescList() []config.EnvVarDesc {
	return []config.EnvVarDesc{
		{Key: psrMsgSize, DefaultVal: "20", Required: false},
	}
}

func (w LogGenerator) Work(conf config.CommonConfig, log vzlog.VerrazzanoLogger) {
	for {
		log.Infof("Log Generator Doing Work")
		time.Sleep(conf.IterationSleepNanos)
	}

}

func (w LogGenerator) GetMetricDescList() []prometheus.Desc {
	return nil
}

func (w LogGenerator) GetMetricList() []prometheus.Metric {
	return nil
}
