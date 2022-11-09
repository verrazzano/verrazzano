// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package example

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"testing"
)

// TestGetters tests the worker getters
// GIVEN a worker
//
//	WHEN the getter methods are calls
//	THEN ensure that the correct results are returned
func TestGetters(t *testing.T) {
	w, err := NewExampleWorker()
	assert.NoError(t, err)

	wd := w.GetWorkerDesc()
	assert.Equal(t, config.WorkerTypeExample, wd.WorkerType)
	assert.Equal(t, "Example worker that demonstrates executing a fake use case", wd.Description)
	assert.Equal(t, config.WorkerTypeExample, wd.MetricsName)

	el := w.GetEnvDescList()
	assert.Len(t, el, 0)
	mdl := w.GetMetricDescList()
	assert.Len(t, mdl, 0)
	ml := w.GetMetricList()
	assert.Len(t, ml, 0)
	logged := w.WantLoopInfoLogged()
	assert.True(t, logged)
}

// TestDoWork tests the DoWork method
// GIVEN a worker
//
//	WHEN the DoWork methods is called
//	THEN ensure that the correct results are returned
func TestDoWork(t *testing.T) {
	w, err := NewExampleWorker()
	assert.NoError(t, err)

	err = w.DoWork(config.CommonConfig{
		WorkerType: "Fake",
	}, vzlog.DefaultLogger())
	assert.NoError(t, err)

	ew := w.(exampleWorker)
	assert.Equal(t, int64(1), ew.loggedLinesTotal)
}
