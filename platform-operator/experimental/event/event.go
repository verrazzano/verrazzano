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

type PayloadKey string

const (
	ActionKey            PayloadKey = "action"
	ResourceNamespaceKey PayloadKey = "resourceNamespace"
	ResourceNameKey      PayloadKey = "resourceName"
	TargetNamespaceKey   PayloadKey = "targetNamespace"
)

type Action string

const (
	Installed Action = "installed"
	Upgraded  Action = "upgraded"
	Deleted   Action = "deleted"
)

type LifecycleEvent struct {
	Action
	ResourceNSN     types.NamespacedName
	ModuleName      string
	ModuleVersion   string
	TargetNamespace string
}

type EventCustomResource struct {
	corev1.Event
}

// CreateModuleEvent creates a lifecycle event for a module
func CreateModuleEvent(cli client.Client, module *moduleapi.Module, action Action) result.Result {
	return CreateEvent(cli, LifecycleEvent{
		Action:          action,
		ResourceNSN:     types.NamespacedName{Namespace: module.Namespace, Name: module.Name},
		ModuleName:      module.Spec.ModuleName,
		ModuleVersion:   module.Spec.Version,
		TargetNamespace: module.Spec.TargetNamespace,
	})
}

// CreateEvent creates a lifecycle event
func CreateEvent(cli client.Client, ev LifecycleEvent) result.Result {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(ev.ResourceNSN.Name, string(ev.Action)),
			Namespace: ev.ResourceNSN.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.TODO(), cli, cm, func() error {
		if cm.Labels == nil {
			cm.Labels = make(map[string]string)
		}
		// Always replace existing event data for this module-action
		cm.Labels[constants.VerrazzanoModuleEventLabel] = ev.ModuleName
		cm.Data = make(map[string]string)
		cm.Data[string(ActionKey)] = string(ev.Action)
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

func ConfigMapToEvent(cm *corev1.ConfigMap) *LifecycleEvent {
	ev := LifecycleEvent{}
	if cm.Data == nil {
		return &ev
	}
	s, _ := cm.Data[string(ActionKey)]
	ev.Action = Action(s)
	ev.ResourceNSN.Name, _ = cm.Data[string(ResourceNameKey)]
	ev.ResourceNSN.Namespace, _ = cm.Data[string(ResourceNamespaceKey)]
	ev.TargetNamespace, _ = cm.Data[string(TargetNamespaceKey)]
	return &ev
}

func getResourceName(name string, action string) string {
	return fmt.Sprintf("%s-%s", name, action)
}
