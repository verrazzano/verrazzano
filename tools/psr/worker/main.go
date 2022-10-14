// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package worker

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/psr/worker/config"
	"github.com/verrazzano/verrazzano/tools/psr/worker/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/worker/opensearch"
	"go.uber.org/zap"
	"os"
)

func main() {
	log := zap.S()

	if err := config.LoadCommonConfig(); err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// Configure the worker
	wt := getWorkerType()
	if len(wt) == 0 {
		log.Errorf("Failed, missing Env var PSR_WORKER_TYPE")
		os.Exit(1)
	}
	worker, err := getWorker(wt)
	if err == nil {
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
	log.Info("Starting worker %s", wt)
	worker.Work(config.Config)

}

func getWorker(wt string) (Worker, error) {
	switch config.PsrWorkerType {
	case config.WorkerTypeLogGen:
		return opensearch.LogGenerator{}, nil
	default:
		return nil, fmt.Errorf("Failed, invalid worker type %s", wt)
	}
}
