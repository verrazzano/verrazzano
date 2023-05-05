// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	ociSecretFileName = "oci.yaml"
	dnsGlobal         = "GLOBAL" //default
	dnsPrivate        = "PRIVATE"
)

func CopyOCIDNSSecret(compContext spi.ComponentContext, targetNamespace string) error {
	dns := compContext.EffectiveCR().Spec.Components.DNS
	if dns == nil || dns.OCI == nil {
		return nil
	}
	ociDNS := dns.OCI

	// Get OCI DNS secret from the verrazzano-install namespace
	dnsSecret := v1.Secret{}
	if err := compContext.Client().Get(context.TODO(), client.ObjectKey{Name: ociDNS.OCIConfigSecret, Namespace: constants.VerrazzanoInstallNamespace}, &dnsSecret); err != nil {
		return compContext.Log().ErrorfNewErr("Failed to find secret %s in the %s namespace: %v", ociDNS.OCIConfigSecret, constants.VerrazzanoInstallNamespace, err)
	}

	//check if scope value is valid
	scope := ociDNS.DNSScope
	if scope != dnsGlobal && scope != dnsPrivate && scope != "" {
		return compContext.Log().ErrorfNewErr("Failed, invalid OCI DNS scope value: %s. If set, value can only be 'GLOBAL' or 'PRIVATE", ociDNS.DNSScope)
	}

	// Attach compartment field to secret and apply it in the external DNS namespace
	targetDNSSecret := v1.Secret{}
	compContext.Log().Debug("Creating the external DNS secret")
	targetDNSSecret.Namespace = targetNamespace
	targetDNSSecret.Name = dnsSecret.Name
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &targetDNSSecret, func() error {
		targetDNSSecret.Data = make(map[string][]byte)

		// Verify that the oci secret has one value
		if len(dnsSecret.Data) != 1 {
			return compContext.Log().ErrorNewErr("Failed, OCI secret for OCI DNS should be created from one file")
		}

		// Extract data and create secret in the external DNS namespace
		for k := range dnsSecret.Data {
			targetDNSSecret.Data[ociSecretFileName] = append(dnsSecret.Data[k], []byte(fmt.Sprintf("\ncompartment: %s\n", ociDNS.DNSZoneCompartmentOCID))...)
		}

		return nil
	}); err != nil {
		return compContext.Log().ErrorfNewErr("Failed to create or update the external DNS secret: %v", err)
	}
	return nil
}
