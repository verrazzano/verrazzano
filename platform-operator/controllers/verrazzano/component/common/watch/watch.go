// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package watch

import (
	"context"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/base/controllerspi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// GetModuleWatches get WatchDescriptors for the set of module
func GetModuleWatches(moduleNames []string) []controllerspi.WatchDescriptor {
	var watches = []controllerspi.WatchDescriptor{}

	for i, _ := range moduleNames {
		watches = append(watches, controllerspi.WatchDescriptor{
			WatchedResourceKind: source.Kind{Type: &moduleapi.Module{}},
			FuncShouldReconcile: func(cli client.Client, wev controllerspi.WatchEvent) bool {
				// Return false if the module target namespace doesn't match the desired module name
				modName := moduleNames[i]
				var module = moduleapi.Module{}
				err := cli.Get(context.TODO(), wev.ReconcilingResource, &module)
				if err != nil {
					return false
				}
				if modName != module.Spec.ModuleName {
					return false
				}

				// Get new module Ready condition and return false if not ready
				var newModule = wev.NewWatchedObject.(*moduleapi.Module)
				newCond := status.GetReadyCondition(newModule)
				if newCond == nil {
					return false
				}
				if newCond.Status != corev1.ConditionTrue {
					return false
				}

				// The new module is ready. get old module Ready condition
				var oldModule = wev.OldWatchedObject.(*moduleapi.Module)
				oldCond := status.GetReadyCondition(oldModule)
				if oldCond == nil {
					return false
				}

				// Return false if the old module condition reason matches the new module AND the old condition was ready.
				// In that case we don't need to reconcile
				return !(oldCond.Reason == newCond.Reason && oldCond.Status == corev1.ConditionTrue)
			},
		})
	}
	return watches
}

// ShouldReconcile returns true if reconcile should be done in response to the Module status changing to ready
func ShouldReconcile(_ types.NamespacedName, newWatchedObject client.Object, oldWatchedObject client.Object, event controllerspi.WatchEventType) bool {
	// Get new module Ready condition and return false if not ready
	var newModule = newWatchedObject.(*moduleapi.Module)
	newCond := status.GetReadyCondition(newModule)
	if newCond == nil {
		return false
	}
	if newCond.Status != corev1.ConditionTrue {
		return false
	}

	// The new module is ready. get old module Ready condition
	var oldModule = oldWatchedObject.(*moduleapi.Module)
	oldCond := status.GetReadyCondition(oldModule)
	if oldCond == nil {
		return false
	}

	// Return false if the old module condition reason matches the new module AND the old condition was ready.
	// In that case we don't need to reconcile
	return !(oldCond.Reason == newCond.Reason && oldCond.Status == corev1.ConditionTrue)
}
