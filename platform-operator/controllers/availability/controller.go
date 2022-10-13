// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package availability

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type Controller struct {
	client   clipkg.Client
	tickTime time.Duration
	shutdown chan int
	logger   *zap.SugaredLogger
}

func New(c clipkg.Client, tick time.Duration) *Controller {
	return &Controller{
		client:   c,
		tickTime: tick,
		logger:   zap.S().With(log.FieldController, "availability"),
	}
}

// Start starts the Controller if it is not already running.
// It is safe to call Start multiple times, additional goroutines will not be created
func (c *Controller) Start() {
	if c.shutdown != nil {
		// already running, so nothing to do
		return
	}
	c.shutdown = make(chan int)
	go func() {
		ticker := time.NewTicker(c.tickTime)
		for {
			select {
			case <-ticker.C:
				c.updateStatusAvailability(registry.GetComponents())
			case <-c.shutdown:
				ticker.Stop()
				return
			}
		}
	}()
}

// Pause pauses the Controller if it was running.
// It is safe to call Pause multiple times
func (c *Controller) Pause() {
	if c.shutdown != nil {
		close(c.shutdown)
	}
}

func (c *Controller) updateStatusAvailability(components []spi.Component) {
	// Get the Verrazzano resource
	vz, err := c.getVerrazzanoResource()
	if err != nil {
		c.logger.Errorf("Failed to get Verrazzano resource: %v", err)
		return
	}
	if vz == nil {
		return
	}
	vzlogger, err := newLogger(vz)
	if err != nil {
		c.logger.Errorf("Failed to get Verrazzano resource logger: %v", err)
		return
	}
	// Set availability fields in the provided Verrazzano CR
	update, err := c.setAvailabilityFields(vzlogger, vz, components)
	if err != nil {
		vzlogger.Errorf("Failed to get new Verrazzano availability: %v", err)
		return
	}
	// Persist the updated Verrazzano CR
	if update {
		if err := c.updateStatus(vz); err != nil {
			vzlogger.Errorf("Failed to update Verrazzano availability: %v", err)
		}
	}
}

// getVerrazzanoResource fetches a Verrazzano resource, if one exists
func (c *Controller) getVerrazzanoResource() (*vzapi.Verrazzano, error) {
	vzList := &vzapi.VerrazzanoList{}
	if err := c.client.List(context.TODO(), vzList); err != nil {
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

func (c *Controller) updateStatus(vz *vzapi.Verrazzano) error {
	err := c.client.Status().Update(context.TODO(), vz)
	// if there's a resource conflict, return nil to try again later
	if err != nil && !apierrors.IsConflict(err) {
		return err
	}
	return nil
}
