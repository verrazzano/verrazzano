// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"time"

	modulesplatform "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func AppendCondition(module *modulesplatform.Module, message string, conditionType modulesplatform.LifecycleCondition) {
	conditions := module.Status.Conditions
	newCondition := NewCondition(message, conditionType)
	var lastCondition modulesplatform.ModuleCondition
	if len(conditions) > 0 {
		lastCondition = conditions[len(conditions)-1]
	}

	// Only update the conditions if there is a notable change between the last update
	if needsConditionUpdate(lastCondition, newCondition) {
		// Delete the oldest condition if at tracking limit
		if len(conditions) > modulesplatform.ConditionArrayLimit {
			conditions = conditions[1:]
		}
		module.Status.Conditions = append(conditions, newCondition)
	}
}

func NewCondition(message string, conditionType modulesplatform.LifecycleCondition) modulesplatform.ModuleCondition {
	t := time.Now().UTC()
	return modulesplatform.ModuleCondition{
		Type:    conditionType,
		Message: message,
		Status:  corev1.ConditionTrue,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}
}

// needsConditionUpdate checks if the condition needs an update
func needsConditionUpdate(last, new modulesplatform.ModuleCondition) bool {
	return last.Type != new.Type && last.Message != new.Message
}
