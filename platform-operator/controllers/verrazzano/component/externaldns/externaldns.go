// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
	"time"
)

// ComponentName is the name of the component
const (
	ComponentName = "external-dns"
	externalDNSNamespace = "cert-manager"
	externalDNSDeploymentName = "external-dns"
	ociSecretFileName = "oci.yaml"
)

func (e externalDNSComponent) PreInstall(compContext spi.ComponentContext) error {
	// Get OCI DNS secret from the verrazzano-install namespace
	dns := compContext.EffectiveCR().Spec.Components.DNS
	dnsSecret := v1.Secret{}
	if err := compContext.Client().Get(context.TODO(), client.ObjectKey{Name: dns.OCI.OCIConfigSecret, Namespace: constants.VerrazzanoInstallNamespace}, &dnsSecret); err != nil {
		compContext.Log().Errorf("Could not find secret %s in the %s namespace: %s", dns.OCI.OCIConfigSecret, constants.VerrazzanoInstallNamespace, err)
		return err
	}

	// Attach compartment to secret and apply it in the external DNS namespace
	externalDNSSecret := v1.Secret{}
	externalDNSSecret.Data = make(map[string][]byte)
	compContext.Log().Debug("Creating the external DNS secret")

	// Verify that the oci secret has one value
	if len(dnsSecret.Data) != 1 {
		compContext.Log().Errorf("OCI secret for OCI DNS should be created from one file")
	}

	// Extract data and create secret in the external DNS namespace
	for k := range dnsSecret.Data {
		externalDNSSecret.Data[ociSecretFileName] = append(dnsSecret.Data[k], []byte(fmt.Sprintf("  compartment: %s", dns.OCI.DNSZoneCompartmentOCID))...)
	}
	externalDNSSecret.Namespace = externalDNSNamespace
	externalDNSSecret.Name = dnsSecret.Name
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &externalDNSSecret, func() error {
		return nil
	}); err != nil {
		compContext.Log().Errorf("Failed to create or update the external DNS secret: %s", err)
		return err
	}
	return nil
}

func (e externalDNSComponent) IsReady(compContext spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{Name: externalDNSDeploymentName, Namespace: externalDNSNamespace},
	}
	return status.DeploymentsReady(compContext.Log(), compContext.Client(), deployments, 1)
}

func (e externalDNSComponent) IsEnabled(compContext spi.ComponentContext) bool {
	dns := compContext.EffectiveCR().Spec.Components.DNS
	if dns != nil && dns.OCI != nil {
		return true
	}
	return false
}

// AppendOverrides builds the set of external-dns overrides for the helm install
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Append all helm overrides for external DNS
	kvs = append(kvs, bom.KeyValue{Key: "domainFilters[0]", Value: compContext.EffectiveCR().Spec.Components.DNS.OCI.DNSZoneName})
	kvs = append(kvs, bom.KeyValue{Key: "zoneIDFilters[0]", Value: compContext.EffectiveCR().Spec.Components.DNS.OCI.DNSZoneOCID})
	nameTimeString := fmt.Sprintf("v8o-local-%s-%s", compContext.EffectiveCR().Spec.EnvironmentName, strconv.FormatInt(time.Now().Unix(), 10))
	kvs = append(kvs, bom.KeyValue{Key: "txtOwnerId", Value: nameTimeString})
	kvs = append(kvs, bom.KeyValue{Key: "txtPrefix", Value: "_" + nameTimeString})
	kvs = append(kvs, bom.KeyValue{Key: "extraVolumes[0].name", Value: "config"})
	kvs = append(kvs, bom.KeyValue{Key: "extraVolumes[0].secret.secretName", Value: compContext.EffectiveCR().Spec.Components.DNS.OCI.OCIConfigSecret})
	kvs = append(kvs, bom.KeyValue{Key: "extraVolumeMounts[0].name", Value: "config"})
	kvs = append(kvs, bom.KeyValue{Key: "extraVolumeMounts[0].mountPath", Value: "/etc/kubernetes/"})
	return kvs, nil
}