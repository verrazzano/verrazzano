// Copyright (c) 2023, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import corev1 "k8s.io/api/core/v1"

type capiStatus struct {
	Conditions []capiCondition `json:"conditions,omitempty"`
}
type capiCondition struct {
	Status corev1.ConditionStatus `json:"status"`
	Type   string                 `json:"type"`
	Reason string                 `json:"reason,omitempty"`
}
