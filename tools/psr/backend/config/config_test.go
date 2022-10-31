// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"testing"
	"time"
)

type fakeEnv struct {
	data map[string]string
}

// TestConfig tests the Config interface
// GIVEN a config map with environment vars
//
//	WHEN the GetCommonConfig is called
//	THEN ensure that the resulting configuration is correct
func TestConfig(t *testing.T) {
	var tests = []struct {
		name           string
		envMap         map[string]string
		envKey         string
		workerType     string
		iterationSleep time.Duration
		expectErr      bool
	}{
		{name: "ExampleWorker",
			workerType:     WorkerTypeExample,
			iterationSleep: time.Second,
			expectErr:      false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeExample,
			},
		},
		{name: "GetLogs",
			workerType:     WorkerTypeGetLogs,
			iterationSleep: time.Second,
			expectErr:      false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeGetLogs,
			},
		},
		{name: "ExampleWorkerType",
			workerType:     WorkerTypeWriteLogs,
			iterationSleep: time.Second,
			expectErr:      false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeWriteLogs,
			},
		},
		{name: "MissingWorkerType",
			workerType:     WorkerTypeWriteLogs,
			iterationSleep: time.Second,
			expectErr:      true,
			envMap:         map[string]string{},
		},
		{name: "DefaultSleep",
			workerType:     WorkerTypeWriteLogs,
			iterationSleep: time.Second,
			expectErr:      false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeWriteLogs,
			},
		},
		{name: "TenMilliSleep",
			workerType:     WorkerTypeWriteLogs,
			iterationSleep: 10 * time.Millisecond,
			expectErr:      false,
			envMap: map[string]string{
				PsrWorkerType:     WorkerTypeWriteLogs,
				PsrIterationSleep: "10ms",
			},
		},
		{name: "TenNanoSleep",
			workerType:     WorkerTypeWriteLogs,
			iterationSleep: 10 * time.Nanosecond,
			expectErr:      false,
			envMap: map[string]string{
				PsrWorkerType:     WorkerTypeWriteLogs,
				PsrIterationSleep: "10ns",
			},
		},
		// Test min sleep of 10ns
		{name: "TenNanoSleepMin",
			workerType:     WorkerTypeWriteLogs,
			iterationSleep: 10 * time.Nanosecond,
			expectErr:      false,
			envMap: map[string]string{
				PsrWorkerType:     WorkerTypeWriteLogs,
				PsrIterationSleep: "1ns",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Load the fake env
			f := fakeEnv{data: test.envMap}
			saveEnv := osenv.GetEnvFunc
			osenv.GetEnvFunc = f.GetEnv
			defer func() {
				osenv.GetEnvFunc = saveEnv
			}()

			cc, err := GetCommonConfig(vzlog.DefaultLogger())
			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.workerType, cc.WorkerType)
				assert.Equal(t, test.iterationSleep, cc.IterationSleepNanos)
			}
		})
	}
}

func (f *fakeEnv) GetEnv(key string) string {
	return f.data[key]
}
