// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqlcheck

import (
	"time"

	"github.com/verrazzano/verrazzano/pkg/log"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const channelBufferSize = 100

// MySQLChecker polls Verrazzano component availability every tickTime, and writes status updates to a secret
// It is the job of the Verrazzano controller to read availability status from the secret when updating
// Verrazzano status.
// The secret containing status is used for synchronization - multiple goroutines writing to the same object
// will cause performance degrading write conflicts.
type MySQLChecker struct {
	client   clipkg.Client
	tickTime time.Duration
	logger   *zap.SugaredLogger
	status   *AvailabilityStatus // Last known AvailabilityStatus
	shutdown chan int            // The channel on which shutdown signals are sent/received
}

type AvailabilityStatus struct {
	Components map[string]vzapi.ComponentAvailability
	Available  string
	Verrazzano *vzapi.Verrazzano
}

func NewMySQLChecker(c clipkg.Client, tick time.Duration) *MySQLChecker {
	return &MySQLChecker{
		client:   c,
		tickTime: tick,
		logger:   zap.S().With(log.FieldController, "mysqlcheck"),
	}
}

// Start starts the MySQLChecker if it is not already running.
// It is safe to call Start multiple times, additional goroutines will not be created
func (p *MySQLChecker) Start() {
	if p.shutdown != nil {
		// already running, so nothing to do
		return
	}
	p.shutdown = make(chan int, channelBufferSize)

	// goroutine updates availability every p.tickTime. If a shutdown signal is received (or channel is closed),
	// the goroutine returns.
	go func() {
		ticker := time.NewTicker(p.tickTime)
		for {
			select {
			case <-ticker.C:
				// timer event causes availability update
			case <-p.shutdown:
				// shutdown event causes termination
				ticker.Stop()
				return
			}
		}
	}()
}

// Pause pauses the MySQLChecker if it was running.
// It is safe to call Pause multiple times
func (p *MySQLChecker) Pause() {
	if p.shutdown != nil {
		close(p.shutdown)
		p.shutdown = nil
	}
}

func (a *AvailabilityStatus) merge(vz *vzapi.Verrazzano) {
	for component := range a.Components {
		if comp, ok := vz.Status.Components[component]; ok {
			available := a.Components[component]
			comp.Available = &available
		}
	}
	vz.Status.Available = &a.Available
}

func (a *AvailabilityStatus) needsUpdate() bool {
	if a.Verrazzano == nil {
		return true
	}
	for component := range a.Components {
		if comp, ok := a.Verrazzano.Status.Components[component]; ok {
			if comp.Available == nil || *comp.Available != a.Components[component] {
				return true
			}
		}
	}
	return a.Verrazzano.Status.Available == nil || *a.Verrazzano.Status.Available != a.Available
}
