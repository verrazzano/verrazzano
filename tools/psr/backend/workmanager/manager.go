// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workmanager

import (
	"fmt"
	"sync"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	metrics2 "github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/example"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/http/get"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/opensearch/getlogs"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/opensearch/postlogs"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/opensearch/scale"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workers/opensearch/writelogs"
)

var startMetricsFunc = metrics2.StartMetricsServerOrDie

// StartWorkerRunners starts the runner threads, each which runs a worker in a loop
func StartWorkerRunners(log vzlog.VerrazzanoLogger) error {
	// Get the common config for all the workers
	conf, err := config.GetCommonConfig(log)
	if err != nil {
		log.Error(err)
		return err
	}

	// get the worker type
	wt := conf.WorkerType
	worker, err := getWorker(wt)
	if err != nil {
		log.Error(err)
		return err
	}
	// add the worker config
	if err := config.PsrEnv.LoadFromEnv(worker.GetEnvDescList()); err != nil {
		log.Error(err)
		return err
	}

	// init the runner with the worker that it will call repeatedly to DoWork
	log.Infof("Initializing worker %s", wt)
	runner, err := NewRunner(worker, conf, log)
	if err != nil {
		log.Errorf("Failed initializing runner and worker: %v", err)
		return err
	}

	// start metrics server as go routine
	log.Info("Starting metrics server")
	mProviders := []spi.WorkerMetricsProvider{}
	mProviders = append(mProviders, runner)
	mProviders = append(mProviders, worker)
	go startMetricsFunc(mProviders)

	// run the worker in go-routine to completion (usually forever)
	var wg sync.WaitGroup
	for i := 1; i <= conf.WorkerThreadCount; i++ {
		wg.Add(1)
		log.Infof("Running worker %s in thread %v", wt, i)
		go func() {
			defer wg.Done()
			runner.RunWorker(conf, log)
		}()
	}
	wg.Wait()
	return nil
}

// getWorker returns a worker given the name of the worker
func getWorker(wt string) (spi.Worker, error) {
	switch wt {
	case config.WorkerTypeExample:
		return example.NewExampleWorker()
	case config.WorkerTypeHTTPGet:
		return http.NewHTTPGetWorker()
	case config.WorkerTypeWriteLogs:
		return writelogs.NewWriteLogsWorker()
	case config.WorkerTypeGetLogs:
		return getlogs.NewGetLogsWorker()
	case config.WorkerTypePostLogs:
		return postlogs.NewPostLogsWorker()
	case config.WorkerTypeScale:
		return scale.NewScaleWorker()
	default:
		return nil, fmt.Errorf("Failed, invalid worker type '%s'", wt)
	}
}
