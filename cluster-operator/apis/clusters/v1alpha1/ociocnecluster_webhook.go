// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// OCIOCNECluster should be both a validating and defaulting webhook
var _ webhook.Validator = &OCIOCNECluster{}
var _ webhook.Defaulter = &OCIOCNECluster{}

// SetupWebhookWithManager is used to let the controller manager know about the webhook
func (o *OCIOCNECluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(o).
		Complete()
}

func (o *OCIOCNECluster) ValidateCreate() error {
	return nil
}

func (o *OCIOCNECluster) ValidateUpdate(old runtime.Object) error {
	oldCluster, ok := old.(*OCIOCNECluster)
	if !ok {
		return errors.New("update resource must be of kind OCIOCNECluster")
	}
	if err := o.updateAllowed(oldCluster); err != nil {
		return err
	}
	return nil
}

func (o *OCIOCNECluster) updateAllowed(other *OCIOCNECluster) error {
	if o.Spec.IdentityRef != other.Spec.IdentityRef {
		return cannotBeChangedErr("spec.identityRef")
	}
	if o.Spec.PrivateRegistry != other.Spec.PrivateRegistry {
		return cannotBeChangedErr("spec.privateRegistry")
	}
	if o.Spec.KubernetesVersion != other.Spec.KubernetesVersion {
		return cannotBeChangedErr("spec.kubernetesVersion")
	}
	if o.Spec.OCI.Compartment != other.Spec.OCI.Compartment {
		return cannotBeChangedErr("spec.oci.compartment")
	}
	if o.Spec.OCI.Region != other.Spec.OCI.Region {
		return cannotBeChangedErr("spec.oci.region")
	}
	if o.Spec.OCI.ImageName != other.Spec.OCI.ImageName {
		return cannotBeChangedErr("spec.oci.imageName")
	}
	if o.Spec.OCI.ControlPlane != other.Spec.OCI.ControlPlane {
		return cannotBeChangedErr("spec.oci.controlPlane")
	}
	if o.Spec.OCI.Network != other.Spec.OCI.Network {
		return cannotBeChangedErr("spec.oci.network")
	}
	if o.Spec.OCI.SSHPublicKey != other.Spec.OCI.SSHPublicKey {
		return cannotBeChangedErr("spec.oci.sshPublicKey")
	}
	if len(o.Spec.OCI.Workers) != len(other.Spec.OCI.Workers) {
		return cannotBeChangedErr("spec.oci.workers")
	}
	if !sliceEqual(o.Spec.OCI.CloudInitCommands, other.Spec.OCI.CloudInitCommands) {
		return cannotBeChangedErr("spec.oci.cloudInitCommands")
	}
	if !sliceEqual(o.Spec.OCI.Workers, other.Spec.OCI.Workers) {
		return cannotBeChangedErr("spec.oci.workers")
	}
	return nil
}

func cannotBeChangedErr(field string) error {
	return fmt.Errorf("%s cannot be changed", field)
}

func sliceEqual[T comparable](a1, a2 []T) bool {
	if len(a1) != len(a2) {
		return false
	}

	for i := range a1 {
		v1 := a1[i]
		v2 := a2[i]
		if v1 != v2 {
			return false
		}
	}
	return true
}

func (o *OCIOCNECluster) ValidateDelete() error {
	return nil
}

func (o *OCIOCNECluster) Default() {
	if o.Spec.KubernetesVersion == "" {
		o.Spec.KubernetesVersion = "FROBBER!"
	}
}
