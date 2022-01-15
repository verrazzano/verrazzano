// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"strings"

	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/vzinstance"
	ctrl "sigs.k8s.io/controller-runtime"
)

// reconcileComponents reconciles each component using the following rules:
// 1. Always requeue until all enabled components have completed installation
// 2. Don't update the component state until all the work in that state is done, since
//    that update will cause a state transition
// 3. Loop through all components before returning, except for the case
//    where update status fails, in which case we exit the function and requeue
//    immediately.
func (r *Reconciler) reconcileComponents(_ context.Context, spiCtx spi.ComponentContext) (ctrl.Result, error) {
	cr := spiCtx.ActualCR()
	spiCtx.Log().Progress("Reconciling components for Verrazzano installation")

	var requeue bool
	// Loop through all of the Verrazzano components and upgrade each one sequentially for now; will parallelize later
	for _, comp := range spiCtx.GetComponentRegistry().GetComponents() {
		compName := comp.Name()
		compContext := spiCtx.Init(compName).Operation(vzconst.InstallOperation)
		compLog := compContext.Log()
		compLog.Oncef("Component %s is being reconciled", compName)
		err := comp.Reconcile(compContext)
		if err != nil {
			handleError(compLog, err)
			requeue = true
		}
	}
	// Update the status with the instance URLs
	// TODO: See if we can conditionally get the instance info URLs
	cr.Status.VerrazzanoInstance = vzinstance.GetInstanceInfo(spiCtx)
	// Update the VZ status before returning
	err := r.Status().Update(context.TODO(), spiCtx.ActualCR())
	if err != nil {
		spiCtx.Log().Errorf("Failed to update Verrazzano resource status: %v", err)
		return newRequeueWithDelay(), err
	}
	if requeue {
		return newRequeueWithDelay(), nil
	}
	return ctrl.Result{}, nil
}

// handleError - detects if a an error is a RetryableError; if it is, logs it appropriately and
func handleError(log vzlog.VerrazzanoLogger, err error) {
	switch actualErr := err.(type) {
	case ctrlerrors.RetryableError:
		if actualErr.HasCause() {
			cause := actualErr.Cause
			if ctrlerrors.IsUpdateConflict(cause) ||
				strings.Contains(cause.Error(), "failed calling webhook") {
				log.Debugf("Failed during install: %v", cause)
				return
			}
			log.Errorf("Failed during install: %v", actualErr.Error())
		}
	default:
		log.Errorf("Unexpected error occurred during install/upgrade: %s", actualErr.Error())
	}
}
