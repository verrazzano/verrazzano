// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package healthcheck

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/metricsexporter"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const reldir = "../../../manifests/profiles"

type fakeComponent struct {
	name      string
	JSONName  string
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

func (f fakeComponent) GetJSONName() string {
	return f.JSONName
}

func newFakeComponent(name string, JSONname string, available vzapi.ComponentAvailability, enabled bool) fakeComponent {
	return fakeComponent{name: name, JSONName: JSONname, available: available, enabled: enabled}
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
			newFakeComponent("availableComponent", "JSONName", vzapi.ComponentAvailable, true),
			vzapi.CompStateReady,
			vzapi.ComponentAvailable,
		},
		{
			newFakeComponent("unreadyComponent", "JSONName", vzapi.ComponentAvailable, true),
			vzapi.CompStateInstalling,
			vzapi.ComponentUnavailable,
		},
		{
			newFakeComponent("unavailableComponent", "JSONName", vzapi.ComponentUnavailable, true),
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
	metricsexporter.Init()

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
		states     []vzapi.CompStateType
	}{
		{
			"no components, no availability",
			[]spi.Component{},
			zeroOfZero,
			[]vzapi.CompStateType{},
		},
		{
			"enabled but not available",
			[]spi.Component{newFakeComponent(rancher, rancher, vzapi.ComponentUnavailable, true)},
			"0/1",
			[]vzapi.CompStateType{vzapi.CompStateInstalling},
		},
		{
			"enabled and available",
			[]spi.Component{newFakeComponent(rancher, rancher, vzapi.ComponentAvailable, true)},
			"1/1",
			[]vzapi.CompStateType{vzapi.CompStateReady},
		},
		{
			"multiple components",
			[]spi.Component{
				newFakeComponent(rancher, rancher, vzapi.ComponentAvailable, true),
				newFakeComponent(opensearch, opensearch, vzapi.ComponentAvailable, true),
				newFakeComponent(grafana, grafana, vzapi.ComponentUnavailable, true),
			},
			"2/3",
			[]vzapi.CompStateType{vzapi.CompStateReady, vzapi.CompStateReady, vzapi.CompStateInstalling},
		},
		{
			"uninstalled component",
			[]spi.Component{
				newFakeComponent(grafana, grafana, vzapi.ComponentAvailable, false),
			},
			"0/0",
			[]vzapi.CompStateType{vzapi.CompStateUninstalled},
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
			for i, component := range tt.components {
				vz.Status.Components[component.Name()] = &vzapi.ComponentStatusDetails{
					State: tt.states[i],
				}
			}
			status, err := p.newStatus(log, vz, tt.components)
			assert.NoError(t, err)
			assert.NotNil(t, status)
			assert.Equal(t, tt.available, status.Available)

			// make sure any components with a state of "uninstalled" have their availability set to "unavailable"
			for i, component := range tt.components {
				if tt.states[i] == vzapi.CompStateUninstalled {
					assert.Equal(t, vzapi.ComponentUnavailable, string(status.Components[component.Name()]))
				}
			}
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
