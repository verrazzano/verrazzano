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

type Updater interface {
	Update(vz *vzapi.Verrazzano)
}

type VerrazzanoStatusUpdater struct {
	client        clipkg.Client
	updateChannel chan *vzapi.Verrazzano
	channelLock   *sync.Mutex
	logger        *zap.SugaredLogger
}

type FakeVerrazzanoStatusUpdater struct {
	Client clipkg.Client
}

func (f *FakeVerrazzanoStatusUpdater) Update(vz *vzapi.Verrazzano) {
	f.Client.Status().Update(context.TODO(), vz)
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

func (v *VerrazzanoStatusUpdater) Update(vz *vzapi.Verrazzano) {
	v.updateChannel <- vz
}

func (v *VerrazzanoStatusUpdater) Start() {
	go func() {
		v.channelLock.Lock()
		defer v.channelLock.Unlock()
		if v.updateChannel != nil {
			return
		}
		v.updateChannel = make(chan *vzapi.Verrazzano, channelBufferSize)
		go func() {
			for {
				vz := <-v.updateChannel
				if vz == nil {
					v.shutdown()
					return
				}
				v.doUpdate(vz)
			}
		}()
	}()
}

func (v *VerrazzanoStatusUpdater) doUpdate(vz *vzapi.Verrazzano) {
	err := v.client.Status().Update(context.TODO(), vz)
	if err != nil && !apierrors.IsConflict(err) {
		v.logger.Errorf("Error updating Verrazzano status: %v", err)
	}
}

func (v *VerrazzanoStatusUpdater) shutdown() {
	v.channelLock.Lock()
	defer v.channelLock.Unlock()
	if v.updateChannel != nil {
		close(v.updateChannel)
		v.updateChannel = nil
	}
}
