// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	_ webhook.Validator = &OCNEOCIQuickCreate{}
)

// SetupWebhookWithManager is used to let the controller manager know about the webhook
func (o *OCNEOCIQuickCreate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(o).
		Complete()
}

// ValidateCreate validates the OCNEOCIQuickCreate input.
// We do not provide a deep validation of OCI cloud resources, because the provided
// credentials may not have the necessary policies to do so.
func (o *OCNEOCIQuickCreate) ValidateCreate() (admission.Warnings, error) {
	warnings := []string{}
	ctx, err := NewValidationContext()
	if err != nil {
		return warnings, fmt.Errorf("failed to create validation context: %w", err)
	}
	nsn := o.Spec.IdentityRef.AsNamespacedName()
	creds, err := ctx.CredentialsLoader.GetCredentialsIfAllowed(ctx.Ctx, ctx.Cli, nsn, o.Namespace)
	if err != nil {
		return warnings, fmt.Errorf("cannot access OCI credentials %s/%s: %v", nsn.Namespace, nsn.Name, err)
	}
	ociClient, err := ctx.OCIClientGetter(creds)
	if err != nil {
		return warnings, fmt.Errorf("failed to create OCI Client: %w", err)
	}
	// Validate the OCI Network
	addOCINetworkErrors(ctx, ociClient, o.Spec.OCI.Network, 4, "spec.oci.network")
	// Validate the OCI Nodes
	addOCINodeErrors(ctx, o.Spec.OCI.ControlPlane, "spec.oci.controlPlane")
	for i, worker := range o.Spec.OCI.Workers {
		addOCINodeErrors(ctx, worker.OCINode, fmt.Sprintf("spec.oci.workers[%d]", i))
	}
	addOCNEErrors(ctx, o.Spec.OCNE, "spec.ocne")
	addProxyErrors(ctx, o.Spec.Proxy, "spec.proxy")
	addPrivateRegistryErrors(ctx, o.Spec.PrivateRegistry, "spec.privateRegistry")
	if ctx.Errors.HasError() {
		return warnings, ctx.Errors
	}
	return warnings, nil
}

// ValidateUpdate rejects any changes to the quick create spec.
func (o *OCNEOCIQuickCreate) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	warnings := []string{}
	oldCluster, ok := old.(*OCNEOCIQuickCreate)
	if !ok {
		return warnings, errors.New("update resource must be of kind OCNEOCIQuickCreate")
	}
	if !reflect.DeepEqual(o.Spec, oldCluster.Spec) {
		return warnings, errors.New("spec updates are not permitted")
	}
	return warnings, nil
}

func (o *OCNEOCIQuickCreate) ValidateDelete() (admission.Warnings, error) {
	return []string{}, nil
}
