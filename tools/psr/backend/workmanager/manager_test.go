// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workmanager

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"

	"testing"
)

type fakeManagerWorker struct {
	doWorkCount int64
}

var _ spi.Worker = &fakeManagerWorker{2}

type fakeEnv struct {
	data map[string]string
}

// TestStartWorkerRunners tests the manager
// GIVEN a manager
//
//	WHEN StartWorkerRunners is called
//	THEN ensure that the runners are called
func TestStartWorkerRunners(t *testing.T) {
	var tests = []struct {
		name       string
		envMap     map[string]string
		envKey     string
		workerType string
		expectErr  bool
	}{
		{name: "ExampleWorker",
			workerType: config.WorkerTypeExample,
			expectErr:  false,
			envMap: map[string]string{
				config.PsrWorkerType: config.WorkerTypeExample,
				config.PsrNumLoops:   "1",
			},
		},
		{name: "WorkerTypeWriteLogs",
			workerType: config.WorkerTypeWriteLogs,
			expectErr:  false,
			envMap: map[string]string{
				config.PsrWorkerType: config.WorkerTypeWriteLogs,
				config.PsrNumLoops:   "1",
			},
		},
		{name: "WorkerTypeGetLogs",
			workerType: config.WorkerTypeGetLogs,
			expectErr:  false,
			envMap: map[string]string{
				config.PsrWorkerType: config.WorkerTypeGetLogs,
				config.PsrNumLoops:   "1",
			},
		},
		{name: "MissingWorkerType",
			expectErr: true,
			envMap: map[string]string{
				config.PsrNumLoops: "1",
			},
		},
		{name: "InvalidWorkerType",
			expectErr: true,
			envMap: map[string]string{
				config.PsrWorkerType: "InvalidWorker",
				config.PsrNumLoops:   "1",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := fakeEnv{data: test.envMap}
			saveEnv := osenv.GetEnvFunc
			osenv.GetEnvFunc = f.GetEnv
			defer func() {
				osenv.GetEnvFunc = saveEnv
			}()

			saveStartMetrics := startMetricsFunc
			startMetricsFunc = fakeStartMetrics
			defer func() {
				startMetricsFunc = saveStartMetrics
			}()

			err := StartWorkerRunners(vzlog.DefaultLogger())
			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w fakeManagerWorker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		WorkerType:  config.WorkerTypeExample,
		Description: "Example worker that demonstrates executing a fake use case",
		MetricsName: "example",
	}
}

func (w *fakeManagerWorker) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{}
}

func (w *fakeManagerWorker) GetMetricDescList() []prometheus.Desc {
	return nil
}

func (w *fakeManagerWorker) GetMetricList() []prometheus.Metric {
	return nil
}

func (w *fakeManagerWorker) WantLoopInfoLogged() bool {
	return true
}

func (w *fakeManagerWorker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	w.doWorkCount = w.doWorkCount + 1
	return nil
}

func (f *fakeEnv) GetEnv(key string) string {
	return f.data[key]
}

func fakeStartMetrics(_ []spi.WorkerMetricsProvider) {
}
