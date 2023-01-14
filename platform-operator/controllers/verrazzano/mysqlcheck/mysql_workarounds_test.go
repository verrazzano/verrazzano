// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqlcheck

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysqloperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	testScheme                = runtime.NewScheme()
	innoDBClusterStatusFields = []string{"status", "cluster", "status"}
	checkPeriodDuration       = time.Duration(1) * time.Second
	timeoutDuration           = time.Duration(120) * time.Second
)

const (
	innoDBClusterStatusOnline = "ONLINE"
)

func init() {
	_ = k8scheme.AddToScheme(testScheme)
}

// TestRepairMySQLPodsWaitingReadinessGates tests the temporary workaround for MySQL
// pods getting stuck during install waiting for all readiness gates to be true.
// GIVEN a MySQL Pod with readiness gates defined
// WHEN they are not all ready after a given time period
// THEN recycle the mysql-operator
func TestRepairMySQLPodsWaitingReadinessGates(t *testing.T) {
	mySQLOperatorPod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysqloperator.ComponentName,
			Namespace: mysqloperator.ComponentNamespace,
			Labels: map[string]string{
				"name": mysqloperator.ComponentName,
			},
		},
	}

	mySQLPod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysql-0",
			Namespace: componentNamespace,
			Labels: map[string]string{
				mySQLComponentLabel: mySQLDComponentName,
			},
		},
		Spec: v1.PodSpec{
			ReadinessGates: []v1.PodReadinessGate{{ConditionType: "gate1"}, {ConditionType: "gate2"}},
		},
		Status: v1.PodStatus{
			Conditions: []v1.PodCondition{
				{Type: "gate1", Status: v1.ConditionTrue},
				{Type: "gate2", Status: v1.ConditionTrue},
			},
		},
	}

	// Set up the initial context
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLPod, mySQLOperatorPod).Build()
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err := NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)
	assert.True(t, getLastTimeReadinessGateChecked().IsZero())

	// Timer should remain zero when all conditions are true. Expect no error and mysql-operator pod to still exist.
	// No readiness gate event should be created
	err = mysqlCheck.RepairMySQLPodsWaitingReadinessGates()
	assert.NoError(t, err)
	assert.True(t, getLastTimeReadinessGateChecked().IsZero())
	assert.False(t, isReadinessGateEvent(fakeCtx))
	assert.False(t, isMySQLOperatorEvent(fakeCtx))

	pod := &v1.Pod{}
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, pod)
	assert.NoError(t, err)

	// Set one of the conditions to false
	mySQLPod.Status.Conditions = []v1.PodCondition{{Type: "gate1", Status: v1.ConditionTrue}, {Type: "gate2", Status: v1.ConditionFalse}}
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLPod, mySQLOperatorPod).Build()
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err = NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)

	// Expect timer to get started when ond of the conditions is not met.  The mysql-operator pod should still exist.
	// No readiness gate event should be created
	err = mysqlCheck.RepairMySQLPodsWaitingReadinessGates()
	assert.NoError(t, err)
	assert.False(t, getLastTimeReadinessGateChecked().IsZero())
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, pod)
	assert.NoError(t, err)
	assert.False(t, isReadinessGateEvent(fakeCtx))
	assert.False(t, isMySQLOperatorEvent(fakeCtx))

	// Set the last time readiness gate checked to exceed the RepairTimeout period.  Expect the mysql-operator to get recycled.
	// Expect readiness gate event and mysql-operator event
	setInitialTimeReadinessGateChecked(time.Now().Add(-time.Hour * 2))
	err = mysqlCheck.RepairMySQLPodsWaitingReadinessGates()
	assert.NoError(t, err, fmt.Sprintf("unexpected error: %v", err))
	assert.True(t, getLastTimeReadinessGateChecked().IsZero())
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, pod)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
	assert.True(t, isReadinessGateEvent(fakeCtx))
	assert.True(t, isMySQLOperatorEvent(fakeCtx))
}

