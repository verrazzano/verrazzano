// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// SetupWebhookWithManager is used to let the controller manager know about the webhook
func (r *Verrazzano) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

var _ webhook.Validator = &Verrazzano{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Verrazzano) ValidateCreate() error {
	zap.S().Info("validate create")

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Verrazzano) ValidateUpdate(old runtime.Object) error {
	zap.S().Info("validate update")

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Verrazzano) ValidateDelete() error {
	zap.S().Info("validate delete")

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
