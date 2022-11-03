// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	"strings"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var getControllerRuntimeClient = validators.GetClient

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
	ValidateInstall(vz *Verrazzano) []error
	ValidateUpdate(old *Verrazzano, new *Verrazzano) []error
}

var componentValidator ComponentValidator

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

	// Verify only one instance of the operator is running
	if err := v.verifyPlatformOperatorSingleton(); err != nil {
		return err
	}

	client, err := getControllerRuntimeClient(newScheme())
	if err != nil {
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
		if errs := componentValidator.ValidateInstall(v); len(errs) > 0 {
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

	// Verify only one instance of the operator is running
	if err := v.verifyPlatformOperatorSingleton(); err != nil {
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

	// Check to see if the update is an upgrade request, and if it is valid and allowable
	newSpecVerString := strings.TrimSpace(v.Spec.Version)
	currStatusVerString := strings.TrimSpace(oldResource.Status.Version)
	currSpecVerString := strings.TrimSpace(oldResource.Spec.Version)
	err := validators.ValidateUpgradeRequest(newSpecVerString, currStatusVerString, currSpecVerString)
	if err != nil {
		log.Errorf("Invalid upgrade request: %s", err.Error())
		return err
	}

	client, err := getControllerRuntimeClient(newScheme())
	if err != nil {
		return err
	}
	if err := validateOCISecrets(client, &v.Spec); err != nil {
		return err
	}

	// hand the old and new Verrazzano to component validator to validate
	if componentValidator != nil {
		if errs := componentValidator.ValidateUpdate(oldResource, v); len(errs) > 0 {
			return validators.CombineErrors(errs)
		}
	}

	return nil
}

// verifyPlatformOperatorSingleton Verifies that only one instance of the VPO is running; when upgrading operators,
// if the terminationGracePeriod for the pod is > 0 there's a chance that an old version may try to handle resource
// updates before terminating.  In the longer term we may want some kind of leader-election strategy to support
// multiple instances, if that makes sense.
func (v *Verrazzano) verifyPlatformOperatorSingleton() error {
	runtimeClient, err := getControllerRuntimeClient(newScheme())
	if err != nil {
		return err
	}
	var podList v1.PodList
	runtimeClient.List(context.TODO(), &podList,
		client.InNamespace(constants.VerrazzanoInstallNamespace),
		client.MatchingLabels{"app": "verrazzano-platform-operator"})
	if len(podList.Items) > 1 {
		return fmt.Errorf("Found more than one running instance of the platform operator, only one instance allowed")
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *Verrazzano) ValidateDelete() error {

	// Webhook is not configured for deletes so function will not be called.
	return nil
}

// newScheme creates a new scheme that includes this package's object for use by client
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	AddToScheme(scheme)
	clientgoscheme.AddToScheme(scheme)
	return scheme
}
