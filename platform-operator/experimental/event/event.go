// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package event

import (
	"context"
	"fmt"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EventType is the type of event
type EventType string

// EventType constants
const (
	// ModuleLifeCycleEvent describes a lifecycle event for a module
	ModuleLifeCycleEvent EventType = "module-lifecycle"

	// ModuleIntegrateAllRequestEvent is an event request to integrate all the modules
	ModuleIntegrateAllRequestEvent EventType = "module-integrate-all"
)

// DataKey is a configmap data key
type DataKey string

// DataKey constants
const (
	EventTypeKey         DataKey = "eventType"
	ActionKey            DataKey = "action"
	ResourceNamespaceKey DataKey = "resourceNamespace"
	ResourceNameKey      DataKey = "resourceName"
	TargetNamespaceKey   DataKey = "targetNamespace"
	ModuleNameKey        DataKey = "moduleName"
	ModuleVersionKey     DataKey = "moduleVersion"
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
	ResourceNSN     types.NamespacedName
	ModuleName      string
	ModuleVersion   string
	TargetNamespace string
}

// CreateModuleEvent creates a ModuleIntegrationEvent event for a module
func CreateModuleEvent(cli client.Client, module *moduleapi.Module, action Action, eventType EventType) result.Result {
	return CreateEvent(cli, &ModuleIntegrationEvent{
		EventType:       eventType,
		Action:          action,
		ResourceNSN:     types.NamespacedName{Namespace: module.Namespace, Name: module.Name},
		ModuleName:      module.Spec.ModuleName,
		ModuleVersion:   module.Spec.Version,
		TargetNamespace: module.Spec.TargetNamespace,
	})
}

// CreateEvent creates a lifecycle event
func CreateEvent(cli client.Client, ev *ModuleIntegrationEvent) result.Result {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getEventResourceName(ev.ResourceNSN.Name, string(ev.Action)),
			Namespace: ev.ResourceNSN.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.TODO(), cli, cm, func() error {
		if cm.Labels == nil {
			cm.Labels = make(map[string]string)
		}
		// Always replace existing event data for this module-action
		cm.Labels[constants.VerrazzanoModuleEventLabel] = string(ev.EventType)
		cm.Data = make(map[string]string)
		cm.Data[string(EventTypeKey)] = string(ev.EventType)
		cm.Data[string(ActionKey)] = string(ev.Action)
		cm.Data[string(ModuleNameKey)] = ev.ModuleName
		cm.Data[string(ModuleVersionKey)] = ev.ModuleVersion
		cm.Data[string(ResourceNamespaceKey)] = ev.ResourceNSN.Namespace
		cm.Data[string(ResourceNameKey)] = ev.ResourceNSN.Name
		cm.Data[string(TargetNamespaceKey)] = ev.TargetNamespace
		return nil
	})
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	return result.NewResult()
}

// ConfigMapToModuleIntegrationEvent converts an event configmap to a ModuleIntegrationEvent
func ConfigMapToModuleIntegrationEvent(cm *corev1.ConfigMap) *ModuleIntegrationEvent {
	ev := ModuleIntegrationEvent{}
	if cm.Data == nil {
		return &ev
	}
	s, _ := cm.Data[string(EventTypeKey)]
	ev.EventType = EventType(s)
	s, _ = cm.Data[string(ActionKey)]
	ev.Action = Action(s)
	ev.ModuleName, _ = cm.Data[string(ModuleNameKey)]
	ev.ModuleVersion, _ = cm.Data[string(ModuleVersionKey)]
	ev.ResourceNSN.Name, _ = cm.Data[string(ResourceNameKey)]
	ev.ResourceNSN.Namespace, _ = cm.Data[string(ResourceNamespaceKey)]
	ev.TargetNamespace, _ = cm.Data[string(TargetNamespaceKey)]
	return &ev
}

func getEventResourceName(name string, action string) string {
	return fmt.Sprintf("event-%s-%s", name, action)
}
