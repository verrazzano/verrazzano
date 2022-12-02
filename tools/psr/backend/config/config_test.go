// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
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
			workerType: WorkerTypeOpsGetLogs,
			expectErr:  false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeOpsGetLogs,
			},
		},
		{name: "ExampleWorkerType",
			workerType: WorkerTypeOpsWriteLogs,
			expectErr:  false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeOpsWriteLogs,
			},
		},
		{name: "MissingWorkerType",
			workerType: WorkerTypeOpsWriteLogs,
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

// TestDuration tests the Config interface
// GIVEN a config map with environment vars PsrDuration
//
//	WHEN the GetCommonConfig is called
//	THEN ensure that the resulting configuration is correct
func TestDuration(t *testing.T) {
	var tests = []struct {
		name      string
		envMap    map[string]string
		envKey    string
		duration  time.Duration
		expectErr bool
	}{
		{name: "DefaultSleep",
			duration:  UnlimitedWorkerDuration,
			expectErr: false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeOpsWriteLogs,
			},
		},
		{name: "OneSecDuration",
			duration:  1 * time.Second,
			expectErr: false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeOpsWriteLogs,
				PsrDuration:   "1s",
			},
		},
		{name: "OneMinDuration",
			duration:  1 * time.Minute,
			expectErr: false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeOpsWriteLogs,
				PsrDuration:   "1m",
			},
		},
		{name: "NegativeDuration",
			duration:  UnlimitedWorkerDuration,
			expectErr: false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeOpsWriteLogs,
				PsrDuration:   "-3s",
			},
		},
		{name: "BadNumericStringFormat",
			expectErr: true,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeOpsWriteLogs,
				PsrDuration:   "10xyz",
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
				assert.Equal(t, test.duration, cc.PsrDuration)
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
				PsrWorkerType: WorkerTypeOpsWriteLogs,
			},
		},
		{name: "TenMilliSleep",
			loopSleep: 10 * time.Millisecond,
			expectErr: false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeOpsWriteLogs,
				PsrLoopSleep:  "10ms",
			},
		},
		{name: "TenNanoSleep",
			loopSleep: 10 * time.Nanosecond,
			expectErr: false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeOpsWriteLogs,
				PsrLoopSleep:  "10ns",
			},
		},
		// Test min sleep of 10ns
		{name: "TenNanoSleepMin",
			loopSleep: 10 * time.Nanosecond,
			expectErr: false,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeOpsWriteLogs,
				PsrLoopSleep:  "1ns",
			},
		},
		{name: "BadNumericStringFormat",
			expectErr: true,
			envMap: map[string]string{
				PsrWorkerType: WorkerTypeOpsWriteLogs,
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
				PsrWorkerType: WorkerTypeOpsWriteLogs,
			},
		},
		{name: "MultipleWorkerThreads",
			workerThreads: 50,
			expectErr:     false,
			envMap: map[string]string{
				PsrWorkerType:        WorkerTypeOpsWriteLogs,
				PsrWorkerThreadCount: "50",
			},
		},
		// Test max threads 100
		{name: "MaxWorkerThreads",
			workerThreads: 100,
			expectErr:     false,
			envMap: map[string]string{
				PsrWorkerType:        WorkerTypeOpsWriteLogs,
				PsrWorkerThreadCount: "1000",
			},
		},
		{name: "BadThreadCountFormat",
			expectErr: true,
			envMap: map[string]string{
				PsrWorkerType:        WorkerTypeOpsWriteLogs,
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
