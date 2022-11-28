// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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
			Namespace: ComponentNamespace,
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

	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLPod, mySQLOperatorPod).Build()
	mysqlComp := NewComponent().(mysqlComponent)
	mysqlComp.LastTimeReadinessGateRepairStarted = &time.Time{}
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)

	// First time calling, expect timer to get initialized
	assert.True(t, mysqlComp.LastTimeReadinessGateRepairStarted.IsZero())
	err := mysqlComp.repairMySQLPodsWaitingReadinessGates(fakeCtx)
	assert.NoError(t, err)
	assert.False(t, mysqlComp.LastTimeReadinessGateRepairStarted.IsZero())

	// Second time calling, expect no error and mysql-operator pod to still exist
	err = mysqlComp.repairMySQLPodsWaitingReadinessGates(fakeCtx)
	assert.NoError(t, err)

	pod := v1.Pod{}
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, &pod)
	assert.NoError(t, err)

	// Third time calling, set the timer to exceed the expiration time which will force a check of the readiness gates.
	// The readiness gates will be set to true going into the call, so the mysql-operator should not get recycled.
	*mysqlComp.LastTimeReadinessGateRepairStarted = time.Now().Add(-time.Hour * 2)
	err = mysqlComp.repairMySQLPodsWaitingReadinessGates(fakeCtx)
	assert.NoError(t, err)

	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, &pod)
	assert.NoError(t, err)

	// Fourth time calling, set one of the readiness gates to false.  This should force deletion of the mysql-operator pod.
	// The timer should also get reset.
	mySQLPod.Status.Conditions = []v1.PodCondition{{Type: "gate1", Status: v1.ConditionTrue}, {Type: "gate2", Status: v1.ConditionFalse}}
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLPod, mySQLOperatorPod).Build()
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)
	*mysqlComp.LastTimeReadinessGateRepairStarted = time.Now().Add(-time.Hour * 2)
	err = mysqlComp.repairMySQLPodsWaitingReadinessGates(fakeCtx)
	assert.NoError(t, err, fmt.Sprintf("unexpected error: %v", err))
	assert.True(t, mysqlComp.LastTimeReadinessGateRepairStarted.IsZero())

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
	mysqlComp := NewComponent().(mysqlComponent)
	mysqlComp.initialTimeICUninstallChecked = &time.Time{}
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)

	err := mysqlComp.repairICStuckDeleting(fakeCtx)
	assert.NoError(t, err)
	assert.True(t, mysqlComp.GetInitialTimeICUninstallChecked().IsZero())

	// Test first time calling with a deletion timestamp, the timer should get initialized
	// and the mysql-operator pod should not get deleted.
	innoDBCluster = newInnoDBCluster(innoDBClusterStatusOnline)
	startTime := metav1.Now()
	innoDBCluster.SetDeletionTimestamp(&startTime)
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLOperatorPod, innoDBCluster).Build()
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)

	assert.True(t, mysqlComp.GetInitialTimeICUninstallChecked().IsZero())
	err = mysqlComp.repairICStuckDeleting(fakeCtx)
	assert.Error(t, err)
	assert.False(t, mysqlComp.GetInitialTimeICUninstallChecked().IsZero())

	pod := v1.Pod{}
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, &pod)
	assert.NoError(t, err, "expected the mysql-operator pod to be found")

	// Call repair after the timer has started, but not expired.  Expect an error because the IC object is not deleted yet.
	err = mysqlComp.repairICStuckDeleting(fakeCtx)
	assert.Error(t, err)

	// Force the timer to be expired, expect the mysql-operator pod to be deleted
	mysqlComp.SetInitialTimeICUninstallChecked(time.Now().Add(-time.Hour * 2))
	err = mysqlComp.repairICStuckDeleting(fakeCtx)
	assert.NoError(t, err)

	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, &pod)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

	// If the IC object is already deleted, then no error should be returned
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLOperatorPod).Build()
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)
	err = mysqlComp.repairICStuckDeleting(fakeCtx)
	assert.NoError(t, err)

}

// TestRepairMySQLPodsWaitingReadinessGates tests the temporary workaround for MySQL
// pods getting stuck during install waiting for all readiness gates to be true.
// GIVEN a MySQL Pod with readiness gates defined
// WHEN they are not all ready after a given time period
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
			Namespace: ComponentNamespace,
			Labels: map[string]string{
				mySQLComponentLabel: mySQLDComponentName,
			},
		},
	}

	mySQLPod1 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysql-1",
			Namespace: ComponentNamespace,
			Labels: map[string]string{
				mySQLComponentLabel: mySQLDComponentName,
			},
		},
	}

	mySQLPod2DeleteTime := metav1.Now()
	mySQLPod2 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysql-2",
			Namespace: ComponentNamespace,
			Labels: map[string]string{
				mySQLComponentLabel: mySQLDComponentName,
			},
			DeletionTimestamp: &mySQLPod2DeleteTime,
		},
	}

	// Call with no MySQL pods being deleted, expect success
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLPod0, mySQLPod1).Build()
	mysqlComp := NewComponent().(mysqlComponent)
	mysqlComp.initialTimeMySQLPodsStuckChecked = &time.Time{}
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)

	err := mysqlComp.repairMySQLPodStuckDeleting(fakeCtx)
	assert.NoError(t, err)
	assert.True(t, mysqlComp.GetInitialTimeMySQLPodsStuckChecked().IsZero())

	// Call with MySQL pods being deleted, first time expect timer to start
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLPod0, mySQLPod1, mySQLPod2).Build()
	mysqlComp = NewComponent().(mysqlComponent)
	mysqlComp.initialTimeMySQLPodsStuckChecked = &time.Time{}
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)

	err = mysqlComp.repairMySQLPodStuckDeleting(fakeCtx)
	assert.Error(t, err)
	assert.False(t, mysqlComp.GetInitialTimeMySQLPodsStuckChecked().IsZero())

	// Call with MySQL pods being deleted and timer expired, expect mysql-operator pod to be deleted
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLOperatorPod, mySQLPod0, mySQLPod1, mySQLPod2).Build()
	mysqlComp = NewComponent().(mysqlComponent)
	mysqlComp.initialTimeMySQLPodsStuckChecked = &time.Time{}
	mysqlComp.SetInitialTimeMySQLPodsStuckChecked(time.Now().Add(-time.Hour * 2))
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)

	err = mysqlComp.repairMySQLPodStuckDeleting(fakeCtx)
	assert.NoError(t, err)
	assert.True(t, mysqlComp.GetInitialTimeMySQLPodsStuckChecked().IsZero())

	pod := v1.Pod{}
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: mysqloperator.ComponentNamespace, Name: mysqloperator.ComponentName}, &pod)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

	// Call with no MySQL pods, expect success
	cli = fake.NewClientBuilder().WithScheme(testScheme).WithObjects().Build()
	mysqlComp = NewComponent().(mysqlComponent)
	mysqlComp.initialTimeMySQLPodsStuckChecked = &time.Time{}
	fakeCtx = spi.NewFakeContext(cli, nil, nil, false)

	err = mysqlComp.repairMySQLPodStuckDeleting(fakeCtx)
	assert.NoError(t, err)
	assert.True(t, mysqlComp.GetInitialTimeMySQLPodsStuckChecked().IsZero())
}
