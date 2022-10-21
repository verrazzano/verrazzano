// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"sync/atomic"
)

const (
	psrMsgSize = "PSR_MSG_SIZE"
)

type logGenerator struct {
	loggedLinesTotal int64
	loggedCharsTotal int64
	logMsg           string
	logMsgSize       int
}

var _ spi.Worker = logGenerator{}

func NewLogGenerator() spi.Worker {
	return logGenerator{
		logMsg: "Log Generator doing work",
	}
}

func (w logGenerator) GetEnvDescList() []config.EnvVarDesc {
	return []config.EnvVarDesc{
		{Key: psrMsgSize, DefaultVal: "20", Required: false},
	}
}

func (w logGenerator) WantIterationInfoLogged() bool {
	return false
}

func (w logGenerator) Init(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	return nil
}

func (w logGenerator) Work(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	if w.logMsgSize == 0 {
		w.logMsgSize = len(w.logMsg)
	}
	log.Infof(w.logMsg)
	atomic.AddInt64(&w.loggedCharsTotal, int64(w.logMsgSize))
	atomic.AddInt64(&w.loggedLinesTotal, 1)
	return nil
}

func (w logGenerator) GetMetricDescList() []prometheus.Desc {
	return nil
}

func (w logGenerator) GetMetricList() []prometheus.Metric {
	return nil
}
