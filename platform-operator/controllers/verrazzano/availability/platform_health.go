// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package availability

import (
	"github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

// PlatformHealth polls Verrazzano component availability every tickTime, and writes status updates to a secret
// It is the job of the Verrazzano controller to read availability status from the secret when updating
// Verrazzano status.
// The secret containing status is used for synchronization - multiple goroutines writing to the same object
// will cause performance degrading write conflicts.
type PlatformHealth struct {
	client     clipkg.Client
	tickTime   time.Duration
	shutdown   chan int
	logger     *zap.SugaredLogger
	statusLock *sync.RWMutex
	status     *Status
}

type Status struct {
	Components map[string]bool
	Available  string
}

func New(c clipkg.Client, tick time.Duration) *PlatformHealth {
	return &PlatformHealth{
		client:     c,
		tickTime:   tick,
		logger:     zap.S().With(log.FieldController, "availability"),
		statusLock: &sync.RWMutex{},
	}
}

// Start starts the PlatformHealth if it is not already running.
// It is safe to call Start multiple times, additional goroutines will not be created
func (c *PlatformHealth) Start() {
	if c.shutdown != nil {
		// already running, so nothing to do
		return
	}
	c.shutdown = make(chan int)

	// goroutine updates availability every c.tickTime. If a shutdown signal is received (or channel is closed),
	// the goroutine returns.
	go func() {
		ticker := time.NewTicker(c.tickTime)
		for {
			select {
			case <-ticker.C:
				// timer event causes availability update
				c.updateAvailability(registry.GetComponents())

			case <-c.shutdown:
				// shutdown event causes termination
				ticker.Stop()
				return
			}
		}
	}()
}

// Pause pauses the PlatformHealth if it was running.
// It is safe to call Pause multiple times
func (c *PlatformHealth) Pause() {
	if c.shutdown != nil {
		close(c.shutdown)
	}
}

func (c *PlatformHealth) Get() *Status {
	c.statusLock.RLock()
	defer c.statusLock.RUnlock()
	return c.status
}

func (c *PlatformHealth) Clear() {
	c.update(nil)
}
