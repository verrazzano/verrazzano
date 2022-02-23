// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package context

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// implicit base profile (defaults)
	baseProfile = "base"
)

// VerrazzanoContext the context needed to reconcile a Verrazzano CR
type VerrazzanoContext struct {
	// log logger for the execution context
	Log vzlog.VerrazzanoLogger
	// client Kubernetes client
	Client clipkg.Client
	// dryRun If true, do a dry run of operations
	DryRun bool
	// ActualCR is the CR passed to top level Reconcile.  It epresents the desired Verrazzano state in the cluster
	ActualCR *vzapi.Verrazzano
}

// NewVerrazzanoContext creates a VerrazzanoContext, while creating an effective CR
func NewVerrazzanoContext(log vzlog.VerrazzanoLogger, c clipkg.Client, actualCR *vzapi.Verrazzano, dryRun bool) (VerrazzanoContext, error) {
	return VerrazzanoContext{
		Log:      log,
		Client:   c,
		DryRun:   dryRun,
		ActualCR: actualCR,
	}, nil
}
