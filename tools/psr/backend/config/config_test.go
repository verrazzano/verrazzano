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

// TestWorkerType tests the Config interface
// GIVEN a config map with environment vars related to worker type
//
//	WHEN the GetCommonConfig is called
//	THEN ensure that the resulting configuration is correct
func TestWorkerType(t *testing.T) {
	var tests = []struct {
		name       string
		envMap     map[string]string
		envKey     string
		workerType string
		expectErr  bool
	}{
		{name: "ExampleWorker",
			workerType: WorkerTypeExample,
			expectErr:  false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeExample,
			},
		},
		{name: "GetLogs",
			workerType: WorkerTypeGetLogs,
			expectErr:  false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeGetLogs,
			},
		},
		{name: "ExampleWorkerType",
			workerType: WorkerTypeWriteLogs,
			expectErr:  false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeWriteLogs,
			},
		},
		{name: "MissingWorkerType",
			workerType: WorkerTypeWriteLogs,
			expectErr:  true,
			envMap:     map[string]string{},
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
			}
		})
	}
}

// TestLoopSleep tests the Config interface
// GIVEN a config map with environment vars loop sleep
//
//	WHEN the GetCommonConfig is called
//	THEN ensure that the resulting configuration is correct
func TestLoopSleep(t *testing.T) {
	var tests = []struct {
		name      string
		envMap    map[string]string
		envKey    string
		loopSleep time.Duration
		expectErr bool
	}{
		{name: "DefaultSleep",
			loopSleep: time.Second,
			expectErr: false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeWriteLogs,
			},
		},
		{name: "TenMilliSleep",
			loopSleep: 10 * time.Millisecond,
			expectErr: false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeWriteLogs,
				PsrLoopSleep:  "10ms",
			},
		},
		{name: "TenNanoSleep",
			loopSleep: 10 * time.Nanosecond,
			expectErr: false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeWriteLogs,
				PsrLoopSleep:  "10ns",
			},
		},
		// Test min sleep of 10ns
		{name: "TenNanoSleepMin",
			loopSleep: 10 * time.Nanosecond,
			expectErr: false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeWriteLogs,
				PsrLoopSleep:  "1ns",
			},
		},
		{name: "BadNumericStringFormat",
			expectErr: true,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeWriteLogs,
				PsrLoopSleep:  "10xyz",
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
				assert.Equal(t, test.loopSleep, cc.LoopSleepNanos)
			}
		})
	}
}

// TestThreadCount tests the Config interface
// GIVEN a config map with environment vars related to thread count
//
//	WHEN the GetCommonConfig is called
//	THEN ensure that the resulting configuration is correct
func TestThreadCount(t *testing.T) {
	var tests = []struct {
		name          string
		envMap        map[string]string
		envKey        string
		workerThreads int
		expectErr     bool
	}{
		{name: "DefaultWorkerThreads",
			workerThreads: 1,
			expectErr:     false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeWriteLogs,
			},
		},
		{name: "MultipleWorkerThreads",
			workerThreads: 50,
			expectErr:     false,
			envMap: map[string]string{
				PsrWorkerType:        WorkerTypeWriteLogs,
				PsrWorkerThreadCount: "50",
			},
		},
		// Test max threads 100
		{name: "MaxWorkerThreads",
			workerThreads: 100,
			expectErr:     false,
			envMap: map[string]string{
				PsrWorkerType:        WorkerTypeWriteLogs,
				PsrWorkerThreadCount: "1000",
			},
		},
		{name: "BadThreadCountFormat",
			expectErr: true,
			envMap: map[string]string{
				PsrWorkerType:        WorkerTypeWriteLogs,
				PsrWorkerThreadCount: "100n",
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
				assert.Equal(t, test.workerThreads, cc.WorkerThreadCount)
			}
		})
	}
}

func (f *fakeEnv) GetEnv(key string) string {
	return f.data[key]
}
