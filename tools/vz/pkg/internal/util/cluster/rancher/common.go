// Copyright (c) 2023, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	corev1 "k8s.io/api/core/v1"
)

type cattleStatus struct {
	Conditions []cattleCondition `json:"conditions,omitempty"`
}
type cattleCondition struct {
	Status  corev1.ConditionStatus `json:"status"`
	Type    string                 `json:"type"`
	Reason  string                 `json:"reason,omitempty"`
	Message string                 `json:"message,omitempty"`
}
