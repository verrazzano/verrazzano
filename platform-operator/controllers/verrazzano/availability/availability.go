// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package availability

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

type componentAvailability struct {
	name      string
	reason    string
	available bool
}

// newStatus creates a new availability status based on the current state of the component set.
func (p *PlatformHealth) newStatus(log vzlog.VerrazzanoLogger, vz *vzapi.Verrazzano, components []spi.Component) (*Status, error) {
	ch := make(chan componentAvailability)
	ctx, err := spi.NewContext(log, p.client, vz, nil, false)
	if err != nil {
		return nil, err
	}

	countEnabled := 0
	countAvailable := 0

	for _, component := range components {
		// If status is not fully initialized, do not check availability
		if vz.Status.Components[component.Name()] == nil {
			return nil, nil
		}
		// determine a component's availability
		if isEnabled(vz, component) {
			countEnabled++
			comp := component
			go func() {
				ch <- p.getComponentAvailability(comp, ctx.Copy())
			}()
		}
	}

	status := &Status{
		Components: map[string]bool{},
	}
	// count available components and set component availability
	for _, component := range components {
		if isEnabled(vz, component) {
			a := <-ch
			if a.available {
				countAvailable++
			}
			status.Components[a.name] = a.available
		}
	}
	// format the printer column with both values
	availabilityColumn := fmt.Sprintf("%d/%d", countAvailable, countEnabled)
	status.Available = availabilityColumn
	log.Debugf("Set component availability: %s", availabilityColumn)
	return status, nil
}

func isEnabled(vz *vzapi.Verrazzano, component spi.Component) bool {
	return vz.Status.Components[component.Name()].State != vzapi.CompStateDisabled
}

// getComponentAvailability calculates componentAvailability for a given Verrazzano component
func (p *PlatformHealth) getComponentAvailability(component spi.Component, ctx spi.ComponentContext) componentAvailability {
	name := component.Name()
	ctx.Init(name)
	reason, available := component.IsAvailable(ctx)
	return componentAvailability{
		name:      name,
		reason:    reason,
		available: available,
	}
}

// updateAvailability updates the availability for a given set of components
func (p *PlatformHealth) updateAvailability(components []spi.Component) {
	// Get the Verrazzano resource
	vz, err := p.getVerrazzanoResource()
	if err != nil {
		p.logger.Errorf("Failed to get Verrazzano resource: %v", err)
		return
	}
	if vz == nil {
		return
	}
	vzlogger, err := newLogger(vz)
	if err != nil {
		p.logger.Errorf("Failed to get Verrazzano resource logger: %v", err)
		return
	}
	// calculate a new availability status
	status, err := p.newStatus(vzlogger, vz, components)
	if err != nil {
		vzlogger.Errorf("Failed to get new Verrazzano availability: %v", err)
		return
	}
	// only send an update if status has changed
	if p.status != status {
		p.status = status
		p.C <- p.status
	}
}

// getVerrazzanoResource fetches a Verrazzano resource, if one exists
func (p *PlatformHealth) getVerrazzanoResource() (*vzapi.Verrazzano, error) {
	vzList := &vzapi.VerrazzanoList{}
	if err := p.client.List(context.TODO(), vzList); err != nil {
		return nil, err
	}
	if len(vzList.Items) != 1 {
		return nil, nil
	}
	return &vzList.Items[0], nil
}

func newLogger(vz *vzapi.Verrazzano) (vzlog.VerrazzanoLogger, error) {
	return vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           vz.Name,
		Namespace:      vz.Namespace,
		ID:             string(vz.UID),
		Generation:     vz.Generation,
		ControllerName: "availability",
	})
}
