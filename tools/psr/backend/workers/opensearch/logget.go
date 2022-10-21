// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"net/http"
	"net/url"
)

const osIngestService = "vmi-system-es-ingest.verrazzano-system:9200"

type logGetter struct{}

var _ spi.Worker = logGetter{}

func NewLogGetter() spi.Worker {
	return logGetter{}
}

func (w logGetter) GetEnvDescList() []config.EnvVarDesc {
	return []config.EnvVarDesc{
		{Key: psrMsgSize, DefaultVal: "20", Required: false},
	}
}

func (w logGetter) WantIterationInfoLogged() bool {
	return true
}

func (w logGetter) Work(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	c := http.Client{}
	req := http.Request{
		URL: &url.URL{
			Scheme: "http",
			Host:   osIngestService,
		},
	}
	_, err := c.Do(&req)
	if err == nil {
		log.Info("OpenSearch GET request successful")
	}
	return err
}

func (w logGetter) GetMetricDescList() []prometheus.Desc {
	return nil
}

func (w logGetter) GetMetricList() []prometheus.Metric {
	return nil
}
