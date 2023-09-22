// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/handlerspi"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

// StatusData contains the data for the component status field
type StatusData struct {
	Vznsn       types.NamespacedName
	CondType    vzapi.ConditionType
	CompName    string
	CompVersion string
	Msg         string
	Ready       bool
}

// UpdateVerrazzanoComponentStatusToDisabled updates the component status to disabled
func UpdateVerrazzanoComponentStatusToDisabled(ctx handlerspi.HandlerContext, Vznsn types.NamespacedName, compName string) result.Result {
	// Always get the latest module from the controller-runtime cache to try and avoid conflict error
	vzcr := &vzapi.Verrazzano{}
	if err := ctx.Client.Get(context.TODO(), Vznsn, vzcr); err != nil {
		ctx.Log.Progress("Failed getting Verrazzano CR, retrying...")
		return result.NewResultShortRequeueDelay()
	}
	if vzcr.Status.Components == nil {
		vzcr.Status.Components = vzapi.ComponentStatusMap{}
	}
	compStatus := &vzapi.ComponentStatusDetails{
		Name:  compName,
		State: vzapi.CompStateDisabled,
	}
	vzcr.Status.Components[compName] = compStatus

	if err := ctx.Client.Status().Update(context.TODO(), vzcr); err != nil {
		if !errors.IsConflict(err) {
			ctx.Log.Progress("Failed to update Verrazzano component status, retrying: %v", err)
		}
		return result.NewResultShortRequeueDelay()
	}
	return result.NewResult()
}

// UpdateVerrazzanoComponentStatus updates the Verrazzano component status
func UpdateVerrazzanoComponentStatus(ctx handlerspi.HandlerContext, sd StatusData) result.Result {
	// Always get the latest module from the controller-runtime cache to try and avoid conflict error
	vzcr := &vzapi.Verrazzano{}
	if err := ctx.Client.Get(context.TODO(), sd.Vznsn, vzcr); err != nil {
		ctx.Log.Progress("Failed getting Verrazzano CR, retrying...")
		return result.NewResultShortRequeueDelay()
	}
	if vzcr.Status.Components == nil {
		vzcr.Status.Components = vzapi.ComponentStatusMap{}
	}
	compStatus := vzcr.Status.Components[sd.CompName]
	if compStatus == nil {
		compStatus = &vzapi.ComponentStatusDetails{
			Available:                getAvailPtr(vzapi.ComponentUnavailable),
			LastReconciledGeneration: 0,
			Name:                     sd.CompName,
			State:                    vzapi.CompStateUninstalled,
		}
		vzcr.Status.Components[sd.CompName] = compStatus
	}
	if sd.Ready {
		compStatus.Available = getAvailPtr(vzapi.ComponentAvailable)
		compStatus.State = vzapi.CompStateReady
		compStatus.LastReconciledGeneration = vzcr.Generation
		compStatus.Version = sd.CompVersion
	} else {
		compStatus.State = vzapi.CompStateReconciling
	}

	// Append or replace the condition
	cond := vzapi.Condition{
		Type:    sd.CondType,
		Status:  corev1.ConditionTrue,
		Message: sd.Msg,
	}
	addOrReplaceCondition(compStatus, cond)

	if err := ctx.Client.Status().Update(context.TODO(), vzcr); err != nil {
		if !errors.IsConflict(err) {
			ctx.Log.Progress("Failed to update Verrazzano component status, retrying: %v", err)
		}
		return result.NewResultShortRequeueDelay()
	}
	return result.NewResult()
}

// addOrReplaceCondition appends the condition to the list of conditions.
// if the condition already exists, then remove it
func addOrReplaceCondition(compStatus *vzapi.ComponentStatusDetails, cond vzapi.Condition) {
	cond.LastTransitionTime = getCurrentTime()

	// Copy conditions that have a different type than the input condition into a new list
	var newConditions []vzapi.Condition
	if compStatus.Conditions != nil {
		for i, existing := range compStatus.Conditions {
			if existing.Type != cond.Type {
				newConditions = append(newConditions, compStatus.Conditions[i])
			}
		}
	}

	// Always put the new condition at the end of the list since the kubectl status display and
	// some upgrade stuff depends on the most recent condition being the last one
	compStatus.Conditions = append(newConditions, cond)
}

func getCurrentTime() string {
	t := time.Now().UTC()
	return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func getAvailPtr(avail vzapi.ComponentAvailability) *vzapi.ComponentAvailability {
	return &avail
}
