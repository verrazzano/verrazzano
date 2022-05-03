// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *VerrazzanoSecretsReconciler) reconcileHelmOverrideSource(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO List (cont):
	// 3. Update the Verrazzano CR to start a helm upgrade command
	//      a) Update the status.ReconcileGeneration for the prometheus operator
	//	    b) as an example: vz.Status.Components["prometheus-operator"].LastReconciledGeneration = 0 (it should be component generic)
	// 4. Create unit tests for new functions

	return ctrl.Result{}, nil
}
