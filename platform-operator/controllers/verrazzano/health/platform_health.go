// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package health

import (
	"github.com/verrazzano/verrazzano/pkg/log"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const channelBufferSize = 10

// PlatformHealth polls Verrazzano component availability every tickTime, and writes status updates to a secret
// It is the job of the Verrazzano controller to read availability status from the secret when updating
// Verrazzano status.
// The secret containing status is used for synchronization - multiple goroutines writing to the same object
// will cause performance degrading write conflicts.
type PlatformHealth struct {
	client        clipkg.Client
	tickTime      time.Duration
	logger        *zap.SugaredLogger
	status        *Status      // Last known Status
	statusChannel chan *Status // The channel on which Status is sent/received
	shutdown      chan int     // The channel on which shutdown signals are sent/received
	ackChannel    chan bool    // The channel on which acknowledgements are sent/received
}

type Status struct {
	Components map[string]bool
	Available  string
	Verrazzano *vzapi.Verrazzano
}

func New(c clipkg.Client, tick time.Duration) *PlatformHealth {
	return &PlatformHealth{
		client:        c,
		tickTime:      tick,
		logger:        zap.S().With(log.FieldController, "availability"),
		statusChannel: make(chan *Status, channelBufferSize),
		ackChannel:    make(chan bool, channelBufferSize),
	}
}

// Start starts the PlatformHealth if it is not already running.
// It is safe to call Start multiple times, additional goroutines will not be created
func (p *PlatformHealth) Start() {
	if p.shutdown != nil {
		// already running, so nothing to do
		return
	}
	p.shutdown = make(chan int, channelBufferSize)

	// goroutine updates availability every p.tickTime. If a shutdown signal is received (or channel is closed),
	// the goroutine returns.
	go func() {
		ticker := time.NewTicker(p.tickTime)
		p.ackChannel <- true
		for {
			select {
			case <-ticker.C:
				// timer event causes availability update
				err := p.updateAvailability(registry.GetComponents())
				if err != nil {
					p.logger.Errorf("%v", err)
				}
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

// SetAvailabilityStatus adds health check status to a Verrazzano resource, if it is available
func (p *PlatformHealth) SetAvailabilityStatus(vz *vzapi.Verrazzano) {
	select {
	case status := <-p.statusChannel:
		if status != nil {
			status.merge(vz)
		}
		p.ackChannel <- true
	default:
		return
	}
}

func (a *Status) merge(vz *vzapi.Verrazzano) {
	for component := range a.Components {
		if comp, ok := vz.Status.Components[component]; ok {
			available := a.Components[component]
			comp.Available = &available
		}
	}
	vz.Status.Available = &a.Available
}

func (a *Status) needsUpdate() bool {
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
