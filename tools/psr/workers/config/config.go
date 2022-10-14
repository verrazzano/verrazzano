// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"fmt"
	"os"
)

// Define common worker configuration params
const (
	// PsrWorkerType specifies a worker type
	PsrWorkerType = "PSR_WORKER_TYPE"

	// PsrDuration specified the duration of the test using a duration string ("4m or 2h")
	// By default the worker runs until the pod terminates
	PsrDuration = "PSR_DURATION"

	// PsrIterationDelay specified the delay between iterations of work actions using a duration string ("4m or 2h")
	// For example, delay 1 second between logging
	// By default the worker does not delay
	PsrIterationDelay = "PSR_ITERATION_DELAY"
)

// Define worker types
const (
	WorkerTypeLogGen        = "WT_LOG_GEN"
	WorkerTypeLogGeT        = "WT_LOG_GET"
	WorkerTypePodTerminate  = "WT_POD_TERMINATE"
	WorkerTypeWorkloadScale = "WT_WORKLOAD_SCALE"
)

type ConfigItem struct {
	Key        string
	DefaultVal string
	Required   bool
}

var Config = make(map[string]string)

// LoadCommonConfig loads the common config from env vars
func LoadCommonConfig() error {
	cc := []ConfigItem{
		{PsrWorkerType, "", true},
		{PsrDuration, "", false},
		{PsrIterationDelay, "1s", false},
	}

	if err := AddConfigItems(cc); err != nil {
		return err
	}
	return nil
}

func GetWorkerType() string {
	return Config[PsrWorkerType]
}

// AddConfigItems adds items to the config
func AddConfigItems(cc []ConfigItem) error {
	for _, c := range cc {
		if err := addItemConfig(c); err != nil {
			return err
		}
	}
	return nil
}

func addItemConfig(c ConfigItem) error {
	val := os.Getenv(c.Key)
	if len(val) == 0 {
		if c.Required {
			return fmt.Errorf("Failed, missing required Env var %s", c.Key)
		}
		val = c.DefaultVal
	}
	Config[c.Key] = val
	return nil
}
