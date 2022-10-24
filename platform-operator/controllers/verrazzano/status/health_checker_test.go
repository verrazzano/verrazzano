// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"testing"
	"time"
)

func TestAddStatus(t *testing.T) {
	config.TestProfilesDir = reldir
	defer func() { config.TestProfilesDir = "" }()
	waitTime := 5 * time.Second

	vz := &vzapi.Verrazzano{}
	p := newTestHealthCheck(vz)
	assert.Nil(t, p.status)
	p.Start()
	time.Sleep(waitTime)
	// No components are ready, check fails
	assert.Nil(t, p.status)
	p.Pause()

	vz.Status.Components = map[string]*vzapi.ComponentStatusDetails{}
	for _, component := range registry.GetComponents() {
		vz.Status.Components[component.Name()] = &vzapi.ComponentStatusDetails{}
	}
	p = newTestHealthCheck(vz)
	p.Start()
	time.Sleep(waitTime)
	p.Pause()
	assert.NotNil(t, p.status)
	vz, err := getVerrazzanoResource(p.client)
	assert.NoError(t, err)
	assert.NotNil(t, vz.Status.Components)
}

func TestStart(t *testing.T) {
	p := newTestHealthCheck()
	p.Start()
	p.Start()
	p.Pause()
	p.Pause()
}
