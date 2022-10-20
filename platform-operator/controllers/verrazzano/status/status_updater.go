// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/log"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

const maxUpdateAttempts = 5

type Updater interface {
	Update(event *UpdateEvent)
}

type UpdateEventType int8

type UpdateEvent struct {
	Verrazzano   *vzapi.Verrazzano
	Version      *string
	State        vzapi.VzStateType
	Conditions   []vzapi.Condition
	Availability *AvailabilityStatus
	InstanceInfo *vzapi.InstanceInfo
	Components   map[string]*vzapi.ComponentStatusDetails
}

type VerrazzanoStatusUpdater struct {
	client        clipkg.Client
	updateChannel chan *UpdateEvent
	channelLock   *sync.Mutex
	logger        *zap.SugaredLogger
}

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

func (v *VerrazzanoStatusUpdater) Update(event *UpdateEvent) {
	v.updateChannel <- event
}

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
				if err := v.doUpdate(event, 0); err != nil {
					v.logger.Errorf("%v", err)
				}
			}
		}()
	}()
}

func (v *VerrazzanoStatusUpdater) doUpdate(event *UpdateEvent, attempt int) error {
	vz, err := getVerrazzanoResource(v.client)
	if err != nil {
		return err
	}
	event.merge(vz)
	err = v.client.Status().Update(context.TODO(), vz)
	if apierrors.IsConflict(err) && attempt < maxUpdateAttempts {
		return v.doUpdate(event, attempt+1)
	}
	return err
}

func (v *VerrazzanoStatusUpdater) shutdown() {
	v.channelLock.Lock()
	defer v.channelLock.Unlock()
	if v.updateChannel != nil {
		close(v.updateChannel)
		v.updateChannel = nil
	}
}

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
