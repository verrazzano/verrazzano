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
func (newResource *Verrazzano) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(newResource).
		Complete()
}

var _ webhook.Validator = &Verrazzano{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (newResource *Verrazzano) ValidateCreate() error {
	log := zap.S().With("source", "webhook", "resource", fmt.Sprintf("%s:%s", newResource.Namespace, newResource.Name))
	log.Info("Validate create")

	if err := ValidateVersion(newResource.Spec.Version); err != nil {
		return err
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (newResource *Verrazzano) ValidateUpdate(old runtime.Object) error {
	log := zap.S().With("source", "webhook", "resource", fmt.Sprintf("%s:%s", newResource.Namespace, newResource.Name))
	log.Info("Validate update")

	oldResource := old.(*Verrazzano)
	log.Infof("oldResource Annotations: %v, Finalizers: %v, Spec: %v", oldResource.Annotations, oldResource.Finalizers, oldResource.Spec)
	log.Infof("to Annotations: %v, Finalizers: %v, Spec: %v", newResource.Annotations, newResource.Finalizers, newResource.Spec)

	// The profile field is immutable
	if oldResource.Spec.Profile != newResource.Spec.Profile {
		return fmt.Errorf("Profile change is not allowed oldResource %s to %s", oldResource.Spec.Profile, newResource.Spec.Profile)
	}

	// Check to see if the update is an upgrade request, and if it is valid and allowable
	err := ValidateUpgradeRequest(&oldResource.Spec, &newResource.Spec)
	if err != nil {
		log.Error("Invalid upgrade request: %s", err.Error())
		return err
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (newResource *Verrazzano) ValidateDelete() error {
	log := zap.S().With("source", "webhook", "resource", fmt.Sprintf("%s:%s", newResource.Namespace, newResource.Name))
	log.Info("Validate delete")

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
