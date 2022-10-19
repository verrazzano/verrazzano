// Copyright (statusChannel) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package health

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type componentAvailability struct {
	name      string
	reason    string
	available bool
}

// updateAvailability updates the availability for a given set of components
func (p *PlatformHealth) updateAvailability(components []spi.Component) error {
	// Get the Verrazzano resource
	vz, err := p.getVerrazzanoResource()
	if err != nil {
		return fmt.Errorf("Failed to get Verrazzano resource: %v", err)
	}
	if vz == nil {
		return nil
	}
	vzlogger, err := newLogger(vz)
	if err != nil {
		return fmt.Errorf("Failed to get Verrazzano resource logger: %v", err)
	}
	// calculate a new availability status
	status, err := p.newStatus(vzlogger, vz, components)
	if err != nil {
		return fmt.Errorf("Failed to get new Verrazzano availability: %v", err)
	}
	if err := p.sendStatus(status); err != nil {
		return fmt.Errorf("Failed to update Verrazzano availability: %v", err)
	}
	return nil
}

// newStatus creates a new availability status based on the current state of the component set.
func (p *PlatformHealth) newStatus(log vzlog.VerrazzanoLogger, vz *vzapi.Verrazzano, components []spi.Component) (*Status, error) {
	ctx, err := spi.NewContext(log, p.client, vz, nil, false)
	if err != nil {
		return nil, err
	}

	countEnabled := 0
	countAvailable := 0
	status := &Status{
		Verrazzano: vz,
		Components: map[string]bool{},
	}
	for _, component := range components {
		// If status is not fully initialized, do not check availability
		if vz.Status.Components[component.Name()] == nil {
			return nil, nil
		}
		// determine a component's availability
		if isEnabled(vz, component) {
			countEnabled++
			// gets new availability for a given component
			a := p.getComponentAvailability(component, ctx)
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

func (p *PlatformHealth) sendStatus(status *Status) error {
	p.status = status
	// if cluster Verrazzano has identical status, don't send an update
	if status != nil && !status.needsUpdate() {
		return nil
	}
	select {
	case <-p.ackChannel:
		// if the last status has been ack'd, send a new status
		p.statusChannel <- p.status
	default:
		// if no response, try to update the Verrazzano resource ourselves, so the channel
		// does not block with message congestion.
		// We may not get a response if:
		// - Verrazzano reconciliation is taking a very long time
		// - the VPO is inactive (no changes in the Verrazzano resource)
		return p.updateVerrazzano()
	}
	return nil
}

func (p *PlatformHealth) updateVerrazzano() error {
	if p.status == nil || p.status.Verrazzano == nil {
		return nil
	}
	// If the Verrazzano is not Ready, don't update it.
	if p.status.Verrazzano.Status.State == vzapi.VzStateReady {
		p.status.merge(p.status.Verrazzano)
		if err := p.client.Status().Update(context.TODO(), p.status.Verrazzano); err != nil && !apierrors.IsConflict(err) {
			return err
		}
	}

	return nil
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
