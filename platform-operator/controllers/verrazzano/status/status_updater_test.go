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

func TestStatusUpdater(t *testing.T) {

	vz := &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "a",
			Name:      "b",
		},
	}
	testScheme := runtime.NewScheme()
	_ = vzapi.AddToScheme(testScheme)
	c := fake.NewClientBuilder().WithObjects(vz).WithScheme(testScheme).Build()
	updater := NewStatusUpdater(c)
	version := "1.5.0"
	availability := &AvailabilityStatus{
		Components: map[string]bool{
			fluentd.ComponentName: true,
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
	time.Sleep(timeout) // updater takes a bit to initialize...
	updater.Update(u)
	retryFunction(func(i int) bool {
		return checkUpdate(t, c, u, i)
	})

	available := true
	u2 := &UpdateEvent{
		Verrazzano: vz,
		Components: map[string]*vzapi.ComponentStatusDetails{
			opensearch.ComponentName: &vzapi.ComponentStatusDetails{
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
	updater.Update(u2)
	retryFunction(func(i int) bool {
		return checkUpdate(t, c, u2, i)
	})
}

func checkUpdate(t *testing.T, c client.Client, u *UpdateEvent, i int) bool {
	vz := &vzapi.Verrazzano{}
	if err := c.Get(context.TODO(), types.NamespacedName{
		Name:      u.Verrazzano.Name,
		Namespace: u.Verrazzano.Namespace,
	}, vz); err != nil {
		assert.Failf(t, "Failed to get Verrazzano resource: %s", err.Error())
	}

	vzCopy := vz.DeepCopy()
	u.merge(vzCopy)
	equal := reflect.DeepEqual(vzCopy.Status, vz.Status)
	if equal {
		return true
	}
	if i >= retries {
		assert.Fail(t, "Status not equal after update")
		return true
	}
	return false
}

func retryFunction(f func(i int) bool) {
	for i := 0; i < retries; i++ {
		ok := f(i)
		if ok {
			break
		}
		time.Sleep(timeout)
	}
}
