// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package event

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type PayloadKey string
const (
	ResourceNamespaceKey PayloadKey = "moduleNamespace"
	ResourceNameKey      PayloadKey = "moduleName"
	ActionKey            PayloadKey = "action"
)

type Action string
const (
	Installed Action = "installed"
	Upgraded  Action = "upgraded"
	Deleted   Action = "deleted"
)

type LifecycleEvent struct {
	ResourceNSN types.NamespacedName
	Action
}

type EventCustomResource struct {
	corev1.Event
}
