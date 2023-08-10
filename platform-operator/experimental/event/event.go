// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package event

import (
	"context"
	"fmt"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type PayloadKey string

const (
	ResourceNamespaceKey PayloadKey = "moduleNamespace"
	ResourceNameKey      PayloadKey = "moduleName"
	ActionKey            PayloadKey = "action"
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
		Action:        action,
		ResourceNSN:   types.NamespacedName{Namespace: module.Namespace, Name: module.Name},
		ModuleName:    module.Spec.ModuleName,
		ModuleVersion: module.Spec.Version,
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
		// Always replace existing event data for this module-action
		cm.Labels[constants.VerrazzanoEventLabel] = ev.ModuleName
		cm.Data = make(map[string]string)
		cm.Data[string(ResourceNamespaceKey)] = ev.ResourceNSN.Namespace
		cm.Data[string(ResourceNameKey)] = ev.ResourceNSN.Name
		cm.Data[string(ActionKey)] = string(ev.Action)
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
	ev.ResourceNSN.Name, _ = cm.Data[string(ResourceNameKey)]
	ev.ResourceNSN.Namespace, _ = cm.Data[string(ResourceNamespaceKey)]
	s, _ := cm.Data[string(ActionKey)]
	ev.Action = Action(s)
	return &ev
}

func getResourceName(name string, action string) string {
	return fmt.Sprintf("%s-%s", name, action)
}
