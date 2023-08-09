// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integration

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/base/controllerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/event"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Reconcile reconciles the Verrazzano CR
func (r Reconciler) Reconcile(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	cm := &corev1.ConfigMap{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, cm); err != nil {
		spictx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error, don't requeue
		return result.NewResult()
	}
	ev := r.loadEvent(cm)

	log := vzlog.DefaultLogger()

	return r.applyIntegrationCharts(log, ev)
}

// applyIntegrationCharts applies all the integration charts for components that are enabled
func (r Reconciler) applyIntegrationCharts(log vzlog.VerrazzanoLogger, ev event.LifecycleEvent) result.Result {

	return result.NewResult()
}

// deleteIntegrationRelease deletes the integration release
func (r Reconciler) deleteIntegrationRelease(log vzlog.VerrazzanoLogger, ev event.LifecycleEvent) result.Result {

	return result.NewResult()
}

func (r Reconciler) loadEvent(cm corev1.ConfigMap) event.LifecycleEvent {
	ev := event.LifecycleEvent{}
	if cm.Data == nil {
		return ev
	}
	ev.ResourceNSN.Name, _ = cm.Data[string(event.ResourceNameKey)]
	ev.ResourceNSN.Namespace, _ = cm.Data[string(event.ResourceNamespaceKey)]
	s, _ := cm.Data[string(event.ActionKey)]
	ev.Action = event.Action(s)
	return ev
}
