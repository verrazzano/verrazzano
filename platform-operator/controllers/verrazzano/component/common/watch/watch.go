// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package watch

import (
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	vzapiv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetModuleInstalledWatches gets WatchDescriptors for the set of module where the code watches for the module transitioning to installed ready.
func GetModuleInstalledWatches(moduleNames []string) []controllerspi.WatchDescriptor {
	return GetModuleWatchesForReason(moduleNames, moduleapi.ReadyReasonInstallSucceeded)
}

// GetModuleUpdatedWatches gets WatchDescriptors for the set of module where the code watches for the module transitioning to update ready.
func GetModuleUpdatedWatches(moduleNames []string) []controllerspi.WatchDescriptor {
	return GetModuleWatchesForReason(moduleNames, moduleapi.ReadyReasonUpdateSucceeded)
}

// GetModuleUpgradedWatches gets WatchDescriptors for the set of module where the code watches for the module transitioning to upgrade ready.
func GetModuleUpgradedWatches(moduleNames []string) []controllerspi.WatchDescriptor {
	return GetModuleWatchesForReason(moduleNames, moduleapi.ReadyReasonUpgradeSucceeded)
}

// GetModuleWatchesForReason gets WatchDescriptors for the set a modules that transition to ready for a given reason .
func GetModuleWatchesForReason(moduleNames []string, reason moduleapi.ModuleConditionReason) []controllerspi.WatchDescriptor {
	var watches []controllerspi.WatchDescriptor
	moduleNameSet := vzstring.SliceToSet(moduleNames)

	// Just use a single watch that looks up the name in the set for a match
	watches = append(watches, controllerspi.WatchDescriptor{
		WatchedResourceKind: &moduleapi.Module{},
		FuncShouldReconcile: func(cli client.Client, wev controllerspi.WatchEvent) bool {
			// Return false if this is not one of the modules that is being watched
			var newModule = wev.NewWatchedObject.(*moduleapi.Module)
			_, ok := moduleNameSet[newModule.Spec.ModuleName]
			if !ok {
				return false
			}

			// This is a create or delete event don't trigger reconcile
			if wev.OldWatchedObject == nil {
				return false
			}

			// Get new module Ready condition and return false if not ready
			newCond := status.GetReadyCondition(newModule)
			if newCond == nil {
				return false
			}
			if newCond.Reason != reason {
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

			// Return true if the module transitioned to Ready from a different reason (installing to installed)
			return oldCond.Reason != newCond.Reason && oldCond.Status != corev1.ConditionTrue
		},
	})

	return watches
}

// GetVerrazzanoSpecWatch watches for any Verrazzano spec update.
func GetVerrazzanoSpecWatch() []controllerspi.WatchDescriptor {
	// Use a single watch that looks up the name in the set for a match
	var watches []controllerspi.WatchDescriptor
	watches = append(watches, controllerspi.WatchDescriptor{
		WatchedResourceKind: &vzapiv1beta1.Verrazzano{},
		FuncShouldReconcile: func(cli client.Client, wev controllerspi.WatchEvent) bool {
			if wev.WatchEventType != controllerspi.Updated {
				return false
			}
			return wev.OldWatchedObject.(*vzapiv1beta1.Verrazzano).Generation != wev.NewWatchedObject.(*vzapiv1beta1.Verrazzano).Generation
		},
	})
	return watches
}

// GetCreateSecretWatch watches for a secret creation with the specified name
func GetCreateSecretWatch(name, namespace string) []controllerspi.WatchDescriptor {
	// Use a single watch that looks up the name in the set for a match
	var watches []controllerspi.WatchDescriptor
	watches = append(watches, controllerspi.WatchDescriptor{
		WatchedResourceKind: &corev1.Secret{},
		FuncShouldReconcile: func(cli client.Client, wev controllerspi.WatchEvent) bool {
			if wev.WatchEventType != controllerspi.Created {
				return false
			}
			objectNS := wev.NewWatchedObject.GetNamespace()
			objectName := wev.NewWatchedObject.GetName()
			result := objectNS == namespace && objectName == name
			return result
		},
	})
	return watches
}

// GetUpdateSecretWatch watches for a secret update with the specified name
func GetUpdateSecretWatch(name, namespace string) []controllerspi.WatchDescriptor {
	// Use a single watch that looks up the name in the set for a match
	var watches []controllerspi.WatchDescriptor
	watches = append(watches, controllerspi.WatchDescriptor{
		WatchedResourceKind: &corev1.Secret{},
		FuncShouldReconcile: func(cli client.Client, wev controllerspi.WatchEvent) bool {
			if wev.WatchEventType != controllerspi.Updated {
				return false
			}
			objectNS := wev.NewWatchedObject.GetNamespace()
			objectName := wev.NewWatchedObject.GetName()
			result := objectNS == namespace && objectName == name
			return result
		},
	})
	return watches
}

// GetDeleteSecretWatch watches for a secret deletion with the specified name
func GetDeleteSecretWatch(name, namespace string) []controllerspi.WatchDescriptor {
	// Use a single watch that looks up the name in the set for a match
	var watches []controllerspi.WatchDescriptor
	watches = append(watches, controllerspi.WatchDescriptor{
		WatchedResourceKind: &corev1.Secret{},
		FuncShouldReconcile: func(cli client.Client, wev controllerspi.WatchEvent) bool {
			if wev.WatchEventType != controllerspi.Deleted {
				return false
			}
			return wev.NewWatchedObject.GetNamespace() == namespace && wev.NewWatchedObject.GetName() == name
		},
	})
	return watches
}

// GetCreateNamespaceWatch watches for a namespace creation with the specified name
func GetCreateNamespaceWatch(name string) []controllerspi.WatchDescriptor {
	// Use a single watch that looks up the name in the set for a match
	var watches []controllerspi.WatchDescriptor
	watches = append(watches, controllerspi.WatchDescriptor{
		WatchedResourceKind: &corev1.Namespace{},
		FuncShouldReconcile: func(cli client.Client, wev controllerspi.WatchEvent) bool {
			if wev.WatchEventType != controllerspi.Created {
				return false
			}
			return wev.NewWatchedObject.GetName() == name
		},
	})
	return watches
}

// GetUpdateNamespaceWatch watches for a namespace update with the specified name
func GetUpdateNamespaceWatch(name string) []controllerspi.WatchDescriptor {
	// Use a single watch that looks up the name in the set for a match
	var watches []controllerspi.WatchDescriptor
	watches = append(watches, controllerspi.WatchDescriptor{
		WatchedResourceKind: &corev1.Namespace{},
		FuncShouldReconcile: func(cli client.Client, wev controllerspi.WatchEvent) bool {
			if wev.WatchEventType != controllerspi.Updated {
				return false
			}
			return wev.NewWatchedObject.GetName() == name
		},
	})
	return watches
}

// CombineWatchDescriptors combines multiple arrays of WatchDescriptors into one array
func CombineWatchDescriptors(watchDescriptors ...[]controllerspi.WatchDescriptor) []controllerspi.WatchDescriptor {
	var allWatchDescriptors []controllerspi.WatchDescriptor
	for i := range watchDescriptors {
		allWatchDescriptors = append(allWatchDescriptors, watchDescriptors[i]...)
	}
	return allWatchDescriptors
}
