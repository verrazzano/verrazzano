// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconciler

import (
	"context"
	"fmt"
	modulesv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/modules/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"time"
)

//UpdateStatus configures the Module's status based on the passed in state and then updates the Module on the cluster
func (r *Reconciler) UpdateStatus(ctx spi.ComponentContext, condition modulesv1alpha1.ModuleCondition) error {
	module := ctx.Module()
	phase := modulesv1alpha1.Phase(condition)
	// Update the Module's Phase
	module.SetPhase(phase)
	// Append a new condition, if applicable
	appendCondition(module, string(phase), condition)
	return r.doStatusUpdate(ctx)
}

func NeedsReconcile(ctx spi.ComponentContext) bool {
	return ctx.Module().Status.ObservedGeneration != ctx.Module().Generation
}

func NewCondition(message string, condition modulesv1alpha1.ModuleCondition) modulesv1alpha1.Condition {
	t := time.Now().UTC()
	return modulesv1alpha1.Condition{
		Type:    condition,
		Message: message,
		Status:  corev1.ConditionTrue,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}
}

func (r *Reconciler) doStatusUpdate(ctx spi.ComponentContext) error {
	module := ctx.Module()
	err := r.StatusWriter.Update(context.TODO(), module)
	if err == nil {
		return err
	}
	if k8serrors.IsConflict(err) {
		ctx.Log().Debugf("Update conflict for Module %s: %v", module.Name, err)
	} else {
		ctx.Log().Errorf("Failed to update Module %s :v", module.Name, err)
	}
	// Return error so that reconcile gets called again
	return err
}

func appendCondition(module *modulesv1alpha1.Module, message string, condition modulesv1alpha1.ModuleCondition) {
	conditions := module.Status.Conditions
	lastCondition := conditions[len(conditions)-1]
	newCondition := NewCondition(message, condition)
	// Only update the conditions if there is a notable change between the last update
	if needsConditionUpdate(lastCondition, newCondition) {
		// Delete oldest condition if at tracking limit
		if len(conditions) > modulesv1alpha1.ConditionArrayLimit {
			conditions = conditions[1:]
		}
		module.Status.Conditions = append(conditions, newCondition)
	}
}

//needsConditionUpdate checks if the condition needs an update
func needsConditionUpdate(last, new modulesv1alpha1.Condition) bool {
	return last.Type != new.Type && last.Message != new.Message
}
