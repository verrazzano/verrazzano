// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	asserts "github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var logForTest = zap.S()

func TestCollectReconcileMetrics(t *testing.T) {
	assert := asserts.New(t)
	test := struct {
		name string
	}{
		name: "Test that reoncile counter is incremented by one when function is Successful & reconcile counter is incremented by one when function is Failed",
	}
	t.Run(test.name, func(t *testing.T) {

		reconcileCounterObject, err := GetSimpleCounterMetric(AppconfigReconcileCounter)
		assert.NoError(err)
		reconcileSuccessfulCounterBefore := testutil.ToFloat64(reconcileCounterObject.Get())
		reconcileCounterObject.Inc(logForTest, nil)
		reconcileSuccessfulCounterAfter := testutil.ToFloat64(reconcileCounterObject.Get())
		assert.Equal(reconcileSuccessfulCounterBefore, reconcileSuccessfulCounterAfter-1)

		reconcileerrorCounterObject, err := GetSimpleCounterMetric(AppconfigReconcileError)
		assert.NoError(err)
		reconcileFailedCounterBefore := testutil.ToFloat64(reconcileerrorCounterObject.Get())
		reconcileerrorCounterObject.Inc(logForTest, nil)
		reconcileFailedCounterAfter := testutil.ToFloat64(reconcileerrorCounterObject.Get())
		assert.Equal(reconcileFailedCounterBefore, reconcileFailedCounterAfter-1)

		// Duration Metric test
		reconcileDurationCount, _ := GetDurationMetric(AppconfigReconcileDuration)
		reconcileDurationCount.TimerStart()
		time.Sleep(time.Second)
		reconcileDurationCount.TimerStop()
	})
}
