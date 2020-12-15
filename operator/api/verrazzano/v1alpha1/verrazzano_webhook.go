// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"fmt"
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
	log := zap.S().With("source", "webhook", "resource", fmt.Sprintf("%s:%s", r.Namespace, r.Name))
	log.Info("Validate create")

	if err := ValidateVersion(r.Spec.Version); err != nil {
		return err
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Verrazzano) ValidateUpdate(old runtime.Object) error {
	log := zap.S().With("source", "webhook", "resource", fmt.Sprintf("%s:%s", r.Namespace, r.Name))
	log.Info("Validate update")

	from := old.(*Verrazzano)
	log.Infof("from Annotations: %v, Finalizers: %v, Spec: %v", from.Annotations, from.Finalizers, from.Spec)
	log.Infof("to Annotations: %v, Finalizers: %v, Spec: %v", r.Annotations, r.Finalizers, r.Spec)

	err := ValidateUpgradeRequest(&from.Spec, &r.Spec)
	if err != nil {
		log.Error("Invalid upgrade request: %s", err.Error())
		return err
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Verrazzano) ValidateDelete() error {
	log := zap.S().With("source", "webhook", "resource", fmt.Sprintf("%s:%s", r.Namespace, r.Name))
	log.Info("Validate delete")

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
