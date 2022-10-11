// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package availability

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

type componentAvailability struct {
	name      string
	enabled   bool
	reason    string
	available bool
	err       error
}

func (c *Controller) setAvailabilityFields(log vzlog.VerrazzanoLogger, vz *vzapi.Verrazzano) error {
	ch := make(chan componentAvailability)
	components := registry.GetComponents()
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
		vz.Status.Components[a.name].Available = a.available
	}
	// format the printer column with both values
	availabilityColumn := fmt.Sprintf("%d/%d", countAvailable, countEnabled)
	vz.Status.Available = &availabilityColumn
	return nil
}

func (c *Controller) getComponentAvailability(log vzlog.VerrazzanoLogger, vz *vzapi.Verrazzano, component spi.Component) componentAvailability {
	ctx, err := spi.NewContext(log, c.client, vz, nil, false)
	name := component.Name()
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
