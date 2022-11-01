// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workmanager

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	metrics2 "github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/example"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/opensearch/getlogs"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/opensearch/writelogs"
	"os"
	"sync"
)

// StartWorkerRunners starts the runner threads, each which runs a worker in a loop
func StartWorkerRunners(log vzlog.VerrazzanoLogger) error {
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
	// add the worker config
	if err := config.PsrEnv.LoadFromEnv(worker.GetEnvDescList()); err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// init the runner with the worker that it will call repeatedly to DoWork
	log.Infof("Initializing worker %s", wt)
	runner, err := NewRunner(worker, conf, log)
	if err != nil {
		log.Errorf("Failed initializing runner and worker: %v", err)
		os.Exit(1)
	}

	// start metrics server as go routine
	log.Info("Starting metrics server")
	mProviders := []spi.WorkerMetricsProvider{}
	mProviders = append(mProviders, runner)
	mProviders = append(mProviders, worker)
	go metrics2.StartMetricsServerOrDie(mProviders)

	// run the worker in go-routine to completion (usually forever)
	var wg sync.WaitGroup
	for i := 1; i <= conf.WorkerThreadCount; i++ {
		wg.Add(1)
		log.Infof("Running worker %s in thread %v", wt, i)
		go func() {
			runner.RunWorker(conf, log)
		}()
	}
	wg.Wait()
	return nil
}

// getWorker returns a worker given the	 name of the worker
func getWorker(wt string) (spi.Worker, error) {
	switch wt {
	case config.WorkerTypeExample:
		return example.NewExampleWorker()
	case config.WorkerTypeWriteLogs:
		return writelogs.NewWriteLogsWorker()
	case config.WorkerTypeGetLogs:
		return getlogs.NewGetLogsWorker()
	default:
		return nil, fmt.Errorf("Failed, invalid worker type '%s'", wt)
	}
}
