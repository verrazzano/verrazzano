// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"time"
)

// Define common worker configuration params
const (
	// PsrWorkerType specifies a worker type
	PsrWorkerType = "PSR_WORKER_TYPE"

	// PsrDuration specified the duration of the test using a duration string ("4m or 2h")
	// By default the worker runs until the pod terminates
	PsrDuration = "PSR_DURATION"

	// PsrIterationSleep specified the sleep duration between iterations
	// of work actions using a duration string ("4m or 2h")
	// For example, delay 1 second between logging
	// By default the worker does not delay
	PsrIterationSleep = "PSR_ITERATION_SLEEP"
)

// Define worker types
const (
	WorkerTypeExample       = "WT_EXAMPLE"
	WorkerTypeLogGen        = "WT_LOG_GEN"
	WorkerTypeLogGet        = "WT_LOG_GET"
	WorkerTypePodTerminate  = "WT_POD_TERMINATE"
	WorkerTypeWorkloadScale = "WT_WORKLOAD_SCALE"
)

var env = osenv.NewEnv()

type CommonConfig struct {
	WorkerType          string
	IterationSleepNanos time.Duration
}

// GetCommonConfig loads the common config from env vars
func LoadCommonConfig(log vzlog.VerrazzanoLogger) (CommonConfig, error) {
	dd := []osenv.EnvVarDesc{
		{Key: PsrWorkerType, DefaultVal: "", Required: true},
		{Key: PsrDuration, DefaultVal: "", Required: false},
		{Key: PsrIterationSleep, DefaultVal: "1s", Required: false},
	}
	if err := env.LoadFromEnv(dd); err != nil {
		return CommonConfig{}, err
	}
	sleepDuration, err := time.ParseDuration(env.GetEnv(PsrIterationSleep))
	if err != nil {
		return CommonConfig{}, log.ErrorfNewErr("Error parsing iteration sleep duration: %v", err)
	}
	// Sleep at least 10 nanos
	if sleepDuration < (10 * time.Nanosecond) {
		sleepDuration = 10 * time.Nanosecond
	}

	return CommonConfig{
		WorkerType:          env.GetEnv(PsrWorkerType),
		IterationSleepNanos: sleepDuration,
	}, nil
}
