// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package healthcheck

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/log"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

// Updater interface for Verrazzano status updates
type Updater interface {
	// Update given an UpdateEvent, updates a Verrazzano resource's status object
	Update(event *UpdateEvent)
}

// UpdateEvent defines an event used during Verrazzano update. Event fields are merged into the Verrazzano
// resource's status object.
type UpdateEvent struct {
	Verrazzano   *vzapi.Verrazzano // resource reference for test injection
	Version      *string
	State        vzapi.VzStateType
	Conditions   []vzapi.Condition
	Availability *AvailabilityStatus
	InstanceInfo *vzapi.InstanceInfo
	Components   map[string]*vzapi.ComponentStatusDetails
}

// VerrazzanoStatusUpdater implement Updater for asynchronous status updates, using updateChannel to receive UpdateEvent objects
// from other goroutines
type VerrazzanoStatusUpdater struct {
	client        clipkg.Client
	updateChannel chan *UpdateEvent // The channel on which UpdateEvent objects are sent/received
	channelLock   *sync.Mutex       // For restricting access to the updateChannel
	logger        *zap.SugaredLogger
}

// FakeVerrazzanoStatusUpdater for stubbing unit tests
type FakeVerrazzanoStatusUpdater struct {
	Client clipkg.Client
}

func (f *FakeVerrazzanoStatusUpdater) Update(event *UpdateEvent) {
	event.merge(event.Verrazzano)
	f.Client.Status().Update(context.TODO(), event.Verrazzano)
}

func NewStatusUpdater(client clipkg.Client) *VerrazzanoStatusUpdater {
	s := &VerrazzanoStatusUpdater{
		client:      client,
		channelLock: &sync.Mutex{},
		logger:      zap.S().With(log.FieldController, "statusUpdater"),
	}
	s.Start()
	return s
}

// Update initiates an asynchronous Verrazzano status update
func (v *VerrazzanoStatusUpdater) Update(event *UpdateEvent) {
	v.updateChannel <- event
}

// Start initiates a goroutine that listens of the status update channel for events
func (v *VerrazzanoStatusUpdater) Start() {
	go func() {
		v.channelLock.Lock()
		defer v.channelLock.Unlock()
		if v.updateChannel != nil {
			return
		}
		v.updateChannel = make(chan *UpdateEvent, channelBufferSize)
		go func() {
			for {
				event := <-v.updateChannel
				if event == nil {
					v.shutdown()
					return
				}
				if err := v.doUpdate(event); err != nil {
					v.logger.Errorf("Error updating component status: %v", err)
				}
			}
		}()
	}()
}

// doUpdate updates the cluster Verrazzano. Resource conflicts are retried.
func (v *VerrazzanoStatusUpdater) doUpdate(event *UpdateEvent) error {
	// Status updates are retried indefinitely in case of conflicts
	for {
		vz, err := getVerrazzanoResource(v.client)
		if err != nil {
			return err
		}
		event.merge(vz)
		err = v.client.Status().Update(context.TODO(), vz)
		if err == nil || !apierrors.IsConflict(err) {
			// Update didn't have a conflict, so return and exit update loop
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// shutdown closes the status channel
func (v *VerrazzanoStatusUpdater) shutdown() {
	v.channelLock.Lock()
	defer v.channelLock.Unlock()
	if v.updateChannel != nil {
		close(v.updateChannel)
		v.updateChannel = nil
	}
}

// merge merges the UpdateEvent into the Verrazzano resource's status object
func (u *UpdateEvent) merge(vz *vzapi.Verrazzano) {
	if vz == nil {
		return
	}
	// Add Version
	if u.Version != nil {
		vz.Status.Version = *u.Version
	}
	// Add Verrazzano State
	if len(u.State) > 0 {
		vz.Status.State = u.State
	}
	// Add Verrazzano Conditions
	if u.Conditions != nil {
		vz.Status.Conditions = u.Conditions
	}
	// Add component status details
	for component, details := range u.Components {
		if vz.Status.Components == nil {
			vz.Status.Components = map[string]*vzapi.ComponentStatusDetails{}
		}
		vz.Status.Components[component] = details
	}
	// Add instance info
	if u.InstanceInfo != nil {
		vz.Status.VerrazzanoInstance = u.InstanceInfo
	}
	// Add availability
	if u.Availability != nil {
		u.Availability.merge(vz)
	}
}
