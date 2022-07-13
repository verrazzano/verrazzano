// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	asserts "github.com/stretchr/testify/assert"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

//TestCollectReconcileMetrics tests the CollectReconcileMetrics fn
//GIVEN a call to CollectReconcileMetrics
//WHEN A starting time is passed into the function
//THEN the function updates the reconcileCounterMetric by 1 and creates a new time for that reconcile in the reconcileLastDurationMetric
func TestCollectReconcileMetrics(t *testing.T) {
	assert := asserts.New(t)
	tests := []struct {
		name                   string
		expectedIncrementValue float64
	}{
		{
			name:                   "Test that reoncile counter is incremented by one when function is called",
			expectedIncrementValue: float64(1),
		},
		{
			name:                   "Test that reconcile Duration is being updated correctly",
			expectedIncrementValue: float64(1),
		},
		{
			name:                   "Test that reconcile Error counter metric is incremented when a value of ErrorOccured is passed into the function",
			expectedIncrementValue: float64(1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime := time.Now().UnixMilli()
			time.Sleep(1 * time.Millisecond)
			reconcileCounterBefore := testutil.ToFloat64(reconcileCounterMetric)
			CollectReconcileMetricsTime(startTime)
			reconcileCounterAfter := testutil.ToFloat64(reconcileCounterMetric)
			assert.Equal(tt.expectedIncrementValue, reconcileCounterAfter-reconcileCounterBefore)
			//Reconcile Index is decremented by one because when the function is called Reconcile index is incremented by one at the end of the fn
			//However, the gauge inside the gauge vector that we want to test is accessed with the original value of reconcile index that was used in the function call
			//Before passing
			metric, _ := reconcileLastDurationMetric.GetMetricWithLabelValues(strconv.Itoa(reconcileIndex - 1))
			assert.Greater(testutil.ToFloat64(metric), float64(0))
		})
	}
}
func TestErrorCounterMetric(t *testing.T) {
	assert := asserts.New(t)
	tests := []struct {
		name                        string
		expectedErrorIncrementValue float64
	}{
		{
			name:                        "Test that reoncile counter is incremented by one when function is called",
			expectedErrorIncrementValue: float64(1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorCounterBefore := testutil.ToFloat64(reconcileErrorCounterMetric)
			CollectReconcileMetricsError()
			errorCounterAfter := testutil.ToFloat64(reconcileErrorCounterMetric)
			assert.Equal(tt.expectedErrorIncrementValue, errorCounterAfter-errorCounterBefore)
		})
	}
}

