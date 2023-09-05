// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	ocnemeta "github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/ocne"
	vzerror "github.com/verrazzano/verrazzano/cluster-operator/internal/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net/url"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type (
	validationContext struct {
		Ctx               context.Context
		Cli               clipkg.Client
		OCIClientGetter   func(creds *oci.Credentials) (oci.Client, error)
		CredentialsLoader oci.CredentialsLoader
		Errors            *vzerror.ErrorAggregator
	}
)

var (
	NewValidationContext = newValidationContext
)

func newValidationContext() (*validationContext, error) {
	cli, err := getWebhookClient()
	if err != nil {
		return nil, err
	}
	return &validationContext{
		Ctx:               context.Background(),
		Cli:               cli,
		CredentialsLoader: oci.CredentialsLoaderImpl{},
		OCIClientGetter:   oci.NewClient,
		Errors:            vzerror.NewAggregator("\n"),
	}, nil
}

func getWebhookClient() (clipkg.Client, error) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	config, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}
	return clipkg.New(config, clipkg.Options{Scheme: scheme})
}

func addOCINodeErrors(ctx *validationContext, n OCINode, field string) {
	if n.Shape == nil {
		ctx.Errors.Addf("%s.shape is required", field)
	} else if !strings.Contains(*n.Shape, "Flex") {
		if n.OCPUs != nil {
			ctx.Errors.Addf("%s.ocpus should only be specified when using flex shapes", field)
		}
		if n.MemoryGbs != nil {
			ctx.Errors.Addf("%s.memoryGbs should only be specified when using flex shapes", field)
		}
	}
}

func addOCINetworkErrors(ctx *validationContext, ociClient oci.Client, network *Network, field string) {
	if network == nil {
		return
	}
	// If creating a new VCN, pre-existing VCN and subnet information should not be specified
	if network.CreateVCN {
		if len(network.VCN) > 0 {
			ctx.Errors.Addf("%s.vcn should not be specified when creating a new VCN", field)
		}
		if len(network.Subnets) > 0 {
			ctx.Errors.Addf("%s.subnets should not be specified when creating a new VCN", field)
		}
	} else { // If using an existing VCN and subnets, validate that these resources are accessible using the provided credentials.
		if len(network.Subnets) != 4 {
			ctx.Errors.Addf("%s.subnets should have 1 subnet each for worker, service-lb, control-plane, and control-plane-endpoint subnet roles.", field)
		}
		if _, err := ociClient.GetVCNByID(ctx.Ctx, network.VCN); err != nil {
			ctx.Errors.Addf("%s.vcn [%s] is not accessible", field, network.VCN)
		}
		subnetCache := map[string]bool{}
		for i, subnet := range network.Subnets {
			if ok := subnetCache[subnet.ID]; ok {
				continue
			}
			ociSubnet, err := ociClient.GetSubnetByID(ctx.Ctx, subnet.ID, string(subnet.Role))
			if err != nil {
				ctx.Errors.Addf("%s.subnets[%d] : [%s] is not accessible", field, i, subnet.ID)
			} else {
				subnetCache[ociSubnet.ID] = true
			}
		}
	}
}

func addOCNEErrors(ctx *validationContext, ocne OCNE, field string) {
	if _, err := ocnemeta.GetVersionDefaults(ctx.Ctx, ctx.Cli, ocne.Version); err != nil {
		ctx.Errors.Addf("%s.version [%s] is not a known OCNE version", field, ocne.Version)
	}
}

func addProxyErrors(ctx *validationContext, proxy *Proxy, field string) {
	if proxy == nil {
		return
	}
	if _, err := url.ParseRequestURI(proxy.HTTPSProxy); err != nil {
		ctx.Errors.Addf("%s.httpProxy is not a valid URL", field)
	}
	if _, err := url.ParseRequestURI(proxy.HTTPProxy); err != nil {
		ctx.Errors.Addf("%s.httpProxy is not a valid URL", field)
	}
}

func addPrivateRegistryErrors(ctx *validationContext, privateRegistry *PrivateRegistry, field string) {
	if privateRegistry == nil {
		return
	}
	if _, err := url.ParseRequestURI(privateRegistry.URL); err != nil {
		ctx.Errors.Addf("%s.url is not a valid URL", field)
	}
}
