// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"strconv"
	"time"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
)

// Define common worker configuration params
const (
	// PsrWorkerType specifies a worker type
	PsrWorkerType = "PSR_WORKER_TYPE"

	// PsrDuration specified the duration of the test using a duration string ("4m or 2h")
	// By default, the worker runs until the pod terminates
	PsrDuration = "PSR_DURATION"

	// PsrLoopSleep specified the sleep duration between loops
	// of work actions using a duration string ("4m or 2h")
	// For example, delay 1 second between logging
	// By default, the worker does not delay
	PsrLoopSleep = "PSR_LOOP_SLEEP"

	// PsrNumLoops specifies the number of loops
	// of work actions.  The default is -1 (forever)
	// By default, the worker iterates forever
	PsrNumLoops = "PSR_NUM_ITERATIONS"

	// PsrWorkerThreadCount specifies the number of worker threads to run.
	// By default, there is one thread per worker
	PsrWorkerThreadCount = "PSR_WORKER_THREAD_COUNT"

	// PsrWorkerNamespace is the namespace of the PSR release
	PsrWorkerNamespace = "NAMESPACE"

	// PsrWorkerReleaseName is the name of the PSR release
	PsrWorkerReleaseName = "RELEASE_NAME"

	// PsrWorkerTypeMetricsName is the metrics label key for the PSR worker type
	PsrWorkerTypeMetricsName = "psr_worker_type"
)

// Define worker types
const (
	WorkerTypeExample       = "example"
	WorkerTypeOpsWriteLogs  = "ops-writelogs"
	WorkerTypeOpsGetLogs    = "ops-getlogs"
	WorkerTypeOpsPostLogs   = "ops-postlogs"
	WorkerTypeOpsScale      = "ops-scale"
	WorkerTypeOpsRestart    = "ops-restart"
	WorkerTypeHTTPGet       = "http-get"
	WorkerTypeReceiveAlerts = "prom-alerts"
	WorkerTypeWlsScale      = "wls-scale"
	WorkerTypeWlsTodoDelete = "wls-todo-delete"
	WorkerTypeWlsTodoPut    = "wls-todo-put"
)

const (
	UnlimitedWorkerLoops    = -1
	UnlimitedWorkerDuration = -1 * time.Second
)

var PsrEnv = osenv.NewEnv()

type CommonConfig struct {
	WorkerType        string
	PsrDuration       time.Duration
	LoopSleepNanos    time.Duration
	NumLoops          int64
	WorkerThreadCount int
	Namespace         string
	ReleaseName       string
}

// GetCommonConfig loads the common config from env vars
func GetCommonConfig(log vzlog.VerrazzanoLogger) (CommonConfig, error) {
	dd := []osenv.EnvVarDesc{
		{Key: PsrWorkerType, DefaultVal: "", Required: true},
		{Key: PsrDuration, DefaultVal: "-1s", Required: false},
		{Key: PsrLoopSleep, DefaultVal: "1s", Required: false},
		{Key: PsrNumLoops, DefaultVal: "-1", Required: false},
		{Key: PsrWorkerThreadCount, DefaultVal: "1", Required: false},
		{Key: PsrWorkerNamespace, DefaultVal: "", Required: false},
		{Key: PsrWorkerReleaseName, DefaultVal: "", Required: false},
	}
	if err := PsrEnv.LoadFromEnv(dd); err != nil {
		return CommonConfig{}, err
	}

	duration, err := time.ParseDuration(PsrEnv.GetEnv(PsrDuration))
	if err != nil {
		return CommonConfig{}, log.ErrorfNewErr("Error parsing worker duration: %v", err)
	}
	// Negative values are not allowed, -1s represents no duration timeout
	if duration < 0 {
		duration = UnlimitedWorkerDuration
	}

	sleepDuration, err := time.ParseDuration(PsrEnv.GetEnv(PsrLoopSleep))
	if err != nil {
		return CommonConfig{}, log.ErrorfNewErr("Error parsing loop sleep duration: %v", err)
	}
	// Sleep at least 10 nanos
	if sleepDuration < (10 * time.Nanosecond) {
		sleepDuration = 10 * time.Nanosecond
	}

	threadCount, err := strconv.Atoi(PsrEnv.GetEnv(PsrWorkerThreadCount))
	if err != nil {
		return CommonConfig{}, log.ErrorfNewErr("Error parsing worker thread count: %v", err)
	}
	// Max thread count is 100
	if threadCount > 100 {
		threadCount = 100
	}

	numLoops, err := strconv.Atoi(PsrEnv.GetEnv(PsrNumLoops))
	if err != nil {
		return CommonConfig{}, log.ErrorfNewErr("Failed to convert ENV var %s to integer", PsrNumLoops)
	}

	return CommonConfig{
		WorkerType:        PsrEnv.GetEnv(PsrWorkerType),
		PsrDuration:       duration,
		LoopSleepNanos:    sleepDuration,
		NumLoops:          int64(numLoops),
		WorkerThreadCount: threadCount,
		Namespace:         PsrEnv.GetEnv(PsrWorkerNamespace),
		ReleaseName:       PsrEnv.GetEnv(PsrWorkerReleaseName),
	}, nil
}
