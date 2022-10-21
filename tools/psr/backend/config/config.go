// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"os"
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

type EnvVarDesc struct {
	Key        string
	DefaultVal string
	Required   bool
}

type CommonConfig struct {
	WorkerType          string
	IterationSleepNanos time.Duration
}

var EnvVars = make(map[string]string)

// GetCommonConfig loads the common config from env vars
func GetCommonConfig(log vzlog.VerrazzanoLogger) (CommonConfig, error) {
	dd := []EnvVarDesc{
		{Key: PsrWorkerType, DefaultVal: "", Required: true},
		{Key: PsrDuration, DefaultVal: "", Required: false},
		{Key: PsrIterationSleep, DefaultVal: "1s", Required: false},
	}
	if err := AddEnvConfig(dd); err != nil {
		return CommonConfig{}, err
	}
	sleepDuration, err := time.ParseDuration(EnvVars[PsrIterationSleep])
	if err != nil {
		return CommonConfig{}, log.ErrorfNewErr("Error parsing iteration sleep duration: %v", err)
	}
	// Sleep at least 100 millis
	if sleepDuration < (10 * time.Millisecond) {
		sleepDuration = 10 * time.Millisecond
	}

	return CommonConfig{
		WorkerType:          EnvVars[PsrWorkerType],
		IterationSleepNanos: sleepDuration,
	}, nil
}

// AddEnvConfig adds items to the config
func AddEnvConfig(cc []EnvVarDesc) error {
	for _, c := range cc {
		if err := addItemConfig(c); err != nil {
			return err
		}
	}
	return nil
}

func addItemConfig(c EnvVarDesc) error {
	val := os.Getenv(c.Key)
	if len(val) == 0 {
		if c.Required {
			return fmt.Errorf("Failed, missing required Env var %s", c.Key)
		}
		val = c.DefaultVal
	}
	EnvVars[c.Key] = val
	return nil
}
