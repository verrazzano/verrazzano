// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package watch

import (
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/base/controllerspi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// GetModuleWatches get WatchDescriptors for the set of module
func GetModuleWatches(moduleNames []string) []controllerspi.WatchDescriptor {
	var watches = []controllerspi.WatchDescriptor{}

	for i, moduleName := range moduleNames {
		watches = append(watches, controllerspi.WatchDescriptor{
			WatchedResourceKind: source.Kind{Type: &moduleapi.Module{}},
			FuncShouldReconcile: shouldReconcile,
		})
	}
	return watches
}

// shouldReconcile returns true if reconcile should be done in response to a Module status change to ready
func shouldReconcile(moduleNSN types.NamespacedName, newModule *moduleapi.Module, oldModule *moduleapi.Module, event controllerspi.WatchEventType) bool {
	// Get new Ready condition and return false if not ready
	newCond := status.GetReadyCondition(newModule)
	if newCond == nil {
		return false
	}
	if newCond.Status != corev1.ConditionTrue {
		return false
	}

	// The new module is ready get old Ready condition
	oldCond := status.GetReadyCondition(oldModule)
	if oldCond == nil {
		return false
	}

	// Return false if the old condition reason was different OR if old condition was NOT ready
	return oldCond.Reason != newCond.Reason || oldCond.Status != corev1.ConditionTrue
}
