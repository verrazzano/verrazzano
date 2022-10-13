// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package availability

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type componentAvailability struct {
	name      string
	reason    string
	available bool
}

type Status struct {
	Components map[string]bool
	Available  string
}

const (
	statusObjectName = "verrazzano-%s-availability"
	statusKey        = "status"
)

func GetStatus(c client.Client, vz *vzapi.Verrazzano) (*Status, error) {
	statusObject := &corev1.Secret{}
	if err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: vz.Namespace,
		Name:      getStatusObjectName(vz),
	}, statusObject); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	data, ok := statusObject.Data[statusKey]
	if !ok {
		return nil, nil
	}
	status := &Status{}
	if err := json.Unmarshal(data, status); err != nil {
		return nil, err
	}
	return status, nil
}

func DeleteStatus(c client.Client, vz *vzapi.Verrazzano) error {
	statusObject := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getStatusObjectName(vz),
			Namespace: vz.Namespace,
		},
	}
	if err := c.Delete(context.TODO(), statusObject); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func getStatusObjectName(vz *vzapi.Verrazzano) string {
	return fmt.Sprintf(statusObjectName, vz.Name)
}

// getNewStatus loops through the provided components sets their availability set.
// The top level Verrazzano status.available field is set to (available components)/(enabled components).
func (c *Controller) getNewStatus(log vzlog.VerrazzanoLogger, vz *vzapi.Verrazzano, components []spi.Component) (*Status, error) {
	ch := make(chan componentAvailability)
	ctx, err := spi.NewContext(log, c.client, vz, nil, false)
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
				ch <- c.getComponentAvailability(comp, ctx.Copy())
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

func (c *Controller) updateStatus(status *Status, vz *vzapi.Verrazzano) error {
	statusObject := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vz.Namespace,
			Name:      getStatusObjectName(vz),
		},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), c.client, statusObject, func() error {
		if status == nil {
			statusObject.Data = nil
			return nil
		}
		data, err := json.Marshal(status)
		if err != nil {
			return err
		}
		statusObject.Data = map[string][]byte{
			statusKey: data,
		}
		return nil
	})
	return err
}

func isEnabled(vz *vzapi.Verrazzano, component spi.Component) bool {
	return vz.Status.Components[component.Name()].State != vzapi.CompStateDisabled
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