// TestRepairICStuckDeleting tests the temporary workaround for MySQL
// IC object getting stuck being deleted.
// GIVEN a IC with deletion timestamp
// WHEN not deleted after the expected timer expires
// THEN recycle the mysql-operator
func TestRepairICStuckDeleting(t *testing.T) {
	mySQLOperatorPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysqloperator.ComponentName,
			Namespace: mysqloperator.ComponentNamespace,
			Labels: map[string]string{
				"name": mysqloperator.ComponentName,
			},
		},
	}

	// Test without a deletion timestamp, the timer should not get initialized
	// No events should be created
	innoDBCluster := newInnoDBCluster(innoDBClusterStatusOnline)
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLOperatorPod, innoDBCluster).Build()
	resetInitialTimeICUninstallChecked()
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err := NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)
	assert.NotNil(t, mysqlCheck)

	err = RepairICStuckDeleting(fakeCtx)
	assert.NoError(t, err)
	assert.True(t, getInitialTimeICUninstallChecked().IsZero())
	assert.False(t, isInnobDBClusterEvent(fakeCtx))
	assert.False(t, isMySQLOperatorEvent(fakeCtx))

	// Test first time calling with a deletion timestamp, the timer should get initialized
	// and the mysql-operator pod should not get deleted.
	// No events should be created.
	innoDBCluster = newInnoDBCluster(innoDBClusterStatusOnline)
	startTime := metav1.Now()
	innoDBCluster.SetDeletionTimestamp(&startTime)
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLOperatorPod, innoDBCluster).Build()
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)

	assert.True(t, getInitialTimeICUninstallChecked().IsZero())
	err = RepairICStuckDeleting(fakeCtx)
	assert.NoError(t, err)
	assert.False(t, getInitialTimeICUninstallChecked().IsZero())
	assert.False(t, isInnobDBClusterEvent(fakeCtx))
	assert.False(t, isMySQLOperatorEvent(fakeCtx))

	pod := &v1.Pod{}
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, pod)
	assert.NoError(t, err, "expected the mysql-operator pod to be found")

	// Call repair after the timer has started, but not expired.
	err = RepairICStuckDeleting(fakeCtx)
	assert.NoError(t, err)
	assert.False(t, isInnobDBClusterEvent(fakeCtx))
	assert.False(t, isMySQLOperatorEvent(fakeCtx))

	// Force the timer to be expired, expect the mysql-operator pod to be deleted.
	// Expect event to be created for IC stuck deleting
	// Expect event to be created for restarting mysql-operator
	setInitialTimeICUninstallChecked(time.Now().Add(-time.Hour * 2))
	err = RepairICStuckDeleting(fakeCtx)
	assert.NoError(t, err)
	assert.True(t, isInnobDBClusterEvent(fakeCtx))
	assert.True(t, isMySQLOperatorEvent(fakeCtx))

	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, pod)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

	// If the IC object is already deleted, then no error should be returned
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLOperatorPod).Build()
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)
	err = RepairICStuckDeleting(fakeCtx)
	assert.NoError(t, err)
	assert.False(t, isInnobDBClusterEvent(fakeCtx))
	assert.False(t, isMySQLOperatorEvent(fakeCtx))
}

// TestRepairMySQLPodsStuckTerminating tests the temporary workaround for MySQL
// pods getting stuck terminating.
// GIVEN a pod is deleting
// WHEN still deleting after the expiration time period
// THEN recycle the mysql-operator
func TestRepairMySQLPodsStuckTerminating(t *testing.T) {
	mySQLOperatorPod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysqloperator.ComponentName,
			Namespace: mysqloperator.ComponentNamespace,
			Labels: map[string]string{
				"name": mysqloperator.ComponentName,
			},
		},
	}

	mySQLPod0 := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysql-0",
			Namespace: componentNamespace,
			Labels: map[string]string{
				mySQLComponentLabel: mySQLDComponentName,
			},
		},
	}

	mySQLPod1 := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysql-1",
			Namespace: componentNamespace,
			Labels: map[string]string{
				mySQLComponentLabel: mySQLDComponentName,
			},
		},
	}

	mySQLPod2DeleteTime := metav1.Now()
	mySQLPod2 := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysql-2",
			Namespace: componentNamespace,
			Labels: map[string]string{
				mySQLComponentLabel: mySQLDComponentName,
			},
			DeletionTimestamp: &mySQLPod2DeleteTime,
		},
	}

	// Call with no MySQL pods being deleted, expect success
	// No events should be generated
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLPod0, mySQLPod1).Build()
	resetInitialTimeMySQLPodsStuckChecked()
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err := NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)

	err = mysqlCheck.RepairMySQLPodStuckDeleting()
	assert.NoError(t, err)
	assert.True(t, getInitialTimeMySQLPodsStuckChecked().IsZero())
	assert.False(t, isPodStuckTerminatingEvent(fakeCtx))
	assert.False(t, isMySQLOperatorEvent(fakeCtx))

	// Call with MySQL pods being deleted, first time expect timer to start
	// No events should be generated
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLPod0, mySQLPod1, mySQLPod2).Build()
	resetInitialTimeMySQLPodsStuckChecked()
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err = NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)

	err = mysqlCheck.RepairMySQLPodStuckDeleting()
	assert.NoError(t, err)
	assert.False(t, getInitialTimeMySQLPodsStuckChecked().IsZero())
	assert.False(t, isPodStuckTerminatingEvent(fakeCtx))
	assert.False(t, isMySQLOperatorEvent(fakeCtx))

	// Call with MySQL pods being deleted and timer expired, expect mysql-operator pod to be deleted
	// Events should be generated
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLOperatorPod, mySQLPod0, mySQLPod1, mySQLPod2).Build()
	setInitialTimeMySQLPodsStuckChecked(time.Now().Add(-time.Hour * 2))
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err = NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)

	err = mysqlCheck.RepairMySQLPodStuckDeleting()
	assert.NoError(t, err)
	assert.True(t, getInitialTimeMySQLPodsStuckChecked().IsZero())
	assert.True(t, isPodStuckTerminatingEvent(fakeCtx))
	assert.True(t, isMySQLOperatorEvent(fakeCtx))

	pod := &v1.Pod{}
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, pod)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

	// Call with no MySQL pods, expect success
	// No events should be generated
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects().Build()
	resetInitialTimeMySQLPodsStuckChecked()
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err = NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)

	err = mysqlCheck.RepairMySQLPodStuckDeleting()
	assert.NoError(t, err)
	assert.True(t, getInitialTimeMySQLPodsStuckChecked().IsZero())
	assert.False(t, isPodStuckTerminatingEvent(fakeCtx))
	assert.False(t, isMySQLOperatorEvent(fakeCtx))
}

