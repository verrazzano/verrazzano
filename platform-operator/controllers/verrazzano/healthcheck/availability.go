// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package healthcheck

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/metricsexporter"
	"go.uber.org/zap/zapcore"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type componentAvailability struct {
	name      string
	reason    string
	available vzapi.ComponentAvailability
}

// updateAvailability updates the availability for a given set of components
func (p *HealthChecker) updateAvailability(components []spi.Component) error {
	// Get the Verrazzano resource
	vz, err := getVerrazzanoResource(p.client)
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
	p.sendStatus(status)
	return nil
}

// newStatus creates a new availability status based on the current state of the component set.
func (p *HealthChecker) newStatus(log vzlog.VerrazzanoLogger, vz *vzapi.Verrazzano, components []spi.Component) (*AvailabilityStatus, error) {
	ctx, err := spi.NewContext(log, p.client, vz, nil, false)
	if err != nil {
		return nil, err
	}

	countEnabled := 0
	countAvailable := 0
	status := &AvailabilityStatus{
		Verrazzano: vz,
		Components: map[string]vzapi.ComponentAvailability{},
	}
	for _, component := range components {
		// If component is not in the metricsMap, move on to the next component
		if metricsexporter.IsNonMetricComponent(component.Name()) {
			continue
		}
		// If status is not fully initialized, do not check availability
		componentStatus, ok := vz.Status.Components[component.Name()]
		if !ok {
			return nil, nil
		}
		// determine a component's availability
		isEnabled := component.IsEnabled(ctx.EffectiveCR())
		if isEnabled {
			countEnabled++
			// gets new availability for a given component
			a := p.getComponentAvailability(component, componentStatus.State, ctx)
			if a.available == vzapi.ComponentAvailable {
				countAvailable++
			}
			// update the component availability metric
			err := metricsexporter.SetComponentAvailabilityMetric(component.GetJSONName(), a.available, isEnabled)
			if err != nil {
				return nil, err
			}

			status.Components[a.name] = a.available
		}
	}

	// publish the associated available/enabled component metrics
	availableComp, err := metricsexporter.GetSimpleGaugeMetric(metricsexporter.AvailableComponents)
	if err != nil {
		return nil, err
	}
	availableComp.Set(float64(countAvailable))
	enabledComp, err := metricsexporter.GetSimpleGaugeMetric(metricsexporter.EnabledComponents)
	if err != nil {
		return nil, err
	}
	enabledComp.Set(float64(countEnabled))

	// format the printer column with both values
	availabilityColumn := fmt.Sprintf("%d/%d", countAvailable, countEnabled)
	status.Available = availabilityColumn
	log.Debugf("Set component availability: %s", availabilityColumn)
	return status, nil
}

func (p *HealthChecker) sendStatus(status *AvailabilityStatus) {
	p.status = status
	// if cluster Verrazzano has identical status, don't send an update
	if p.status == nil || !p.status.needsUpdate() {
		return
	}
	p.updater.Update(&UpdateEvent{
		Availability: p.status,
	})
}

// getComponentAvailability calculates componentAvailability for a given Verrazzano component
func (p *HealthChecker) getComponentAvailability(component spi.Component, componentState vzapi.CompStateType, ctx spi.ComponentContext) componentAvailability {
	name := component.Name()
	ctx.Init(name)
	var available vzapi.ComponentAvailability = vzapi.ComponentAvailable
	var reason string
	// if a component isn't ready, it's not available
	if componentState != vzapi.CompStateReady {
		available = vzapi.ComponentUnavailable
		reason = fmt.Sprintf("component is %s", componentState)
	}
	// if a component is ready, check if it's available
	if available == vzapi.ComponentAvailable {
		reason, available = component.IsAvailable(ctx)
	}
	return componentAvailability{
		name:      name,
		reason:    reason,
		available: available,
	}
}

// getVerrazzanoResource fetches a Verrazzano resource, if one exists
func getVerrazzanoResource(client clipkg.Client) (*vzapi.Verrazzano, error) {
	vzList := &vzapi.VerrazzanoList{}
	if err := client.List(context.TODO(), vzList); err != nil {
		return nil, err
	}
	if len(vzList.Items) != 1 {
		return nil, nil
	}
	return &vzList.Items[0], nil
}

func newLogger(vz *vzapi.Verrazzano) (vzlog.VerrazzanoLogger, error) {
	zaplog, err := log.BuildZapLoggerWithLevel(2, zapcore.ErrorLevel)
	if err != nil {
		return nil, err
	}
	// The ID below needs to be different from the main thread, so add a suffix"
	return vzlog.ForZapLogger(&vzlog.ResourceConfig{
		Name:           vz.Name,
		Namespace:      vz.Namespace,
		ID:             string(vz.UID) + "health",
		Generation:     vz.Generation,
		ControllerName: "availability",
	}, zaplog), nil
}
