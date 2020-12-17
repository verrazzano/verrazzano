// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"fmt"
	"github.com/verrazzano/verrazzano/operator/internal/util/env"
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
	log := zap.S().With("source", "webhook", "operation", "create", "resource", fmt.Sprintf("%s:%s", newResource.Namespace, newResource.Name))
	log.Info("Validate create")

	if env.IsValidationDisabled() {
		log.Info("Validation disabled, skipping")
		return nil
	}

	if err := ValidateVersion(newResource.Spec.Version); err != nil {
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (modifiedResource *Verrazzano) ValidateUpdate(old runtime.Object) error {
	log := zap.S().With("source", "webhook", "operation", "update", "resource", fmt.Sprintf("%s:%s", modifiedResource.Namespace, modifiedResource.Name))
	log.Info("Validate update")

	if env.IsValidationDisabled() {
		log.Info("Validation disabled, skipping")
		return nil
	}

	oldResource := old.(*Verrazzano)
	log.Debugf("oldResource: %v", oldResource)
	log.Debugf("modifiedResource: %v", modifiedResource)

	// The profile field is immutable
	if oldResource.Spec.Profile != modifiedResource.Spec.Profile {
		return fmt.Errorf("Profile change is not allowed oldResource %s to %s", oldResource.Spec.Profile, modifiedResource.Spec.Profile)
	}

	// Check to see if the update is an upgrade request, and if it is valid and allowable
	err := ValidateUpgradeRequest(&oldResource.Spec, &modifiedResource.Spec)
	if err != nil {
		log.Error("Invalid upgrade request: %s", err.Error())
		return err
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (deletedResource *Verrazzano) ValidateDelete() error {
	log := zap.S().With("source", "webhook", "operation", "delete", "resource", fmt.Sprintf("%s:%s", deletedResource.Namespace, deletedResource.Name))
	log.Info("Validate delete")

	if env.IsValidationDisabled() {
		log.Info("Validation disabled, skipping")
		return nil
	}

	return nil
}
