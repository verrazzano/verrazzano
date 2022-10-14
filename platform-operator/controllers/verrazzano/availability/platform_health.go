// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package availability

import (
	"github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// PlatformHealth polls Verrazzano component availability every tickTime, and writes status updates to a secret
// It is the job of the Verrazzano controller to read availability status from the secret when updating
// Verrazzano status.
// The secret containing status is used for synchronization - multiple goroutines writing to the same object
// will cause performance degrading write conflicts.
type PlatformHealth struct {
	client   clipkg.Client
	tickTime time.Duration
	shutdown chan int
	logger   *zap.SugaredLogger
	status   *Status      // Last known Status
	C        chan *Status // The channel on which Status is delivered
}

type Status struct {
	Components map[string]bool
	Available  string
}

func New(c clipkg.Client, tick time.Duration) *PlatformHealth {
	return &PlatformHealth{
		client:   c,
		tickTime: tick,
		logger:   zap.S().With(log.FieldController, "availability"),
		C:        make(chan *Status),
	}
}

// Start starts the PlatformHealth if it is not already running.
// It is safe to call Start multiple times, additional goroutines will not be created
func (p *PlatformHealth) Start() {
	if p.shutdown != nil {
		// already running, so nothing to do
		return
	}
	p.shutdown = make(chan int)

	// goroutine updates availability every p.tickTime. If a shutdown signal is received (or channel is closed),
	// the goroutine returns.
	go func() {
		ticker := time.NewTicker(p.tickTime)
		for {
			select {
			case <-ticker.C:
				// timer event causes availability update
				p.updateAvailability(registry.GetComponents())

			case <-p.shutdown:
				// shutdown event causes termination
				ticker.Stop()
				return
			}
		}
	}()
}

// Pause pauses the PlatformHealth if it was running.
// It is safe to call Pause multiple times
func (p *PlatformHealth) Pause() {
	if p.shutdown != nil {
		close(p.shutdown)
	}
}
