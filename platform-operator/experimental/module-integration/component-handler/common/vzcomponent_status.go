// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/handlerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

// UpdateComponentStatus updates the component status
func UpdateComponentStatus(ctx handlerspi.HandlerContext, vznsn types.NamespacedName, compName string, condType vzapi.ConditionType, msg string, ready bool) result.Result {
	// Always get the latest module from the controller-runtime cache to try and avoid conflict error
	vzcr := &vzapi.Verrazzano{}
	if err := ctx.Client.Get(context.TODO(), vznsn, vzcr); err != nil {
		ctx.Log.Progress("Failed getting Verrazzano CR, retrying...")
		return result.NewResultShortRequeueDelay()
	}
	if vzcr.Status.Components == nil {
		vzcr.Status.Components = vzapi.ComponentStatusMap{}
	}
	compStatus, _ := vzcr.Status.Components[compName]
	if compStatus == nil {
		compStatus = &vzapi.ComponentStatusDetails{
			Available:                getAvailPtr(vzapi.ComponentUnavailable),
			LastReconciledGeneration: 0,
			Name:                     compName,
			State:                    vzapi.CompStateUninstalled,
		}
		vzcr.Status.Components[compName] = compStatus
	}
	if ready {
		compStatus.Available = getAvailPtr(vzapi.ComponentAvailable)
		compStatus.State = vzapi.CompStateReady
		compStatus.LastReconciledGeneration = vzcr.Generation
	}

	// Append or replace the condition
	cond := vzapi.Condition{
		Type:    condType,
		Status:  corev1.ConditionTrue,
		Message: msg,
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
	cond.LastTransitionTime = getTransitionTime()

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

func getTransitionTime() string {
	t := time.Now().UTC()
	return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func getAvailPtr(avail vzapi.ComponentAvailability) *vzapi.ComponentAvailability {
	return &avail
}