// TestAnalyzeVZCR tests the AnalyzeVZCR fn
// GIVEN a call to AnalyzeVZCR
// WHEN a VZ CR with or without timestamps is passed to the fn
// THEN the function properly updates or does nothing to the component's metric
func TestAnalyzeVZCR(t *testing.T) {
	assert := asserts.New(t)
	emptyVZCR := installv1alpha1.Verrazzano{}
	disabledComponentVZCR := installv1alpha1.Verrazzano{
		Status: installv1alpha1.VerrazzanoStatus{
			Components: installv1alpha1.ComponentStatusMap{
				"grafana": &installv1alpha1.ComponentStatusDetails{
					State: installv1alpha1.CompStateDisabled,
				},
			},
		},
	}
	conditionsNotFullyMetVZCR := installv1alpha1.Verrazzano{
		Status: installv1alpha1.VerrazzanoStatus{
			Components: installv1alpha1.ComponentStatusMap{
				"grafana": &installv1alpha1.ComponentStatusDetails{
					Conditions: []installv1alpha1.Condition{
						{
							Type:               installv1alpha1.CondInstallStarted,
							LastTransitionTime: "2022-07-06T13:54:59Z",
						},
					},
				},
			},
		},
	}
	installPopulatedVZCR := installv1alpha1.Verrazzano{
		Status: installv1alpha1.VerrazzanoStatus{
			Components: installv1alpha1.ComponentStatusMap{
				"grafana": &installv1alpha1.ComponentStatusDetails{
					Conditions: []installv1alpha1.Condition{
						{
							Type:               installv1alpha1.CondInstallStarted,
							LastTransitionTime: "2022-07-06T13:54:59Z",
						},
						{
							Type:               installv1alpha1.CondInstallComplete,
							LastTransitionTime: "2022-07-06T13:55:45Z",
						},
						{
							Type:               installv1alpha1.CondUpgradeStarted,
							LastTransitionTime: "2022-07-06T13:58:59Z",
						},
					},
				},
			},
		},
	}
	upgradeStartTimeisAfterUpgradeCompletedTimeVZCR := installv1alpha1.Verrazzano{
		Status: installv1alpha1.VerrazzanoStatus{
			Components: installv1alpha1.ComponentStatusMap{
				"unregistered test component": &installv1alpha1.ComponentStatusDetails{
					Conditions: []installv1alpha1.Condition{
						{
							Type:               installv1alpha1.CondUpgradeStarted,
							LastTransitionTime: "2022-07-06T13:55:45Z",
						},
						{
							Type:               installv1alpha1.CondUpgradeComplete,
							LastTransitionTime: "2022-07-06T13:54:59Z",
						},
					},
				},
			},
		},
	}
	upgradeAndInstallPopulatedVZCR := installv1alpha1.Verrazzano{
		Status: installv1alpha1.VerrazzanoStatus{
			Components: installv1alpha1.ComponentStatusMap{
				"grafana": &installv1alpha1.ComponentStatusDetails{
					Conditions: []installv1alpha1.Condition{
						{
							Type:               installv1alpha1.CondInstallStarted,
							LastTransitionTime: "2022-07-06T13:54:59Z",
						},
						{
							Type:               installv1alpha1.CondInstallComplete,
							LastTransitionTime: "2022-07-06T13:55:45Z",
						},
						{
							Type:               installv1alpha1.CondUpgradeStarted,
							LastTransitionTime: "2022-07-06T13:58:59Z",
						},
						{
							Type:               installv1alpha1.CondUpgradeComplete,
							LastTransitionTime: "2022-07-06T13:59:00Z",
						},
					},
				},
			},
		},
	}
	componentNameNotInDictionaryVZCR := installv1alpha1.Verrazzano{
		Status: installv1alpha1.VerrazzanoStatus{
			Components: installv1alpha1.ComponentStatusMap{
				"unregistered test component": &installv1alpha1.ComponentStatusDetails{
					Conditions: []installv1alpha1.Condition{
						{
							Type:               installv1alpha1.CondInstallStarted,
							LastTransitionTime: "2022-07-06T13:54:59Z",
						},
						{
							Type:               installv1alpha1.CondInstallComplete,
							LastTransitionTime: "2022-07-06T13:55:45Z",
						},
						{
							Type:               installv1alpha1.CondUpgradeStarted,
							LastTransitionTime: "2022-07-06T13:58:59Z",
						},
						{
							Type:               installv1alpha1.CondUpgradeComplete,
							LastTransitionTime: "2022-07-06T13:59:00Z",
						},
					},
				},
			},
		},
	}
	installStartTimeisAfterInstallCompletedTimeVZCR := installv1alpha1.Verrazzano{
		Status: installv1alpha1.VerrazzanoStatus{
			Components: installv1alpha1.ComponentStatusMap{
				"unregistered test component": &installv1alpha1.ComponentStatusDetails{
					Conditions: []installv1alpha1.Condition{
						{
							Type:               installv1alpha1.CondInstallStarted,
							LastTransitionTime: "2022-07-06T13:55:45Z",
						},
						{
							Type:               installv1alpha1.CondInstallComplete,
							LastTransitionTime: "2022-07-06T13:54:59Z",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name                          string
		vzcr                          installv1alpha1.Verrazzano
		expectedValueForInstallMetric float64
		expectedValueForUpdateMetric  float64
	}{
		{
			name:                          "test empty Verrazzano",
			vzcr:                          emptyVZCR,
			expectedValueForInstallMetric: float64(0),
			expectedValueForUpdateMetric:  float64(0),
		},
		{
			name:                          "test that a diabled component does not have an install or upgrade time",
			vzcr:                          disabledComponentVZCR,
			expectedValueForInstallMetric: float64(0),
			expectedValueForUpdateMetric:  float64(0),
		},
		{
			name:                          "Verrazzano where install has started, but not yet completed",
			vzcr:                          conditionsNotFullyMetVZCR,
			expectedValueForInstallMetric: float64(0),
			expectedValueForUpdateMetric:  float64(0),
		},
		{
			name:                          "test populated Verrazzano where install has started and completed, but upgrade has not yet completed",
			vzcr:                          installPopulatedVZCR,
			expectedValueForInstallMetric: float64(46),
			expectedValueForUpdateMetric:  float64(0),
		},
		{
			name:                          "test that a VZ CR with an upgrade start time after its upgrade completion time does not update the update duration metric for that component",
			vzcr:                          upgradeStartTimeisAfterUpgradeCompletedTimeVZCR,
			expectedValueForInstallMetric: float64(46),
			expectedValueForUpdateMetric:  float64(0),
		},
		{
			name:                          "test populated Verrazzano where both install and upgrade have started and completed",
			vzcr:                          upgradeAndInstallPopulatedVZCR,
			expectedValueForInstallMetric: float64(46),
			expectedValueForUpdateMetric:  float64(1),
		},
		{
			name:                          "test that an unrecognized component does not cause a seg fault, the analyze VZCR function keeps going on",
			vzcr:                          componentNameNotInDictionaryVZCR,
			expectedValueForInstallMetric: float64(46),
			expectedValueForUpdateMetric:  float64(1),
		},
		{
			name:                          "test that a VZ CR with an install start time after its install completion time does not update the install duration metric for that component",
			vzcr:                          installStartTimeisAfterInstallCompletedTimeVZCR,
			expectedValueForInstallMetric: float64(46),
			expectedValueForUpdateMetric:  float64(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AnalyzeVZCR(tt.vzcr)
			assert.Equal(tt.expectedValueForInstallMetric, testutil.ToFloat64(grafanaInstallTimeMetric))
			assert.Equal(tt.expectedValueForUpdateMetric, testutil.ToFloat64(grafanaUpgradeTimeMetric))
		})
	}
}
