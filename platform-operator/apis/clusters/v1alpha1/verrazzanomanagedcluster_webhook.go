// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"fmt"
	"net/url"

	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	log.Debug("Validate create")

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
	vz, err := v.validateVerrazzanoInstalled(client)
	if err != nil {
		return err
	}

	// The secret and configmap are required fields _only_ if Rancher is disabled
	if !vzconfig.IsRancherEnabled(vz) {
		err = v.validateSecret(client)
		if err != nil {
			return err
		}

		err = v.validateConfigMap(client)
		if err != nil {
			return err
		}
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *VerrazzanoManagedCluster) ValidateUpdate(old runtime.Object) error {
	log := zap.S().With("source", "webhook", "operation", "update", "resource", fmt.Sprintf("%s:%s", v.Namespace, v.Name))
	log.Debug("Validate update")

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
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.Verrazzano{}, &v1alpha1.VerrazzanoList{})
	meta_v1.AddToGroupVersion(scheme, v1alpha1.SchemeGroupVersion)

	return scheme
}

// validateSecret enforces that the CA secret name was specified and that the secret exists
func (v VerrazzanoManagedCluster) validateSecret(client client.Client) error {
	log := zap.S().With("source", "webhook", "operation", "update", "resource", fmt.Sprintf("%s:%s", v.Namespace, v.Name))
	log.Debug("Validate secret")

	if len(v.Spec.CASecret) == 0 {
		log.Debugf("No CA secret in namespace %s defined, using well-known CA", constants.VerrazzanoMultiClusterNamespace)
		return nil
	}
	secret := corev1.Secret{}
	nsn := types.NamespacedName{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      v.Spec.CASecret,
	}
	err := client.Get(context.TODO(), nsn, &secret)
	if err == nil {
		return nil
	}
	if errors.IsNotFound(err) {
		return fmt.Errorf("The CA secret %s does not exist in namespace %s", v.Spec.CASecret, constants.VerrazzanoMultiClusterNamespace)
	}
	return fmt.Errorf("Error getting the CA secret %s namespace %s. Error: %s", v.Spec.CASecret, constants.VerrazzanoMultiClusterNamespace, err.Error())
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

// validateVerrazzanoInstalled enforces that a Verrazzano installation successfully completed
func (v VerrazzanoManagedCluster) validateVerrazzanoInstalled(localClient client.Client) (*v1alpha1.Verrazzano, error) {
	// Get the Verrazzano resource
	verrazzano := v1alpha1.VerrazzanoList{}
	err := localClient.List(context.TODO(), &verrazzano, &client.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("Verrazzano must be installed: %v", err)
	}

	// Verify the state is install complete
	if len(verrazzano.Items) > 0 {
		for _, cond := range verrazzano.Items[0].Status.Conditions {
			if cond.Type == v1alpha1.CondInstallComplete {
				return &verrazzano.Items[0], nil
			}
		}
	}

	return nil, fmt.Errorf("the Verrazzano install must successfully complete (run the command %q to view the install status)", "kubectl get verrazzano -A")
}
