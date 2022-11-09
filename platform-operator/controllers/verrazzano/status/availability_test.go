// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

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

const reldir = "../../../manifests/profiles"

type fakeComponent struct {
	name      string
	available vzapi.ComponentAvailability
	enabled   bool
	helm.HelmComponent
}

func (f fakeComponent) IsAvailable(_ spi.ComponentContext) (string, vzapi.ComponentAvailability) {
	return "", f.available
}

func (f fakeComponent) IsEnabled(_ runtime.Object) bool {
	return f.enabled
}

func (f fakeComponent) Name() string {
	return f.name
}

func newFakeComponent(name string, available vzapi.ComponentAvailability, enabled bool) fakeComponent {
	return fakeComponent{name: name, available: available, enabled: enabled}
}

func newTestHealthCheck(objs ...client.Object) *HealthChecker {
	c := fake.NewClientBuilder().WithObjects(objs...).WithScheme(testScheme).Build()
	updater := NewStatusUpdater(c)
	return NewHealthChecker(updater, c, 1*time.Second)
}

func TestGetComponentAvailability(t *testing.T) {
	config.TestProfilesDir = reldir
	defer func() { config.TestProfilesDir = "" }()
	var tests = []struct {
		f         fakeComponent
		state     vzapi.CompStateType
		available vzapi.ComponentAvailability
	}{
		{
			newFakeComponent("availableComponent", vzapi.ComponentAvailable, true),
			vzapi.CompStateReady,
			vzapi.ComponentAvailable,
		},
		{
			newFakeComponent("unreadyComponent", vzapi.ComponentAvailable, true),
			vzapi.CompStateInstalling,
			vzapi.ComponentUnavailable,
		},
		{
			newFakeComponent("unavailableComponent", vzapi.ComponentUnavailable, true),
			vzapi.CompStateReady,
			vzapi.ComponentUnavailable,
		},
	}

	c := newTestHealthCheck()
	vz := &vzapi.Verrazzano{}
	ctx := spi.NewFakeContext(c.client, vz, nil, false)
	for _, tt := range tests {
		t.Run(tt.f.name, func(t *testing.T) {
			a := c.getComponentAvailability(tt.f, tt.state, ctx)
			assert.Equal(t, tt.available, a.available)
		})
	}
}

func TestSetAvailabilityFields(t *testing.T) {
	config.TestProfilesDir = reldir
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
			[]spi.Component{newFakeComponent(rancher, vzapi.ComponentUnavailable, true)},
			"0/1",
		},
		{
			"enabled and available",
			[]spi.Component{newFakeComponent(rancher, vzapi.ComponentAvailable, true)},
			"1/1",
		},
		{
			"multiple components",
			[]spi.Component{
				newFakeComponent(rancher, vzapi.ComponentAvailable, true),
				newFakeComponent(opensearch, vzapi.ComponentAvailable, true),
				newFakeComponent(grafana, vzapi.ComponentUnavailable, true),
			},
			"2/3",
		},
	}

	p := newTestHealthCheck()
	log := vzlog.DefaultLogger()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vz := &vzapi.Verrazzano{
				Status: vzapi.VerrazzanoStatus{
					Components: map[string]*vzapi.ComponentStatusDetails{},
				},
			}
			for _, component := range tt.components {
				if component.IsEnabled(nil) {
					vz.Status.Components[component.Name()] = &vzapi.ComponentStatusDetails{
						State: vzapi.CompStateReady,
					}
				}
			}
			status, err := p.newStatus(log, vz, tt.components)
			assert.NoError(t, err)
			assert.NotNil(t, status)
		})
	}
}

func TestUpdateAvailability(t *testing.T) {
	config.TestProfilesDir = reldir
	defer func() { config.TestProfilesDir = "" }()
	var tests = []struct {
		name string
		vz   *vzapi.Verrazzano
	}{
		{
			"no verrazzano",
			nil,
		},
		{
			"verrazzano",
			&vzapi.Verrazzano{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p *HealthChecker
			if tt.vz != nil {
				p = newTestHealthCheck(tt.vz)
			} else {
				p = newTestHealthCheck()
			}

			err := p.updateAvailability([]spi.Component{})
			assert.NoError(t, err)
			assert.Equal(t, p.status == nil, tt.vz == nil)
		})
	}
}
