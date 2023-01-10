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
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysqloperator.ComponentName,
			Namespace: mysqloperator.ComponentNamespace,
			Labels: map[string]string{
				"name": mysqloperator.ComponentName,
			},
		},
	}

	mySQLPod := &v1.Pod{
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
	err = mysqlCheck.RepairMySQLPodsWaitingReadinessGates()
	assert.NoError(t, err)
	assert.True(t, getLastTimeReadinessGateChecked().IsZero())
	pod := v1.Pod{}
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, &pod)
	assert.NoError(t, err)

	// Set one of the conditions to false
	mySQLPod.Status.Conditions = []v1.PodCondition{{Type: "gate1", Status: v1.ConditionTrue}, {Type: "gate2", Status: v1.ConditionFalse}}
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLPod, mySQLOperatorPod).Build()
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err = NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)

	// Expect timer to get started when ond of the conditions is not met.  The mysql-operator pod should still exist.
	err = mysqlCheck.RepairMySQLPodsWaitingReadinessGates()
	assert.NoError(t, err)
	assert.False(t, getLastTimeReadinessGateChecked().IsZero())
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, &pod)
	assert.NoError(t, err)

	// Set the last time readiness gate checked to exceed the RepairTimeout period.  Expect the mysql-operator to get recycled.
	setInitialTimeReadinessGateChecked(time.Now().Add(-time.Hour * 2))
	err = mysqlCheck.RepairMySQLPodsWaitingReadinessGates()
	assert.NoError(t, err, fmt.Sprintf("unexpected error: %v", err))
	assert.True(t, getLastTimeReadinessGateChecked().IsZero())
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, &pod)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
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
	innoDBCluster := newInnoDBCluster(innoDBClusterStatusOnline)
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLOperatorPod, innoDBCluster).Build()
	resetInitialTimeICUninstallChecked()
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)

	err := RepairICStuckDeleting(fakeCtx)
	assert.NoError(t, err)
	assert.True(t, getInitialTimeICUninstallChecked().IsZero())

	// Test first time calling with a deletion timestamp, the timer should get initialized
	// and the mysql-operator pod should not get deleted.
	innoDBCluster = newInnoDBCluster(innoDBClusterStatusOnline)
	startTime := metav1.Now()
	innoDBCluster.SetDeletionTimestamp(&startTime)
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLOperatorPod, innoDBCluster).Build()
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)

	assert.True(t, getInitialTimeICUninstallChecked().IsZero())
	err = RepairICStuckDeleting(fakeCtx)
	assert.Error(t, err)
	assert.False(t, getInitialTimeICUninstallChecked().IsZero())

	pod := v1.Pod{}
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, &pod)
	assert.NoError(t, err, "expected the mysql-operator pod to be found")

	// Call repair after the timer has started, but not expired.  Expect an error because the IC object is not deleted yet.
	err = RepairICStuckDeleting(fakeCtx)
	assert.Error(t, err)

	// Force the timer to be expired, expect the mysql-operator pod to be deleted
	setInitialTimeICUninstallChecked(time.Now().Add(-time.Hour * 2))
	err = RepairICStuckDeleting(fakeCtx)
	assert.NoError(t, err)

	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, &pod)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

	// If the IC object is already deleted, then no error should be returned
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLOperatorPod).Build()
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)
	err = RepairICStuckDeleting(fakeCtx)
	assert.NoError(t, err)

}

// TestRepairMySQLPodsStuckTerminating tests the temporary workaround for MySQL
// pods getting stuck terminating.
// GIVEN a pod is deleting
// WHEN still deleting after the expiration time period
// THEN recycle the mysql-operator
func TestRepairMySQLPodsStuckTerminating(t *testing.T) {
	mySQLOperatorPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysqloperator.ComponentName,
			Namespace: mysqloperator.ComponentNamespace,
			Labels: map[string]string{
				"name": mysqloperator.ComponentName,
			},
		},
	}

	mySQLPod0 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysql-0",
			Namespace: componentNamespace,
			Labels: map[string]string{
				mySQLComponentLabel: mySQLDComponentName,
			},
		},
	}

	mySQLPod1 := &v1.Pod{
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
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLPod0, mySQLPod1).Build()
	resetInitialTimeMySQLPodsStuckChecked()
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err := NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)

	err = mysqlCheck.RepairMySQLPodStuckDeleting()
	assert.NoError(t, err)
	assert.True(t, getInitialTimeMySQLPodsStuckChecked().IsZero())

	// Call with MySQL pods being deleted, first time expect timer to start
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLPod0, mySQLPod1, mySQLPod2).Build()
	resetInitialTimeMySQLPodsStuckChecked()
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err = NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)

	err = mysqlCheck.RepairMySQLPodStuckDeleting()
	assert.NoError(t, err)
	assert.False(t, getInitialTimeMySQLPodsStuckChecked().IsZero())

	// Call with MySQL pods being deleted and timer expired, expect mysql-operator pod to be deleted
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLOperatorPod, mySQLPod0, mySQLPod1, mySQLPod2).Build()
	setInitialTimeMySQLPodsStuckChecked(time.Now().Add(-time.Hour * 2))
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err = NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)

	err = mysqlCheck.RepairMySQLPodStuckDeleting()
	assert.NoError(t, err)
	assert.True(t, getInitialTimeMySQLPodsStuckChecked().IsZero())

	pod := v1.Pod{}
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, &pod)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

	// Call with no MySQL pods, expect success
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects().Build()
	resetInitialTimeMySQLPodsStuckChecked()
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err = NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)

	err = mysqlCheck.RepairMySQLPodStuckDeleting()
	assert.NoError(t, err)
	assert.True(t, getInitialTimeMySQLPodsStuckChecked().IsZero())
}

func newInnoDBCluster(status string) *unstructured.Unstructured {
	innoDBCluster := unstructured.Unstructured{}
	innoDBCluster.SetGroupVersionKind(innoDBClusterGVK)
	innoDBCluster.SetNamespace(componentNamespace)
	innoDBCluster.SetName(helmReleaseName)
	_ = unstructured.SetNestedField(innoDBCluster.Object, status, innoDBClusterStatusFields...)
	return &innoDBCluster
}
