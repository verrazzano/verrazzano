// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"fmt"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var getClientFunc = getClient
var _ webhook.Validator = &VerrazzanoProject{}

// log is for logging in this package.
var log = logf.Log.WithName("ingresstrait-resource")

// SetupWebhookWithManager is used to let the controller manager know about the webhook
func (v *VerrazzanoProject) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(v).
		Complete()
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *VerrazzanoProject) ValidateCreate() error {
	log.Info("validate create", "name", v.Name)

	if v.ObjectMeta.Namespace != constants.VerrazzanoMultiClusterNamespace {
		return fmt.Errorf("Namespace for the resource must be %q", constants.VerrazzanoMultiClusterNamespace)
	}
	/*
		client, err := getClientFunc()
		if err != nil {
			return err
		}
	*/
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *VerrazzanoProject) ValidateUpdate(old runtime.Object) error {
	log.Info("validate update", "name", v.Name)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *VerrazzanoProject) ValidateDelete() error {
	log.Info("validate delete", "name", v.Name)

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
	scheme.AddKnownTypes(schema.GroupVersion{
		Version: "v1",
	}, &corev1.Secret{})
	return scheme
}
