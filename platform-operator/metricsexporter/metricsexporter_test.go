// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

// Constants that hold the times that are used to test various cases of component timestamps being passed
// into the TestAnalyzeVerrazzanoResourceMetrics function
const (
	componentFirstTime        string = "2022-07-06T13:54:59Z"
	componentSecondTime       string = "2022-07-06T13:55:45Z"
	componentThirdTime        string = "2022-07-06T13:58:59Z"
	componentFourthTime       string = "2022-07-06T13:59:00Z"
	unregisteredTestComponent string = "unregistered test component"
)

// TestReconcileCounterIncrement tests the Inc fn of the reconcile counter and the reconcile error counter metrics objects
// GIVEN a call to Inc
// THEN the function should update that internal metric by one
func TestReconcileCounterAndErrorIncrement(t *testing.T) {
	assert := asserts.New(t)
	test := struct {
		name                             string
		expectedIncrementValueForCounter float64
		expectedIncrementValueForError   float64
	}{
		name:                             "Test that reoncile counter is incremented by one when function is called",
		expectedIncrementValueForCounter: float64(1),
		expectedIncrementValueForError:   float64(1),
	}
	t.Run(test.name, func(t *testing.T) {
		reconcileCounterObject, err := GetSimpleCounterMetric(ReconcileCounter)
		assert.NoError(err)
		reconcileCounterBefore := testutil.ToFloat64(reconcileCounterObject.Get())
		reconcileCounterObject.Inc()
		reconcileCounterAfter := testutil.ToFloat64(reconcileCounterObject.Get())
		assert.Equal(test.expectedIncrementValueForCounter, reconcileCounterAfter-reconcileCounterBefore)
		reconcileErrorCounterObject, err := GetSimpleCounterMetric(ReconcileError)
		assert.NoError(err)
		reconcileErrorCounterBefore := testutil.ToFloat64(reconcileErrorCounterObject.Get())
		reconcileErrorCounterObject.Inc()
		reconcileErrorCounterAfter := testutil.ToFloat64(reconcileErrorCounterObject.Get())
		assert.Equal(test.expectedIncrementValueForError, reconcileErrorCounterAfter-reconcileErrorCounterBefore)
	})
}

// TestAnalyzeVerrazzanoResourceMetrics tests the AnalyzeVerrazzanoResourceMetrics fn
// GIVEN a call to AnalyzeVerrazzanoResourceMetrics
// WHEN a VZ CR with or without timestamps is passed to the fn
// THEN the function properly updates or does nothing to the component's metric
func TestAnalyzeVerrazzanoResourceMetrics(t *testing.T) {
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
				unregisteredTestComponent: &installv1alpha1.ComponentStatusDetails{
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
				unregisteredTestComponent: &installv1alpha1.ComponentStatusDetails{
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
				unregisteredTestComponent: &installv1alpha1.ComponentStatusDetails{
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
			grafanaMetricComponentObject, err := GetMetricComponent(grafanaMetricName)
			assert.NoError(err)
			grafanaInstallMetric := grafanaMetricComponentObject.getInstallDuration()
			assert.Equal(tt.expectedValueForInstallMetric, testutil.ToFloat64(grafanaInstallMetric.Get()))
			grafanaUpgradeMetric := grafanaMetricComponentObject.getUpgradeDuration()
			assert.Equal(tt.expectedValueForUpdateMetric, testutil.ToFloat64(grafanaUpgradeMetric.Get()))
		})
	}
}
