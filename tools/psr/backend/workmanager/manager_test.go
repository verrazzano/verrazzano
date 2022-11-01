// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workmanager

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"

	"testing"
)

var _ spi.Worker = &fakeWorker{2}

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
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			type fakeEnv struct {
				data map[string]string
			}
			f := fakeEnv{data: test.envMap}

			func (f *fakeEnv) GetEnv(key string) string {
				return f.data[key]
			}


			config.CommonConfig{}

		})
		//	log := vzlog.DefaultLogger()
		//	f := fakeWorker{}
		//	r, err := StartWorkerRunners(config.CommonConfig{}, log)
		//	actualRunner := r.(runner)
		//	assert.NoError(t, err)
		//
		//	err = r.RunWorker(config.CommonConfig{
		//		WorkerType:    "Fake",
		//		NumIterations: test.iterations,
		//	}, log)
		//
		//	assert.NoError(t, err)
		//	assert.Equal(t, f.doWorkCount, test.iterations)
		//	assert.Equal(t, actualRunner.loopCount.Val, test.iterations)
		//})
	}
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w fakeWorker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		EnvName:     config.WorkerTypeExample,
		Description: "Example worker that demonstrates executing a fake use case",
		MetricsName: "example",
	}
}

func (w *fakeWorker) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{}
}

func (w *fakeWorker) GetMetricDescList() []prometheus.Desc {
	return nil
}

func (w *fakeWorker) GetMetricList() []prometheus.Metric {
	return nil
}

func (w *fakeWorker) WantIterationInfoLogged() bool {
	return true
}

func (w *fakeWorker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	w.doWorkCount = w.doWorkCount + 1
	return nil
}
