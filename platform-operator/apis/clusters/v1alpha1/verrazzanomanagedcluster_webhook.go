// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// MultiClusterNamespace is the namespace that contains multi-cluster resources
const MultiClusterNamespace = "verrazzano-mc"

var getClientFunc = getClient
var _ webhook.Validator = &VerrazzanoManagedCluster{}

// SetupWebhookWithManager is used to let the controller manager know about the webhook
func (v *VerrazzanoManagedCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(v).
		Complete()
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *VerrazzanoManagedCluster) ValidateCreate() error {
	log := zap.S().With("source", "webhook", "operation", "create", "resource", fmt.Sprintf("%s:%s", v.Namespace, v.Name))
	log.Info("Validate create")

	if !config.Get().WebhookValidationEnabled {
		log.Info("Validation disabled, skipping")
		return nil
	}
	if v.ObjectMeta.Namespace != MultiClusterNamespace {
		return fmt.Errorf("Namespace for the resource must be %s", MultiClusterNamespace)
	}
	client, err := getClientFunc()
	if err != nil {
		return err
	}
	err = v.validateSecret(client)
	if err != nil {
		return err
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *VerrazzanoManagedCluster) ValidateUpdate(old runtime.Object) error {
	log := zap.S().With("source", "webhook", "operation", "update", "resource", fmt.Sprintf("%s:%s", v.Namespace, v.Name))
	log.Info("Validate update")

	if !config.Get().WebhookValidationEnabled {
		log.Info("Validation disabled, skipping")
		return nil
	}
	oldResource := old.(*VerrazzanoManagedCluster)
	log.Debugf("oldResource: %v", oldResource)
	log.Debugf("v: %v", v)
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *VerrazzanoManagedCluster) ValidateDelete() error {
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

// validateSecret enforces that the Prometheus secret name was specified and that the secret exists
func (v VerrazzanoManagedCluster) validateSecret(client client.Client) error {
	if len(v.Spec.PrometheusSecret) == 0 {
		return fmt.Errorf("The name of the Prometheus secret in namespace %s must be specified", MultiClusterNamespace)
	}
	secret := corev1.Secret{}
	nsn := types.NamespacedName{
		Namespace: MultiClusterNamespace,
		Name:      v.Spec.PrometheusSecret,
	}
	err := client.Get(context.TODO(), nsn, &secret)
	if err == nil {
		return nil
	}
	if errors.IsNotFound(err) {
		return fmt.Errorf("The Prometheus secret %s does not exist in namespace %s", v.Spec.PrometheusSecret, MultiClusterNamespace)
	}
	return fmt.Errorf("Error getting the Prometheus secret %s namespace %s. Error: %s", v.Spec.PrometheusSecret, MultiClusterNamespace, err.Error())
}
