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
	enabled   bool
	reason    string
	available bool
	err       error
}

// setAvailabilityFields loops through the provided components sets their availability set.
// The top level Verrazzano status.available field is set to (available components)/(enabled components).
func (c *Controller) setAvailabilityFields(log vzlog.VerrazzanoLogger, vz *vzapi.Verrazzano, components []spi.Component) error {
	ch := make(chan componentAvailability)
	for _, component := range components {
		comp := component
		go func() {
			ch <- c.getComponentAvailability(log, vz, comp)
		}()
	}
	countEnabled := 0
	countAvailable := 0
	for i := 0; i < len(components); i++ {
		a := <-ch
		if a.enabled {
			countEnabled++
		}
		if a.err != nil {
			// short-circuit on error, error is related to component context
			return fmt.Errorf("failed to get component availability: %v", a.err)
		}
		if vz.Status.Components[a.name] == nil {
			// component hasn't been reconciled by Verrazzano yet, skip
			continue
		}
		if a.available {
			countAvailable++
		}
		vz.Status.Components[a.name].Available = &a.available
	}
	// format the printer column with both values
	availabilityColumn := fmt.Sprintf("%d/%d", countAvailable, countEnabled)
	vz.Status.Available = &availabilityColumn
	log.Debugf("Set component availability: %s", availabilityColumn)
	return nil
}

// getComponentAvailability calculates componentAvailability for a given Verrazzano component
func (c *Controller) getComponentAvailability(log vzlog.VerrazzanoLogger, vz *vzapi.Verrazzano, component spi.Component) componentAvailability {
	name := component.Name()
	log.Debugf("Checking availability for component %s", name)
	ctx, err := spi.NewContext(log, c.client, vz, nil, false)
	enabled := component.IsEnabled(vz)
	a := componentAvailability{
		name:    name,
		enabled: enabled,
		err:     err,
	}
	if a.err == nil && a.enabled {
		reason, available := component.IsAvailable(ctx)
		a.reason = reason
		a.available = available
	}
	return a
}
