// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"fmt"
	vzlog2 "github.com/verrazzano/verrazzano/pkg/log"
	vzlog "github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/opensearch"
	"os"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	vzlog2.InitLogs(kzap.Options{})
	log := vzlog.DefaultLogger()
	log.Info("Starting PSR worker")

	if err := config.LoadCommonConfig(); err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// Configure the worker
	wt := config.GetWorkerType()
	if len(wt) == 0 {
		log.Errorf("Failed, missing Env var PSR_WORKER_TYPE")
		os.Exit(1)
	}
	worker, err := getWorker(wt)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	if err := config.AddConfigItems(worker.GetConfigItems()); err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// Start metrics server as go routine
	log.Info("Starting metrics server")
	go metrics.StartMetricsServerOrDie()

	// Run the worker to completion (usually forever)
	log.Infof("Running worker %s", wt)
	worker.Work(config.Config, log)

	log.Info("Stopping worker")
}

func getWorker(wt string) (spi.Worker, error) {
	switch wt {
	case config.WorkerTypeLogGen:
		return opensearch.LogGenerator{}, nil
	default:
		return nil, fmt.Errorf("Failed, invalid worker type '%s'", wt)
	}
}
