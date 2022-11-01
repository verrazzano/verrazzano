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

type fakeWorker struct {
	doWorkCount int64
}

var _ spi.Worker = &fakeWorker{}

// TestMetricDesc tests the Runner Metric Descriptors
// GIVEN a Runner
//
//	WHEN GetMetricDescList is called
//	THEN ensure that all the descriptors are correct
func TestMetricDesc(t *testing.T) {
	log := vzlog.DefaultLogger()
	r, err := NewRunner(&fakeWorker{}, config.CommonConfig{}, log)
	assert.NoError(t, err)

	// Make sure each Desc is expected
	mdList := r.GetMetricDescList()
	assertMetricDescList(t, mdList)

	// Make sure each Metric Desc is expected
	mList := r.GetMetricList()
	mdList2 := []prometheus.Desc{}
	for i := range mList {
		mdList2 = append(mdList2, *mList[i].Desc())
	}
	assertMetricDescList(t, mdList2)
}

func assertMetricDescList(t *testing.T, mdList []prometheus.Desc) {
	const (
		desc1 = `Desc{fqName: "psr_example_loop_count_total", help: "The total number of loop iterations executed", constLabels: {}, variableLabels: []}`
		desc2 = `Desc{fqName: "psr_example_worker_thread_count_total", help: "The total number of worker threads (goroutines) running", constLabels: {}, variableLabels: []}`
		desc3 = `Desc{fqName: "psr_example_worker_last_iteration_nanoseconds", help: "The number of nanoseconds that the worker took to run the last iteration of doing work", constLabels: {}, variableLabels: []}`
		desc4 = `Desc{fqName: "psr_example_worker_running_seconds_total", help: "The total number of seconds that the worker has been running", constLabels: {}, variableLabels: []}`
	)

	// Build set to validate descriptors
	mdSet := map[string]bool{
		desc1: true,
		desc2: true,
		desc3: true,
		desc4: true,
	}

	// Make sure each Desc is expected
	assert.Len(t, mdList, 4)

	for _, md := range mdList {
		_, ok := mdSet[md.String()]
		assert.NotNil(t, ok)
		delete(mdSet, md.String())
	}
	// Make sure each string was seen
	assert.Len(t, mdSet, 0)
}

// TestRunWorker tests the Runner.StartWorkerRunners method
// GIVEN a Runner
//
//	WHEN StartWorkerRunners is called for the correct number of iterations
//	THEN ensure that the worker is called
func TestRunWorker(t *testing.T) {
	var tests = []struct {
		name       string
		iterations int64
		expectErr  bool
	}{
		{name: "oneIter", iterations: 1, expectErr: false},
		{name: "tenIter", iterations: 10, expectErr: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			log := vzlog.DefaultLogger()
			f := fakeWorker{}
			r, err := NewRunner(&f, config.CommonConfig{}, log)
			actualRunner := r.(runner)
			assert.NoError(t, err)

			err = r.RunWorker(config.CommonConfig{
				WorkerType:    "Fake",
				NumIterations: test.iterations,
			}, log)

			assert.NoError(t, err)
			assert.Equal(t, f.doWorkCount, test.iterations)
			assert.Equal(t, actualRunner.loopCount.Val, test.iterations)
		})
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
