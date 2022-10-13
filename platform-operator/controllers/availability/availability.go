// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package availability

import (
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

// setAvailabilityFields loops through the provided components sets their availability set.
// The top level Verrazzano status.available field is set to (available components)/(enabled components).
func (c *Controller) setAvailabilityFields(log vzlog.VerrazzanoLogger, vz *vzapi.Verrazzano, components []spi.Component) (bool, error) {
	ch := make(chan componentAvailability)
	ctx, err := spi.NewContext(log, c.client, vz, nil, false)
	if err != nil {
		return false, err
	}

	countEnabled := 0
	countAvailable := 0

	for _, component := range components {
		// If status is not fully initialized, do not check availability
		if vz.Status.Components[component.Name()] == nil {
			close(ch)
			return false, nil
		}

		// determine a components availability
		if vz.Status.Components[component.Name()].State != vzapi.CompStateDisabled {
			countEnabled++
			comp := component
			go func() {
				ch <- c.getComponentAvailability(comp, ctx)
			}()
		}
	}

	// count available components and set component availability
	for i := 0; i < len(components); i++ {
		a := <-ch
		if a.available {
			countAvailable++
		}
		vz.Status.Components[a.name].Available = &a.available
	}
	// format the printer column with both values
	availabilityColumn := fmt.Sprintf("%d/%d", countAvailable, countEnabled)
	vz.Status.Available = &availabilityColumn
	log.Debugf("Set component availability: %s", availabilityColumn)
	return true, nil
}

// getComponentAvailability calculates componentAvailability for a given Verrazzano component
func (c *Controller) getComponentAvailability(component spi.Component, ctx spi.ComponentContext) componentAvailability {
	name := component.Name()
	ctx.Init(name)
	reason, available := component.IsAvailable(ctx)
	return componentAvailability{
		name:      name,
		reason:    reason,
		available: available,
	}
}
