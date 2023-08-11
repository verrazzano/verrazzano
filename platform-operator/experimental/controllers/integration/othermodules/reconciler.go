// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integrate_single

import (
	ctx "context"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/base/controllerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/event"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"path"
)

// Reconcile reconciles the Verrazzano CR
func (r Reconciler) Reconcile(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	log := vzlog.DefaultLogger()

	cm := &corev1.ConfigMap{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, cm); err != nil {
		spictx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error, don't requeue
		return result.NewResult()
	}
	ev := event.ConfigMapToModuleIntegrationEvent(cm)
	res := r.createIntegrationEvents(log, ev)
	if res.ShouldRequeue() {
		return res
	}

	// Delete the event.  This is safe to do since the integration controller
	// is the only controller processing IntegrateOthersRequestEvent events
	if err := r.Client.Delete(ctx.TODO(), cm); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

// createIntegrationEvents creates integration events for all modules that have an integration chart,
// except for the module that was just integrated (i.e. the module in the IntegrateOthersRequestEvent)
func (r Reconciler) createIntegrationEvents(log vzlog.VerrazzanoLogger, ev *event.ModuleIntegrationEvent) result.Result {
	modules := moduleapi.ModuleList{}
	err := r.Client.List(context.TODO(), &modules)
	if err != nil {
		log.ErrorfThrottled("Failed getting the list of modules in the cluster: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	var requeue *result.Result
	for _, module := range modules.Items {
		// If this module was just integrated then ignore it
		if module.Name == ev.ModuleName {
			continue
		}
		
		// Nothing to do if an integration chart doesn't exist for this module
		moduleChartDir := path.Join(config.GetIntegrationChartsDir(), ev.ModuleName)
		_, err := os.Stat(moduleChartDir)
		if err != nil {
			if os.IsNotExist(err) {
				return result.NewResult()
			}
			log.ErrorfThrottled("Failed to check if integration chart exists for module %s: %v", ev.ModuleName, err)
			return result.NewResultShortRequeueDelayWithError(err)
		}
		// Create an event requesting that this module be integrated
		res := event.CreateNonCascadingModuleIntegrationEvent(r.Client, &module, ev.Action)
		if res.ShouldRequeue() {
			requeue = &res
		}
	}

	if requeue != nil {
		return *requeue
	}
	return result.NewResult()
}
