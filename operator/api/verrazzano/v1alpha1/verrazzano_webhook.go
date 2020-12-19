// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"fmt"

	"github.com/verrazzano/verrazzano/operator/internal/util/env"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var getControllerRuntimeClient = getClient

// SetupWebhookWithManager is used to let the controller manager know about the webhook
func (v *Verrazzano) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(v).
		Complete()
}

var _ webhook.Validator = &Verrazzano{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *Verrazzano) ValidateCreate() error {
	log := zap.S().With("source", "webhook", "operation", "create", "resource", fmt.Sprintf("%s:%s", v.Namespace, v.Name))
	log.Info("Validate create")

	client, err := getControllerRuntimeClient()
	if err != nil {
		return err
	}

	// Validate that only one install is allowed.
	if err := ValidateActiveInstall(client); err != nil {
		return err
	}

	if !env.IsValidationEnabled() {
		log.Info("Validation disabled, skipping")
		return nil
	}

	if err := ValidateVersion(v.Spec.Version); err != nil {
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *Verrazzano) ValidateUpdate(old runtime.Object) error {
	log := zap.S().With("source", "webhook", "operation", "update", "resource", fmt.Sprintf("%s:%s", v.Namespace, v.Name))
	log.Info("Validate update")

	if !env.IsValidationEnabled() {
		log.Info("Validation disabled, skipping")
		return nil
	}

	oldResource := old.(*Verrazzano)
	log.Debugf("oldResource: %v", oldResource)
	log.Debugf("v: %v", v)

	// Updates are not allowed when an install or an upgrade is in in progress
	if err := ValidateInProgress(oldResource.Status.State); err != nil {
		return err
	}

	// The profile field is immutable
	if oldResource.Spec.Profile != v.Spec.Profile {
		return fmt.Errorf("Profile change is not allowed oldResource %s to %s", oldResource.Spec.Profile, v.Spec.Profile)
	}

	// Check to see if the update is an upgrade request, and if it is valid and allowable
	err := ValidateUpgradeRequest(&oldResource.Spec, &v.Spec)
	if err != nil {
		log.Error("Invalid upgrade request: %s", err.Error())
		return err
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *Verrazzano) ValidateDelete() error {

	// Webhook is not configured for deletes so function will not be called.
	return nil
}

// getClient returns a controller runtime client for the Verrazzano resource
func getClient() (client.Client, error) {

	config, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}

	return client.New(config, client.Options{Scheme: newScheme()})
}

// newScheme creates a new scheme that includes this package's object for use by client
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	AddToScheme(scheme)
	return scheme
}
