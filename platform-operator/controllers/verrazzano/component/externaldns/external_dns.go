// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ComponentName is the name of the component
const (
	ComponentName             = "external-dns"
	externalDNSNamespace      = "cert-manager"
	externalDNSDeploymentName = "external-dns"
	ociSecretFileName         = "oci.yaml"
	dnsGlobal                 = "GLOBAL" //default
	dnsPrivate                = "PRIVATE"
	imagePullSecretHelmKey    = "global.imagePullSecrets[0]"
)

func preInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager PreInstall dry run")
		return nil
	}

	// Get OCI DNS secret from the verrazzano-install namespace
	dns := compContext.EffectiveCR().Spec.Components.DNS
	dnsSecret := v1.Secret{}
	if err := compContext.Client().Get(context.TODO(), client.ObjectKey{Name: dns.OCI.OCIConfigSecret, Namespace: constants.VerrazzanoInstallNamespace}, &dnsSecret); err != nil {
		compContext.Log().Errorf("Could not find secret %s in the %s namespace: %s", dns.OCI.OCIConfigSecret, constants.VerrazzanoInstallNamespace, err)
		return err
	}

	//check if scope value is valid
	scope := dns.OCI.DNSScope
	if scope != dnsGlobal && scope != dnsPrivate && scope != "" {
		message := fmt.Sprintf("Invalid OCI DNS scope value: %s. If set, value can only be 'GLOBAL' or 'PRIVATE", dns.OCI.DNSScope)
		compContext.Log().Errorf(message)
		return errors.New(message)
	}

	// Attach compartment field to secret and apply it in the external DNS namespace
	externalDNSSecret := v1.Secret{}
	compContext.Log().Debug("Creating the external DNS secret")
	externalDNSSecret.Namespace = externalDNSNamespace
	externalDNSSecret.Name = dnsSecret.Name
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &externalDNSSecret, func() error {
		externalDNSSecret.Data = make(map[string][]byte)

		// Verify that the oci secret has one value
		if len(dnsSecret.Data) != 1 {
			return fmt.Errorf("OCI secret for OCI DNS should be created from one file")
		}

		// Extract data and create secret in the external DNS namespace
		for k := range dnsSecret.Data {
			externalDNSSecret.Data[ociSecretFileName] = append(dnsSecret.Data[k], []byte(fmt.Sprintf("compartment: %s", dns.OCI.DNSZoneCompartmentOCID))...)
		}

		return nil
	}); err != nil {
		compContext.Log().Errorf("Failed to create or update the external DNS secret: %s", err)
		return err
	}
	return nil
}

func isReady(compContext spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{Name: externalDNSDeploymentName, Namespace: externalDNSNamespace},
	}
	return status.DeploymentsReady(compContext.Log(), compContext.Client(), deployments, 1)
}

// AppendOverrides builds the set of external-dns overrides for the helm install
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Append all helm overrides for external DNS
	nameTimeString := fmt.Sprintf("v8o-local-%s-%s", compContext.EffectiveCR().Spec.EnvironmentName, strconv.FormatInt(time.Now().Unix(), 10))
	arguments := []bom.KeyValue{
		{Key: "domainFilters[0]", Value: compContext.EffectiveCR().Spec.Components.DNS.OCI.DNSZoneName},
		{Key: "zoneIDFilters[0]", Value: compContext.EffectiveCR().Spec.Components.DNS.OCI.DNSZoneOCID},
		{Key: "ociDnsScope", Value: compContext.EffectiveCR().Spec.Components.DNS.OCI.DNSScope},
		{Key: "txtOwnerId", Value: nameTimeString},
		{Key: "txtPrefix", Value: "_" + nameTimeString},
		{Key: "extraVolumes[0].name", Value: "config"},
		{Key: "extraVolumes[0].secret.secretName", Value: compContext.EffectiveCR().Spec.Components.DNS.OCI.OCIConfigSecret},
		{Key: "extraVolumeMounts[0].name", Value: "config"},
		{Key: "extraVolumeMounts[0].mountPath", Value: "/etc/kubernetes/"},
	}
	kvs = append(kvs, arguments...)
	return kvs, nil
}
