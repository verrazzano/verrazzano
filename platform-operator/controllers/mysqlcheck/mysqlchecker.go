// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqlcheck

import (
	"time"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	controllerName    = "MySQLChecker"
	channelBufferSize = 100
)

// MySQLChecker polls Verrazzano component availability every tickTime, and writes status updates to a secret
// It is the job of the Verrazzano controller to read availability status from the secret when updating
// Verrazzano status.
// The secret containing status is used for synchronization - multiple goroutines writing to the same object
// will cause performance degrading write conflicts.
type MySQLChecker struct {
	client   clipkg.Client
	tickTime time.Duration
	log      vzlog.VerrazzanoLogger
	shutdown chan int // The channel on which shutdown signals are sent/received
}

func NewMySQLChecker(c clipkg.Client, tick time.Duration) (*MySQLChecker, error) {
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           helmReleaseName,
		Namespace:      componentNamespace,
		ID:             controllerName,
		Generation:     0,
		ControllerName: controllerName,
	})
	if err != nil {
		zap.S().Errorf("Failed to create resource logger for %s: %v", controllerName, err)
		return nil, err
	}

	return &MySQLChecker{
		client:   c,
		tickTime: tick,
		log:      log,
	}, nil
}

// Start starts the MySQLChecker if it is not already running.
// It is safe to call Start multiple times, additional goroutines will not be created
func (mc *MySQLChecker) Start() {
	if mc.shutdown != nil {
		// already running, so nothing to do
		return
	}
	mc.shutdown = make(chan int, channelBufferSize)

	// goroutine updates availability every p.tickTime. If a shutdown signal is received (or channel is closed),
	// the goroutine returns.
	go func() {
		var err error
		ticker := time.NewTicker(mc.tickTime)
		for {
			select {
			case <-ticker.C:
				// timer event causes MySQL checks
				if err = mc.RepairMySQLPodStuckDeleting(); err != nil {
					mc.log.ErrorfThrottled("Failed to repair MySQL pods stuck terminating: %v", err)
				}
			case <-mc.shutdown:
				// shutdown event causes termination
				ticker.Stop()
				return
			}
		}
	}()
}

// Pause pauses the MySQLChecker if it was running.
// It is safe to call Pause multiple times
func (mc *MySQLChecker) Pause() {
	if mc.shutdown != nil {
		close(mc.shutdown)
		mc.shutdown = nil
	}
}
