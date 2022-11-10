// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"context"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearch"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

const (
	timeout = 1 * time.Second
	retries = 5
)

var testvz *vzapi.Verrazzano
var testScheme *runtime.Scheme

func init() {
	testScheme = runtime.NewScheme()
	_ = vzapi.AddToScheme(testScheme)
	testvz = &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "a",
			Name:      "b",
		},
	}
}

func TestFakeStatusUpdater(t *testing.T) {
	vz := testvz.DeepCopy()
	c := fake.NewClientBuilder().WithObjects(vz).WithScheme(testScheme).Build()
	f := FakeVerrazzanoStatusUpdater{Client: c}
	vz.Status.State = vzapi.VzStateReady
	f.Update(&UpdateEvent{Verrazzano: vz})
	updatedVZ := &vzapi.Verrazzano{}
	if err := c.Get(context.TODO(), types.NamespacedName{
		Name:      vz.Name,
		Namespace: vz.Namespace,
	}, updatedVZ); err != nil {
		assert.Failf(t, "%v", err.Error())
	}
	assert.Equal(t, vzapi.VzStateReady, updatedVZ.Status.State)
}

func TestStatusUpdater(t *testing.T) {
	vz := testvz.DeepCopy()
	c := fake.NewClientBuilder().WithObjects(vz).WithScheme(testScheme).Build()
	updater := NewStatusUpdater(c)
	updater.Start()     // should be able to call start multiple times out issue
	time.Sleep(timeout) // updater takes some time to initialize
	version := "1.5.0"
	availability := &AvailabilityStatus{
		Components: map[string]vzapi.ComponentAvailability{
			fluentd.ComponentName: vzapi.ComponentAvailable,
		},
		Available:  "12/12",
		Verrazzano: vz,
	}
	conditions := []vzapi.Condition{
		{
			Type:               vzapi.CondInstallStarted,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: "x",
			Message:            "y",
		},
	}
	url := "myurl.com"
	instanceInfo := &vzapi.InstanceInfo{
		ConsoleURL:    &url,
		KeyCloakURL:   &url,
		RancherURL:    &url,
		ElasticURL:    &url,
		KibanaURL:     &url,
		GrafanaURL:    &url,
		PrometheusURL: &url,
		KialiURL:      &url,
		JaegerURL:     &url,
	}
	u := &UpdateEvent{
		Verrazzano:   vz,
		Version:      &version,
		State:        vzapi.VzStateReady,
		Conditions:   conditions,
		Availability: availability,
		InstanceInfo: instanceInfo,
		Components:   nil,
	}
	// Send an update
	updater.Update(u)
	retryFunction(t, func() bool {
		return checkUpdate(t, c, u)
	})

	var available vzapi.ComponentAvailability = vzapi.ComponentAvailable
	u2 := &UpdateEvent{
		Verrazzano: vz,
		Components: map[string]*vzapi.ComponentStatusDetails{
			opensearch.ComponentName: {
				Name:                     opensearch.ComponentName,
				Conditions:               []vzapi.Condition{},
				State:                    vzapi.CompStateReady,
				Available:                &available,
				Version:                  "x",
				LastReconciledGeneration: 1,
				ReconcilingGeneration:    1,
			},
		},
	}
	// Send another update
	updater.Update(u2)
	retryFunction(t, func() bool {
		return checkUpdate(t, c, u2)
	})
	// Closes the update channel
	updater.Update(nil)
	retryFunction(t, func() bool {
		return updater.updateChannel == nil
	})
}

func checkUpdate(t *testing.T, c client.Client, u *UpdateEvent) bool {
	vz := &vzapi.Verrazzano{}
	if err := c.Get(context.TODO(), types.NamespacedName{
		Name:      u.Verrazzano.Name,
		Namespace: u.Verrazzano.Namespace,
	}, vz); err != nil {
		assert.Failf(t, "Failed to get Verrazzano resource: %s", err.Error())
	}

	vzCopy := vz.DeepCopy()
	u.merge(vzCopy)
	return reflect.DeepEqual(vzCopy.Status, vz.Status)
}

func retryFunction(t *testing.T, f func() bool) {
	for i := 0; i < retries; i++ {
		ok := f()
		if ok {
			break
		}
		if i >= retries {
			assert.Fail(t, "Timed out")
			break
		}
		time.Sleep(timeout)
	}
}
