// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testNameValid   = "Valid Metric Name, no error"
	testNameInvalid = "Invalid Metric name, error"
)

// TestGetSimpleCounterMetric tests GetSimpleCounterMetric function
// GIVEN a metricName
// WHEN a call to GetSimpleCounterMetric is made
// THEN return no error if the metricName is valid, else return an error
func TestGetSimpleCounterMetric(t *testing.T) {
	tests := []struct {
		name       string
		metricName metricName
		wantErr    bool
	}{
		{
			testNameValid,
			"appconfig reconcile counter",
			false,
		},
		{
			testNameValid,
			"istio handle error",
			false,
		},
		{
			testNameInvalid,
			"MultiClusterConfigmap  handle duration",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counterObject, err := GetSimpleCounterMetric(tt.metricName)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, counterObject)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, counterObject)
			}
		})
	}
}

// TestGetDurationMetric tests GetDurationMetric function
// GIVEN a metricName
// WHEN a call to GetDurationMetric is made
// THEN return no error if the metricName is valid, else return an error
func TestGetDurationMetric(t *testing.T) {
	tests := []struct {
		name       string
		metricName metricName
		wantErr    bool
	}{
		{
			testNameValid,
			"MultiClusterConfigmap handle duration",
			false,
		},
		{
			testNameValid,
			"BindingUpdater handle duration",
			false,
		},
		{
			testNameInvalid,
			"VzProj handle counter",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counterObject, err := GetDurationMetric(tt.metricName)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, counterObject)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, counterObject)
			}
		})
	}
}

// TestExposeControllerMetrics tests the ExposeControllerMetrics function
// GIVEN a set of 3 metricNames
// WHEN a call to ExposeControllerMetrics is made
// THEN return respective MetricObject and MetricLogger if the metricNames are valid, else return an error
func TestExposeControllerMetrics(t *testing.T) {
	tests := []struct {
		name                      string
		controllerName            string
		counterMetricName         metricName
		errorCounterMetricName    metricName
		durationCounterMetricName metricName
		wantErr                   bool
	}{
		{
			testNameValid,
			"appconfig",
			AppconfigHandleCounter,
			AppconfigHandleError,
			AppconfigHandleDuration,
			false,
		},
		{
			testNameInvalid,
			"istio",
			IstioHandleDuration,
			IstioHandleCounter,
			IstioHandleError,
			true,
		},
		{
			testNameInvalid,
			"coh",
			CohworkloadReconcileCounter,
			CohworkloadReconcileDuration,
			CohworkloadReconcileDuration,
			true,
		},
		{
			testNameInvalid,
			"helidon",
			HelidonReconcileCounter,
			HelidonReconcileError,
			HelidonReconcileError,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counterMetricObject, errorCounterMetricObject, handleDurationMetricObject, zapLogForMetrics, err := ExposeControllerMetrics(tt.controllerName, tt.counterMetricName, tt.errorCounterMetricName, tt.durationCounterMetricName)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, counterMetricObject)
				assert.Nil(t, errorCounterMetricObject)
				assert.Nil(t, handleDurationMetricObject)
				assert.Nil(t, zapLogForMetrics)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, counterMetricObject)
				assert.NotNil(t, errorCounterMetricObject)
				assert.NotNil(t, handleDurationMetricObject)
				assert.NotNil(t, zapLogForMetrics)
			}
		})
	}
}
