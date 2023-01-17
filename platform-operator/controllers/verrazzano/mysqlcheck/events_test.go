// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqlcheck

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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

	eventName := "test"
	reason := "PodStuck"
	message := "Pod was stuck terminating"
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(mySQLRouterPod1).Build()
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)
	mysqlCheck, err := NewMySQLChecker(fakeCtx.Client(), checkPeriodDuration, timeoutDuration)
	assert.NoError(t, err)

	mysqlCheck.logEvent(mySQLRouterPod1, eventName, reason, message)
	event := &v1.Event{}
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: componentNamespace, Name: generateEventName(eventName)}, event)
	assert.NoError(t, err)
	assert.Equal(t, reason, event.Reason)
	assert.Equal(t, message, event.Message)
	commonEventAsserts(t, mySQLRouterPod1, event)

	// Log the same event again with a different involved object
	saveFirstTime := event.FirstTimestamp
	saveLastTime := event.LastTimestamp
	time.Sleep(2 * time.Second)
	mysqlCheck.logEvent(mySQLRouterPod2, eventName, reason, message)
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: componentNamespace, Name: generateEventName(eventName)}, event)
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
