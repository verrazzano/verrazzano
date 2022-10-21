// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workmanager

import (
	"fmt"
	"os"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	metrics2 "github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/example"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/opensearch"
)

// RunWorker runs a worker to completion
func RunWorker(log vzlog.VerrazzanoLogger) error {
	// Get the common config for all the workers
	conf, err := config.GetCommonConfig(log)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// get the worker type
	wt := conf.WorkerType
	if len(wt) == 0 {
		log.Errorf("Failed, missing Env var PSR_WORKER_TYPE")
		os.Exit(1)
	}
	worker, err := getWorker(wt)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	// Add the worker config
	if err := config.AddEnvConfig(worker.GetEnvDescList()); err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// Start metrics server as go routine
	log.Info("Starting metrics server")
	go metrics2.StartMetricsServerOrDie()

	// Run the worker to completion (usually forever)
	log.Infof("Running worker %s", wt)
	return Runner{Worker: worker}.RunWorker(conf, log)
}

// getWorker returns a worker given the name of the worker
func getWorker(wt string) (spi.Worker, error) {
	switch wt {
	case config.WorkerTypeExample:
		return example.NewExampleWorker(), nil
	case config.WorkerTypeLogGen:
		return opensearch.NewLogGenerator(), nil
	case config.WorkerTypeLogGet:
		return opensearch.NewLogGetter(), nil
	default:
		return nil, fmt.Errorf("Failed, invalid worker type '%s'", wt)
	}
}
