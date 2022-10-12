// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package availability

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

type fakeComponent struct {
	name      string
	available bool
	enabled   bool
	helm.HelmComponent
}

func (f fakeComponent) IsEnabled(_ runtime.Object) bool {
	return f.enabled
}

func (f fakeComponent) IsAvailable(_ spi.ComponentContext) (string, bool) {
	return "", f.available
}

func (f fakeComponent) Name() string {
	return f.name
}

func newFakeComponent(name string, available, enabled bool) fakeComponent {
	return fakeComponent{name: name, available: available, enabled: enabled}
}

func newTestController(objs ...client.Object) *Controller {
	return New(fake.NewClientBuilder().WithObjects(objs...).Build(), 2*time.Second)
}

func TestGetComponentAvailability(t *testing.T) {
	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()
	var tests = []struct {
		f fakeComponent
	}{
		{
			newFakeComponent("availableComponent", true, true),
		},
		{
			newFakeComponent("unavailableComponent", false, true),
		},
		{
			newFakeComponent("disabledComponent", true, false),
		},
	}

	c := newTestController()
	vz := &vzapi.Verrazzano{}
	ctx := spi.NewFakeContext(c.client, vz, nil, false)
	for _, tt := range tests {
		t.Run(tt.f.name, func(t *testing.T) {
			a := c.getComponentAvailability(vz, tt.f, ctx)
			assert.Equal(t, tt.f.enabled, a.enabled)
			if a.enabled {
				assert.Equal(t, tt.f.available, a.available)
			} else {
				assert.False(t, a.available)
			}
		})
	}
}

func TestSetAvailabilityFields(t *testing.T) {
	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()
	zeroOfZero := "0/0"
	rancher := "rancher"
	opensearch := "opensearch"
	grafana := "grafana"

	var tests = []struct {
		name       string
		components []spi.Component
		available  string
	}{
		{
			"no components, no availability",
			[]spi.Component{},
			zeroOfZero,
		},
		{
			"no enabled components, no availability",
			[]spi.Component{newFakeComponent(rancher, false, false)},
			zeroOfZero,
		},
		{
			"enabled but not available",
			[]spi.Component{newFakeComponent(rancher, false, true)},
			"0/1",
		},
		{
			"enabled and available",
			[]spi.Component{newFakeComponent(rancher, true, true)},
			"1/1",
		},
		{
			"multiple components",
			[]spi.Component{
				newFakeComponent(rancher, true, true),
				newFakeComponent(opensearch, true, true),
				newFakeComponent(grafana, false, true),
			},
			"2/3",
		},
	}

	c := newTestController()
	log := vzlog.DefaultLogger()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vz := &vzapi.Verrazzano{
				Status: vzapi.VerrazzanoStatus{
					Components: map[string]*vzapi.ComponentStatusDetails{},
				},
			}
			for _, component := range tt.components {
				vz.Status.Components[component.Name()] = &vzapi.ComponentStatusDetails{}
			}
			err := c.setAvailabilityFields(log, vz, tt.components)
			assert.NoError(t, err)
			assert.Equal(t, tt.available, *vz.Status.Available)
		})
	}
}
