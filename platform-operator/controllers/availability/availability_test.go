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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

type fakeComponent struct {
	name      string
	available bool
	helm.HelmComponent
}

func (f fakeComponent) IsAvailable(_ spi.ComponentContext) (string, bool) {
	return "", f.available
}

func (f fakeComponent) Name() string {
	return f.name
}

func newFakeComponent(name string, available bool) fakeComponent {
	return fakeComponent{name: name, available: available}
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
			newFakeComponent("availableComponent", true),
		},
		{
			newFakeComponent("unavailableComponent", false),
		},
	}

	c := newTestController()
	vz := &vzapi.Verrazzano{}
	ctx := spi.NewFakeContext(c.client, vz, nil, false)
	for _, tt := range tests {
		t.Run(tt.f.name, func(t *testing.T) {
			a := c.getComponentAvailability(tt.f, ctx)
			assert.Equal(t, tt.f.available, a.available)
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
			"enabled but not available",
			[]spi.Component{newFakeComponent(rancher, false)},
			"0/1",
		},
		{
			"enabled and available",
			[]spi.Component{newFakeComponent(rancher, true)},
			"1/1",
		},
		{
			"multiple components",
			[]spi.Component{
				newFakeComponent(rancher, true),
				newFakeComponent(opensearch, true),
				newFakeComponent(grafana, false),
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
			status, err := c.getNewStatus(log, vz, tt.components)
			assert.NoError(t, err)
			assert.Equal(t, tt.available, status.Available)
		})
	}
}
