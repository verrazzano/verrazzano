// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// OCNEOCIQuickCreate should be both a validating and defaulting webhook
var _ webhook.Validator = &OKEQuickCreate{}

// SetupWebhookWithManager is used to let the controller manager know about the webhook
func (o *OKEQuickCreate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(o).
		Complete()
}

func (o *OKEQuickCreate) ValidateCreate() error {
	return nil
}

func (o *OKEQuickCreate) ValidateUpdate(old runtime.Object) error {
	oldCluster, ok := old.(*OKEQuickCreate)
	if !ok {
		return errors.New("update resource must be of kind OKEQuickCreate")
	}
	if err := o.updateAllowed(oldCluster); err != nil {
		return err
	}
	return nil
}

func (o *OKEQuickCreate) updateAllowed(other *OKEQuickCreate) error {
	return nil
}

func (o *OKEQuickCreate) ValidateDelete() error {
	return nil
}
