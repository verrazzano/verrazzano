// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package logget

import (
	"bytes"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"io"
	"net/http"
	"net/url"
)

const osIngestService = "vmi-system-es-ingest.verrazzano-system:9200"

var bodyString = "{\"query\":{\"bool\"{\"filter\":[{\"match_phrase\":{\"kubernetes.container_name\":\"istio-proxy\"}}]}}}"
var body = io.NopCloser(bytes.NewBuffer([]byte(bodyString)))

type LogGetter struct {
	spi.Worker
}

var _ spi.Worker = LogGetter{}

func NewLogGetter() (spi.Worker, error) {
	return LogGetter{}, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w LogGetter) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		EnvName:     config.WorkerTypeLogGet,
		Description: "The log getter worker performs GET requests on the OpenSearch endpoint",
		MetricsName: "LogGet",
	}
}

func (w LogGetter) GetEnvDescList() []config.EnvVarDesc {
	return []config.EnvVarDesc{}
}

func (w LogGetter) WantIterationInfoLogged() bool {
	return false
}

func (w LogGetter) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	c := http.Client{}
	req := http.Request{
		URL: &url.URL{
			Scheme: "http",
			Host:   osIngestService,
			//Path:   "verrazzano-system",
		},
		Header: http.Header{"Content-Type": {"application/json"}},
		Body:   body,
	}
	_, err := c.Do(&req)
	if err != nil {
		//respBody := []byte("")
		//if resp.Body != nil {
		//	respBody, _ = io.ReadAll(resp.Body)
		//}
		//return fmt.Errorf("OpenSearch GET request failed, status code: %d, status %s, body: %s, error: %v", resp.StatusCode, resp.Status, string(respBody), err)
		log.Error(err)
		return err
	}
	log.Info("OpenSearch GET request successful")
	return nil
}

func (w LogGetter) GetMetricDescList() []prometheus.Desc {
	return []prometheus.Desc{}
}

func (w LogGetter) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{}
}
