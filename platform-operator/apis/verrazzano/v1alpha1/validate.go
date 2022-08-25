// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ValidateProfile check that requestedProfile is valid
func ValidateProfile(requestedProfile ProfileType) error {
	if len(requestedProfile) != 0 {
		switch requestedProfile {
		case Prod, Dev, ManagedCluster:
			return nil
		default:
			return fmt.Errorf("Requested profile %s is invalid, valid options are dev, prod, or managed-cluster",
				requestedProfile)
		}
	}
	return nil
}

// ValidateActiveInstall enforces that only one install of Verrazzano is allowed.
func ValidateActiveInstall(client client.Client) error {
	vzList := &VerrazzanoList{}

	err := client.List(context.Background(), vzList)
	if err != nil {
		return err
	}

	if len(vzList.Items) != 0 {
		return fmt.Errorf("Only one install of Verrazzano is allowed")
	}

	return nil
}

// ValidateInProgress makes sure there is not an install, uninstall or upgrade in progress
func ValidateInProgress(old *Verrazzano) error {
	if old.Status.State == "" || old.Status.State == VzStateReady || old.Status.State == VzStateFailed || old.Status.State == VzStatePaused || old.Status.State == VzStateReconciling {
		return nil
	}
	return fmt.Errorf(validators.ValidateInProgressError)
}

// validateOCISecrets - Validate that the OCI DNS and Fluentd OCI secrets required by install exists, if configured
func validateOCISecrets(client client.Client, spec *VerrazzanoSpec) error {
	//figure out which is which, pass in secret

	//if OCIDNS Secret
	if spec.Components.DNS != nil || spec.Components.DNS.OCI != nil {
		secret := &corev1.Secret{}
		ociDNSConfigSecret := spec.Components.DNS.OCI.OCIConfigSecret
		if err := validators.ValidateOCIDNSSecret(client, secret, ociDNSConfigSecret); err != nil {
			return err
		}
	}

	//if fluentd OCI Auth secret
	if spec.Components.Fluentd != nil || spec.Components.Fluentd.OCI != nil {
		apiSecretName := spec.Components.Fluentd.OCI.APISecret
		if err := validators.ValidateFluentdOCIAuthSecret(client, apiSecretName); err != nil {
			return err
		}
	}
	return nil
}

// ValidateInstallOverrides checks that the overrides slice has only one override type per slice item
func ValidateInstallOverrides(Overrides []Overrides) error {
	overridePerItem := 0
	for _, override := range Overrides {
		if override.ConfigMapRef != nil {
			overridePerItem++
		}
		if override.SecretRef != nil {
			overridePerItem++
		}
		if override.Values != nil {
			overridePerItem++
		}
		if overridePerItem > 1 {
			return fmt.Errorf("Invalid install overrides. Cannot specify more than one override type in the same list element")
		}
		if overridePerItem == 0 {
			return fmt.Errorf("Invalid install overrides. No override specified")
		}
		overridePerItem = 0
	}
	return nil
}
