// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"fmt"
	"net/url"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
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
	if v.ObjectMeta.Namespace != constants.VerrazzanoMultiClusterNamespace {
		return fmt.Errorf("Namespace for the resource must be %s", constants.VerrazzanoMultiClusterNamespace)
	}
	client, err := getClientFunc()
	if err != nil {
		return err
	}
	err = v.validateSecret(client)
	if err != nil {
		return err
	}
	err = v.validateConfigMap(client)
	if err != nil {
		return err
	}
	err = v.validateVerrazzanoInstalled(client)
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
	}, &corev1.Secret{}, &corev1.ConfigMap{})
	return scheme
}

// validateSecret enforces that the Prometheus secret name was specified and that the secret exists
func (v VerrazzanoManagedCluster) validateSecret(client client.Client) error {
	if len(v.Spec.PrometheusSecret) == 0 {
		return fmt.Errorf("The name of the Prometheus secret in namespace %s must be specified", constants.VerrazzanoMultiClusterNamespace)
	}
	secret := corev1.Secret{}
	nsn := types.NamespacedName{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      v.Spec.PrometheusSecret,
	}
	err := client.Get(context.TODO(), nsn, &secret)
	if err == nil {
		return nil
	}
	if errors.IsNotFound(err) {
		return fmt.Errorf("The Prometheus secret %s does not exist in namespace %s", v.Spec.PrometheusSecret, constants.VerrazzanoMultiClusterNamespace)
	}
	return fmt.Errorf("Error getting the Prometheus secret %s namespace %s. Error: %s", v.Spec.PrometheusSecret, constants.VerrazzanoMultiClusterNamespace, err.Error())
}

// validateConfigMap enforces that the verrazzano-admin-cluster secret name exists and server key has non-empty value
func (v VerrazzanoManagedCluster) validateConfigMap(client client.Client) error {
	cm := corev1.ConfigMap{}
	nsn := types.NamespacedName{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      constants.AdminClusterConfigMapName,
	}
	err := client.Get(context.TODO(), nsn, &cm)
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("The ConfigMap %s does not exist in namespace %s", constants.AdminClusterConfigMapName, constants.VerrazzanoMultiClusterNamespace)
		}
		return fmt.Errorf("Error getting the ConfigMap %s namespace %s. Error: %s", constants.AdminClusterConfigMapName, constants.VerrazzanoMultiClusterNamespace, err.Error())
	}
	_, err = url.ParseRequestURI(cm.Data[constants.ServerDataKey])
	if err != nil {
		return fmt.Errorf("Data with key %q contains invalid url %q in the ConfigMap %s namespace %s", constants.ServerDataKey, cm.Data[constants.ServerDataKey], constants.AdminClusterConfigMapName, constants.VerrazzanoMultiClusterNamespace)
	}
	return nil
}

// validateVerrazzanoInstalled enforces that a Verrazzano must have successfully completed
func (v VerrazzanoManagedCluster) validateVerrazzanoInstalled(client client.Client) error {
	// Get the Verrazzano resource
	verrazzano := v1alpha1.VerrazzanoList{}
	err := client.List(context.TODO(), &verrazzano)
	if err != nil || len(verrazzano.Items) == 0 {
		return fmt.Errorf("Verrazzano must be installed")
	}

	// Verify the state is install complete
	for _, cond := range verrazzano.Items[0].Status.Conditions {
		if cond.Type == v1alpha1.InstallComplete {
			return nil
		}
	}

	return fmt.Errorf("The Verrazzano install must successfully complete. Run the command %q to view the install status.", "kubectl get verrazzano -A")
}
