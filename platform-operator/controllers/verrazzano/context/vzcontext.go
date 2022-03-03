// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package context

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// VerrazzanoContext the context needed to reconcile a Verrazzano CR
type VerrazzanoContext struct {
	// Log is the logger for the execution context
	Log vzlog.VerrazzanoLogger
	// Client is a Kubernetes client
	Client clipkg.Client
	// DryRun will do a dry run of operations if true
	DryRun bool
	// ActualCR is the CR passed to top level Reconcile.  It represents the desired Verrazzano state in the cluster
	ActualCR *vzapi.Verrazzano
}

// NewVerrazzanoContext creates a VerrazzanoContext
func NewVerrazzanoContext(log vzlog.VerrazzanoLogger, c clipkg.Client, actualCR *vzapi.Verrazzano, dryRun bool) (VerrazzanoContext, error) {
	return VerrazzanoContext{
		Log:      log,
		Client:   c,
		DryRun:   dryRun,
		ActualCR: actualCR,
	}, nil
}