func newInnoDBCluster(status string) *unstructured.Unstructured {
	innoDBCluster := unstructured.Unstructured{}
	innoDBCluster.SetGroupVersionKind(innoDBClusterGVK)
	innoDBCluster.SetNamespace(componentNamespace)
	innoDBCluster.SetName(helmReleaseName)
	_ = unstructured.SetNestedField(innoDBCluster.Object, status, innoDBClusterStatusFields...)
	return &innoDBCluster
}

// RepairMySQLRouterPodsCrashLoopBackoff tests the temporary workaround for mysql-router
// pods getting stuck in CrashLoopBackoff state.
// GIVEN a mysql-router pod
// WHEN it is in state CrashLoopBackoff
// THEN delete the pod
func TestRepairMySQLRouterPodsCrashLoopBackoff(t *testing.T) {
	routerName := "mysql-router-0"
	mySQLRouterPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routerName,
			Namespace: componentNamespace,
			Labels: map[string]string{
				mySQLComponentLabel: mysqlRouterComponentName,
			},
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
		},
	}

	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLRouterPod).Build()
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err := NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)
	err = mysqlCheck.RepairMySQLRouterPodsCrashLoopBackoff()
	assert.NoError(t, err)

	pod := v1.Pod{}
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: componentNamespace, Name: routerName}, &pod)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

}

// TestCreateEvent tests the creation of events
// GIVEN an event to record
// WHEN it is the first time for the event
// THEN event gets created
// WHEN it is second time for the event
// THEN event gets updated
func TestCreateEvent(t *testing.T) {
	router1Name := "mysql-router-1"
	router2Name := "mysql-router-2"
	mySQLRouterPod1 := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            router1Name,
			Namespace:       componentNamespace,
			UID:             "uid1",
			ResourceVersion: "resource-version1",
		},
	}
	mySQLRouterPod2 := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            router2Name,
			Namespace:       componentNamespace,
			UID:             "uid2",
			ResourceVersion: "resource-version2",
		},
	}

	alertName := "test"
	reason := "PodStuck"
	message := "Pod was stuck terminating"
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLRouterPod1).Build()
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err := NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)

	mysqlCheck.logEvent(mySQLRouterPod1, alertName, reason, message)
	event := &v1.Event{}
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: componentNamespace, Name: generateAlertName(alertName)}, event)
	assert.NoError(t, err)
	assert.Equal(t, reason, event.Reason)
	assert.Equal(t, message, event.Message)
	commonEventAsserts(t, mySQLRouterPod1, event)

	// Log the same event again with a different involved object
	saveFirstTime := event.FirstTimestamp
	saveLastTime := event.LastTimestamp
	time.Sleep(2 * time.Second)
	mysqlCheck.logEvent(mySQLRouterPod2, alertName, reason, message)
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: componentNamespace, Name: generateAlertName(alertName)}, event)
	assert.NoError(t, err)
	assert.Equal(t, saveFirstTime, event.FirstTimestamp)
	assert.NotEqual(t, saveLastTime, event.LastTimestamp)
	commonEventAsserts(t, mySQLRouterPod2, event)
}

// commonEventAsserts - common asserts for event and involved object
func commonEventAsserts(t *testing.T, pod *v1.Pod, event *v1.Event) {
	assert.Equal(t, pod.Kind, event.InvolvedObject.Kind)
	assert.Equal(t, pod.APIVersion, event.InvolvedObject.APIVersion)
	assert.Equal(t, pod.Namespace, event.InvolvedObject.Namespace)
	assert.Equal(t, pod.Name, event.InvolvedObject.Name)
	assert.Equal(t, pod.UID, event.InvolvedObject.UID)
	assert.Equal(t, pod.ResourceVersion, event.InvolvedObject.ResourceVersion)
}

func isInnobDBClusterEvent(ctx spi.ComponentContext) bool {
	return isEvent(ctx, alertInnoDBCluster, componentNamespace)
}

func isMySQLOperatorEvent(ctx spi.ComponentContext) bool {
	return isEvent(ctx, alertMySQLOperator, mysqloperator.ComponentNamespace)
}

func isReadinessGateEvent(ctx spi.ComponentContext) bool {
	return isEvent(ctx, alertReadinessGate, componentNamespace)
}

func isPodStuckTerminatingEvent(ctx spi.ComponentContext) bool {
	return isEvent(ctx, alertPodStuckTerminating, componentNamespace)
}

func isEvent(ctx spi.ComponentContext, alertName string, namespace string) bool {
	event := &v1.Event{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: generateAlertName(alertName)}, event)
	return err == nil
}
