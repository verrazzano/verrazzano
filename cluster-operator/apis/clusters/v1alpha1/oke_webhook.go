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
)

// OCNEOCIQuickCreate should be both a validating and defaulting webhook
var _ webhook.Validator = &OKEQuickCreate{}

// SetupWebhookWithManager is used to let the controller manager know about the webhook
func (o *OKEQuickCreate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(o).
		Complete()
}

func (o *OKEQuickCreate) ValidateCreate() error {
	ctx, err := NewValidationContext()
	if err != nil {
		return fmt.Errorf("failed to create validation context: %w", err)
	}
	nsn := o.Spec.IdentityRef.AsNamespacedName()
	creds, err := ctx.CredentialsLoader.GetCredentialsIfAllowed(ctx.Ctx, ctx.Cli, nsn, o.Namespace)
	if err != nil {
		return fmt.Errorf("cannot access OCI credentials %s/%s: %v", nsn.Namespace, nsn.Name, err)
	}
	ociClient, err := ctx.OCIClientGetter(creds)
	if err != nil {
		return fmt.Errorf("failed to create OCI Client: %w", err)
	}
	// Validate the OCI Network
	if o.Spec.OKE.Network != nil {
		addCNITypeErrors(ctx, o.Spec.OKE.Network.CNIType, "spec.oke.network")
		numSubnets := 3 // controlplane-endpoint, worker, service-lb
		if o.Spec.OKE.Network.CNIType == VCNNative {
			numSubnets++ // pod subnet
		}
		addOCINetworkErrors(ctx, ociClient, o.Spec.OKE.Network.Config, numSubnets, "spec.oke.network.config")
	}
	for i, np := range o.Spec.OKE.NodePools {
		addOCINodeErrors(ctx, np.OCINode, fmt.Sprintf("spec.oke.nodePools[%d]", i))
	}
	if ctx.Errors.HasError() {
		return ctx.Errors
	}
	return nil
}

func addCNITypeErrors(ctx *validationContext, cniType CNIType, field string) {
	switch cniType {
	case FlannelOverlay, VCNNative:
		return
	default:
		ctx.Errors.Addf("%s.cniType is invalid", field)
	}
}

func (o *OKEQuickCreate) ValidateUpdate(old runtime.Object) error {
	oldCluster, ok := old.(*OKEQuickCreate)
	if !ok {
		return errors.New("update resource must be of kind OKEQuickCreate")
	}
	if !reflect.DeepEqual(o.Spec, oldCluster.Spec) {
		return errors.New("spec updates are not permitted")
	}
	return nil
}
func (o *OKEQuickCreate) ValidateDelete() error {
	return nil
}
