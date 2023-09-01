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

var (
	// OCNEOCIQuickCreate should be both a validating and defaulting webhook
	_ webhook.Validator = &OCNEOCIQuickCreate{}
)

// SetupWebhookWithManager is used to let the controller manager know about the webhook
func (o *OCNEOCIQuickCreate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(o).
		Complete()
}

func (o *OCNEOCIQuickCreate) ValidateCreate() error {
	ctx, err := NewValidationContext()
	if err != nil {
		return fmt.Errorf("failed to create validation context: %w", err)
	}
	nsn := o.Spec.IdentityRef.AsNamespacedName()
	creds, err := ctx.CredentialsLoader.GetCredentialsIfAllowed(ctx.Ctx, ctx.Cli, nsn, o.Namespace)
	if err != nil {
		return fmt.Errorf("cannot access credentials %s/%s from namespace %s", nsn.Namespace, nsn.Name, o.Namespace)
	}
	ociClient, err := ctx.OCIClientGetter(creds)
	if err != nil {
		return fmt.Errorf("failed to create OCI Client: %w", err)
	}
	// Validate the general OCI spec
	addOCICommonErrors(ctx, ociClient, o.Spec.OCI.CommonOCI, "spec.oci")
	// Validate the OCI Network
	addOCINetworkErrors(ctx, ociClient, o.Spec.OCI.Network, "spec.oci.network")
	// Validate the OCI Nodes
	addOCINodeErrors(ctx, o.Spec.OCI.ControlPlane, "spec.oci.controlPlane")
	for i, worker := range o.Spec.OCI.Workers {
		addOCINodeErrors(ctx, worker.OCINode, fmt.Sprintf("spec.oci.workers[%d]", i))
	}
	addOCNEErrors(ctx, o.Spec.OCNE, "spec.ocne")
	addProxyErrors(ctx, o.Spec.Proxy, "spec.proxy")
	addPrivateRegistryErrors(ctx, o.Spec.PrivateRegistry, "spec.privateRegistry")
	if ctx.Errors.HasError() {
		return ctx.Errors
	}
	return nil
}

func (o *OCNEOCIQuickCreate) ValidateUpdate(old runtime.Object) error {
	oldCluster, ok := old.(*OCNEOCIQuickCreate)
	if !ok {
		return errors.New("update resource must be of kind OCNEOCIQuickCreate")
	}
	if err := o.updateAllowed(oldCluster); err != nil {
		return err
	}
	return nil
}

func (o *OCNEOCIQuickCreate) updateAllowed(other *OCNEOCIQuickCreate) error {
	return nil
}

func (o *OCNEOCIQuickCreate) ValidateDelete() error {
	return nil
}
