// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package reconciler

import (
	"fmt"
	modulesv1beta2 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta2"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/controllers/platformctrl/modlifecycle/delegates"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var componentShimDelegates = map[string]func(*modulesv1beta2.ModuleLifecycle, client.StatusWriter) delegates.DelegateLifecycleReconciler{
	// NOTE: plugin point for component shim reconcilers; these will likely need their own adapter/wrapper
	//keycloak.ComponentName:  keycloak.NewComponent,
	//coherence.ComponentName: coherence.NewComponent,
	//weblogic.ComponentName:  weblogic.NewComponent,
}

func New(mlc *modulesv1beta2.ModuleLifecycle, sw client.StatusWriter) (delegates.DelegateLifecycleReconciler, error) {
	if shimDelegate := newShimDelegate(mlc, sw); shimDelegate != nil {
		return shimDelegate, nil
	}
	if mlc.Spec.Installer.HelmRelease != nil {
		// If an existing delegate does not exist, wrap it in a Helm adapter to just do helm stuff
		return newHelmAdapter(mlc, sw), nil
	}
	if mlc.Spec.Installer.Istio != nil {
		return nil, fmt.Errorf("no installer implemented for Istio installer")
	}
	return nil, fmt.Errorf("no installer specified for lifecycle instance %s/%s", mlc.Namespace, mlc.Name)
}

func newShimDelegate(mlc *modulesv1beta2.ModuleLifecycle, sw client.StatusWriter) delegates.DelegateLifecycleReconciler {
	labels := mlc.ObjectMeta.Labels
	if labels == nil {
		return nil
	}
	shimDelegate := componentShimDelegates[labels[delegates.ControllerLabel]]
	if shimDelegate != nil {
		return shimDelegate(mlc, sw)
	}
	return nil
}
