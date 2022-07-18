// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
)

// Constants that hold the times that are used to test various cases of component timestamps being passed
// into the TestAnalyzeVerrazzanoResourceMetrics function
const (
	componentFirstTime  string = "2022-07-06T13:54:59Z"
	componentSecondTime string = "2022-07-06T13:55:45Z"
	componentThirdTime  string = "2022-07-06T13:58:59Z"
	componentFourthTime string = "2022-07-06T13:59:00Z"
)

// TestCollectReconcileMetricsTime tests the CollectReconcileMetricsTime fn
// GIVEN a call to CollectReconcileMetricsTime
// WHEN A starting time is passed into the function
// THEN the function updates the reconcileCounterMetric by 1 and creates a new time for that reconcile in the reconcileLastDurationMetric
func TestCollectReconcileMetricsTime(t *testing.T) {
	InitalizeMetricsWrapper()
	assert := asserts.New(t)
	test := struct {
		name                   string
		expectedIncrementValue float64
	}{
		name:                   "Test that reoncile counter is incremented by one when function is called",
		expectedIncrementValue: float64(1),
	}
	t.Run(test.name, func(t *testing.T) {
		startTime := time.Now()
		time.Sleep(1 * time.Millisecond)
		reconcileCounterBefore := testutil.ToFloat64(metricsExp.ReconcileCounterMetric)
		CollectReconcileMetricsTime(startTime, zap.S())
		reconcileCounterAfter := testutil.ToFloat64(metricsExp.ReconcileCounterMetric)
		assert.Equal(test.expectedIncrementValue, reconcileCounterAfter-reconcileCounterBefore)
		// Reconcile Index is decremented by one because when the function is called Reconcile index is incremented by one at the end of the fn
		// However, the gauge inside the gauge vector that we want to test is accessed with the original value of reconcile index that was used in the function call
		metric, _ := metricsExp.ReconcileLastDurationMetric.GetMetricWithLabelValues(strconv.Itoa(metricsExp.ReconcileIndex - 1))
		assert.Greater(testutil.ToFloat64(metric), float64(0))
	})
}

// TestCollectReconcileError tests the CollectReconcileError fn
// GIVEN a call to CollectReconcileError
// WHEN the function is called
// THEN the function increments the reconcile error counter metric
func TestCollectReconcileError(t *testing.T) {
	InitalizeMetricsWrapper()
	assert := asserts.New(t)
	test := struct {
		name                        string
		expectedErrorIncrementValue float64
	}{

		name:                        "Test that reoncile counter is incremented by one when function is called",
		expectedErrorIncrementValue: float64(1),
	}
	t.Run(test.name, func(t *testing.T) {
		errorCounterBefore := testutil.ToFloat64(metricsExp.ReconcileErrorCounterMetric)
		CollectReconcileMetricsError(zap.S())
		errorCounterAfter := testutil.ToFloat64(metricsExp.ReconcileErrorCounterMetric)
		assert.Equal(test.expectedErrorIncrementValue, errorCounterAfter-errorCounterBefore)
	})
}

// TestAnalyzeVerrazzanoResourceMetrics tests the AnalyzeVerrazzanoResourceMetrics fn
// GIVEN a call to AnalyzeVerrazzanoResourceMetrics
// WHEN a VZ CR with or without timestamps is passed to the fn
// THEN the function properly updates or does nothing to the component's metric
func TestAnalyzeVerrazzanoResourceMetrics(t *testing.T) {
	InitalizeMetricsWrapper()
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
							LastTransitionTime: componentFirstTime,
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
							LastTransitionTime: componentFirstTime,
						},
						{
							Type:               installv1alpha1.CondInstallComplete,
							LastTransitionTime: componentSecondTime,
						},
						{
							Type:               installv1alpha1.CondUpgradeStarted,
							LastTransitionTime: componentThirdTime,
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
							LastTransitionTime: componentSecondTime,
						},
						{
							Type:               installv1alpha1.CondUpgradeComplete,
							LastTransitionTime: componentFirstTime,
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
							LastTransitionTime: componentFirstTime,
						},
						{
							Type:               installv1alpha1.CondInstallComplete,
							LastTransitionTime: componentSecondTime,
						},
						{
							Type:               installv1alpha1.CondUpgradeStarted,
							LastTransitionTime: componentThirdTime,
						},
						{
							Type:               installv1alpha1.CondUpgradeComplete,
							LastTransitionTime: componentFourthTime,
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
							LastTransitionTime: componentFirstTime,
						},
						{
							Type:               installv1alpha1.CondInstallComplete,
							LastTransitionTime: componentSecondTime,
						},
						{
							Type:               installv1alpha1.CondUpgradeStarted,
							LastTransitionTime: componentThirdTime,
						},
						{
							Type:               installv1alpha1.CondUpgradeComplete,
							LastTransitionTime: componentFourthTime,
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
							LastTransitionTime: componentSecondTime,
						},
						{
							Type:               installv1alpha1.CondInstallComplete,
							LastTransitionTime: componentFirstTime,
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
	testLog, _ := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           "Test",
		Namespace:      "Test namespace",
		ID:             "Test ID",
		Generation:     int64(1),
		ControllerName: "test controller",
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AnalyzeVerrazzanoResourceMetrics(testLog, tt.vzcr)
			assert.Equal(tt.expectedValueForInstallMetric, getValueOfInstallMetricForTesting("grafana"))
			assert.Equal(tt.expectedValueForUpdateMetric, getValueOfUpgradeMetricForTesting("grafana"))
		})
	}
}
func getValueOfInstallMetricForTesting(componentName string) float64 {
	return testutil.ToFloat64(metricsExp.MetricsMap[componentName].LatestInstallDuration)
}
func getValueOfUpgradeMetricForTesting(componentName string) float64 {
	return testutil.ToFloat64(metricsExp.MetricsMap[componentName].LatestUpgradeDuration)
}
