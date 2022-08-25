// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1beta1

import (
	"fmt"
	"strings"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var getControllerRuntimeClient = getClient

// SetupWebhookWithManager is used to let the controller manager know about the webhook
func (v *Verrazzano) SetupWebhookWithManager(mgr ctrl.Manager, log *zap.SugaredLogger) error {
	// clean up any temp files that may have been left over after a container restart
	if err := validators.CleanTempFiles(log); err != nil {
		return err
	}
	return ctrl.NewWebhookManagedBy(mgr).
		For(v).
		Complete()
}

var _ webhook.Validator = &Verrazzano{}

// +k8s:deepcopy-gen=false

type ComponentValidator interface {
	ValidateInstallV1Beta1(vz *Verrazzano) []error
	ValidateUpdateV1Beta1(old *Verrazzano, new *Verrazzano) []error
}

var componentValidator ComponentValidator = nil

func SetComponentValidator(v ComponentValidator) {
	componentValidator = v
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *Verrazzano) ValidateCreate() error {
	log := zap.S().With("source", "webhook", "operation", "create", "resource", fmt.Sprintf("%s:%s", v.Namespace, v.Name))
	log.Info("Validate create")

	if !config.Get().WebhookValidationEnabled {
		log.Info("Validation disabled, skipping")
		return nil
	}

	client, err := getControllerRuntimeClient()
	if err != nil {
		return err
	}

	// Verify only one instance of the operator is running
	if err := validators.VerifyPlatformOperatorSingleton(client); err != nil {
		return err
	}

	// Validate that only one install is allowed.
	if err := ValidateActiveInstall(client); err != nil {
		return err
	}

	if err := validators.ValidateVersion(v.Spec.Version); err != nil {
		return err
	}

	if err := ValidateProfile(v.Spec.Profile); err != nil {
		return err
	}

	if err := validateOCISecrets(client, &v.Spec); err != nil {
		return err
	}

	// hand the Verrazzano to component validator to validate
	if componentValidator != nil {
		if errs := componentValidator.ValidateInstallV1Beta1(v); len(errs) > 0 {
			return validators.CombineErrors(errs)
		}
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *Verrazzano) ValidateUpdate(old runtime.Object) error {
	log := zap.S().With("source", "webhook", "operation", "update", "resource", fmt.Sprintf("%s:%s", v.Namespace, v.Name))
	log.Info("Validate update")

	if !config.Get().WebhookValidationEnabled {
		log.Info("Validation disabled, skipping")
		return nil
	}

	client, err := getControllerRuntimeClient()
	if err != nil {
		return err
	}

	// Verify only one instance of the operator is running
	if err := validators.VerifyPlatformOperatorSingleton(client); err != nil {
		return err
	}

	oldResource := old.(*Verrazzano)
	log.Debugf("oldResource: %v", oldResource)
	log.Debugf("v: %v", v)

	// Only enable updates are not allowed when an install or an upgrade is in in progress
	if err := ValidateInProgress(oldResource); err != nil {
		return err
	}

	// The profile field is immutable
	if oldResource.Spec.Profile != v.Spec.Profile {
		return fmt.Errorf("Profile change is not allowed oldResource %s to %s", oldResource.Spec.Profile, v.Spec.Profile)
	}

	//ASK DEVA ABOUT CLIENT, ERR GET CONTROLLER RUNTIME. needs to happen before platformoperatorsingleton

	// Check to see if the update is an upgrade request, and if it is valid and allowable
	newSpecVerString := strings.TrimSpace(v.Spec.Version)
	currStatusVerString := strings.TrimSpace(oldResource.Status.Version)
	currSpecVerString := strings.TrimSpace(oldResource.Spec.Version)
	err = validators.ValidateUpgradeRequest(newSpecVerString, currStatusVerString, currSpecVerString)

	if err != nil {
		log.Errorf("Invalid upgrade request: %s", err.Error())
		return err
	}
	if err := validateOCISecrets(client, &v.Spec); err != nil {
		return err
	}

	// hand the old and new Verrazzano to component validator to validate
	if componentValidator != nil {
		if errs := componentValidator.ValidateUpdateV1Beta1(oldResource, v); len(errs) > 0 {
			return validators.CombineErrors(errs)
		}
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
	clientgoscheme.AddToScheme(scheme)
	return scheme
}
