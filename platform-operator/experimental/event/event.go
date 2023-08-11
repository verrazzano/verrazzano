// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package event

import (
	"context"
	"encoding/base64"
	"fmt"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

// EventType is the type of event
type EventType string

// EventType constants
const (
	// IntegrateSingleRequestEvent is a request to integrate a single module
	IntegrateSingleRequestEvent EventType = "integrate-single"

	// IntegrateCascadeRequestEvent is an event request to integrate the other modules except the one
	// in the event payload (since it will already have been integrated)
	IntegrateCascadeRequestEvent EventType = "integrate-cascade"
)

// DataKey is a configmap data key
type DataKey string

// DataKey constants
const (
	EventDataKey DataKey = "event"
)

// Action is the lifecycle action
type Action string

// Action constants
const (
	Installed Action = "installed"
	Upgraded  Action = "upgraded"
	Deleted   Action = "deleted"
)

// ModuleIntegrationEvent contains the event data
type ModuleIntegrationEvent struct {
	EventType
	Action

	// Cascade true indicates that the single-module integration controller should potentially create an
	// event to integrate other modules
	Cascade         bool
	ResourceNSN     types.NamespacedName
	ModuleName      string
	ModuleVersion   string
	TargetNamespace string
}

// NewModuleIntegrationEvent creates a ModuleIntegrationEvent event struct for a module
func NewModuleIntegrationEvent(module *moduleapi.Module, action Action, eventType EventType, cascade bool) *ModuleIntegrationEvent {
	return &ModuleIntegrationEvent{
		Cascade:         cascade,
		EventType:       eventType,
		Action:          action,
		ResourceNSN:     types.NamespacedName{Namespace: module.Namespace, Name: module.Name},
		ModuleName:      module.Spec.ModuleName,
		ModuleVersion:   module.Spec.Version,
		TargetNamespace: module.Spec.TargetNamespace,
	}
}

// CreateModuleIntegrationEvent creates a ModuleIntegrationEvent event for a module
func CreateModuleIntegrationEvent(cli client.Client, module *moduleapi.Module, action Action) result.Result {
	return createEvent(cli, NewModuleIntegrationEvent(module, action, IntegrateSingleRequestEvent, true))
}

// CreateModuleIntegrationCascadeEvent creates a ModuleIntegrationEvent event for a module to integrate other modules
func CreateModuleIntegrationCascadeEvent(cli client.Client, sourceEvent *ModuleIntegrationEvent) result.Result {
	// Use the fields from the input event to create a new event of a different type
	ev := *sourceEvent
	ev.EventType = IntegrateCascadeRequestEvent
	ev.Cascade = false
	return createEvent(cli, &ev)
}

// CreateNonCascadingModuleIntegrationEvent creates a ModuleIntegrationEvent event for a module with no cascading
func CreateNonCascadingModuleIntegrationEvent(cli client.Client, module *moduleapi.Module, action Action) result.Result {
	return createEvent(cli, NewModuleIntegrationEvent(module, action, IntegrateSingleRequestEvent, false))
}

// createEvent creates a lifecycle event
func createEvent(cli client.Client, ev *ModuleIntegrationEvent) result.Result {
	y, err := yaml.Marshal(ev)
	if err != nil {
		result.NewResultShortRequeueDelayWithError(err)
	}
	// convert to base64
	encoded := base64.StdEncoding.EncodeToString(y)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getEventResourceName(ev.ResourceNSN.Name, string(ev.Action), string(ev.EventType)),
			Namespace: ev.ResourceNSN.Namespace,
		},
	}
	_, err = controllerutil.CreateOrUpdate(context.TODO(), cli, cm, func() error {
		if cm.Labels == nil {
			cm.Labels = make(map[string]string)
		}
		// Always replace existing event data for this module-action
		cm.Labels[constants.VerrazzanoModuleEventLabel] = string(ev.EventType)
		cm.Data = make(map[string]string)
		cm.Data[string(EventDataKey)] = encoded
		return nil
	})
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	return result.NewResult()
}

// ConfigMapToModuleIntegrationEvent converts an event configmap to a ModuleIntegrationEvent
func ConfigMapToModuleIntegrationEvent(cm *corev1.ConfigMap) (*ModuleIntegrationEvent, error) {
	decoded, err := base64.StdEncoding.DecodeString(cm.Data[string(EventDataKey)])
	if err != nil {
		return nil, err
	}
	ev := ModuleIntegrationEvent{}
	if err := yaml.Unmarshal(decoded, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

func getEventResourceName(name string, action string, eventType string) string {
	return fmt.Sprintf("event-%s-%s-%s", name, action, eventType)
}
