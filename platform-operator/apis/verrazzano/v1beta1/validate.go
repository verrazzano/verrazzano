package v1beta1

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

	//spec does not have ocidns secret nor fluentd oci auth secret
	return nil
}

// ValidateInstallOverrides checks that the overrides slice has only one override type per slice item
func ValidateInstallOverrides(overrides []Overrides) error {
	for _, override := range overrides {
		if err := isValidOverrideItems(override.ConfigMapRef, override.SecretRef, override.Values); err != nil {
			return err
		}
	}
	return nil
}

func isValidOverrideItems(cm *corev1.ConfigMapKeySelector, s *corev1.SecretKeySelector, v *apiextensionsv1.JSON) error {
	items := 0
	if cm != nil {
		items++
	}
	if s != nil {
		items++
	}
	if v != nil {
		items++
	}
	if items > 1 {
		return fmt.Errorf("Invalid install overrides. Cannot specify more than one override type in the same list element")
	}
	if items == 0 {
		return fmt.Errorf("Invalid install overrides. No override specified")
	}
	return nil
}
