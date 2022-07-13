// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	asserts "github.com/stretchr/testify/assert"
)

func TestCollectReconcileMetrics(t *testing.T) {
	assert := asserts.New(t)
	tests := []struct {
		name string
	}{
		{
			name: "Test that reoncile counter is incremented by one when function is Successful & reoncile counter is incremented by one when function is Failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := reconcileMap["appconfig"]

			reconcileSuccessfulCounterBefore := testutil.ToFloat64(r.reconcileSuccessful)
			r.reconcileSuccessful.Inc()
			reconcileSuccessfulCounterAfter := testutil.ToFloat64(r.reconcileSuccessful)
			assert.Equal(reconcileSuccessfulCounterBefore, reconcileSuccessfulCounterAfter-1)

			reconcileFailedCounterBefore := testutil.ToFloat64(r.reconcileFailed)
			r.reconcileFailed.Inc()
			reconcileFailedCounterAfter := testutil.ToFloat64(r.reconcileFailed)
			assert.Equal(reconcileFailedCounterBefore, reconcileFailedCounterAfter-1)

			//Duration Metric test

			// r.GetDurationMetrics().DurationTimerStart()
			// time.Sleep(time.Second)
			// r.GetDurationMetrics().DurationTimerStop()
		})
	}
}
