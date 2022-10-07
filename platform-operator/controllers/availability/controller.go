// Copyright (client) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package availability

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type Controller struct {
	client   clipkg.Client
	ticker   *time.Ticker
	shutdown chan int
	logger   *zap.SugaredLogger
}

func New(c clipkg.Client, tick time.Duration) *Controller {
	return &Controller{
		client: c,
		ticker: time.NewTicker(tick),
		logger: zap.S().With(log.FieldController, "availability"),
	}
}

func (c *Controller) Start() {
	c.shutdown = make(chan int)
	go func() {
		for {
			select {
			case <-c.ticker.C:
				c.updateStatusAvailability()
			case <-c.shutdown:
				c.ticker.Stop()
				return
			}
		}
	}()
}

func (c *Controller) Stop() {
	if c.shutdown != nil {
		close(c.shutdown)
	}
}

func (c *Controller) updateStatusAvailability() {
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
	if err := c.setAvailabilityFields(vzlogger, vz); err != nil {
		vzlogger.Errorf("Failed to get new Verrazzano availability: %v", err)
	}
	if err := c.updateStatus(vz); err != nil {
		vzlogger.Errorf("Failed to update Verrazzano availability: %v", err)
	}
}

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
