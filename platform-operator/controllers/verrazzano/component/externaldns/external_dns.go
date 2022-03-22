// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"hash/fnv"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

// ComponentName is the name of the component
const (
	ociSecretFileName      = "oci.yaml"
	dnsGlobal              = "GLOBAL" //default
	dnsPrivate             = "PRIVATE"
	imagePullSecretHelmKey = "global.imagePullSecrets[0]"
	ownerIDHelmKey         = "txtOwnerId"
	prefixKey              = "txtPrefix"
)

func preInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager PreInstall dry run")
		return nil
	}

	compContext.Log().Debug("Creating namespace %s namespace if necessary", ComponentNamespace)
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		return nil
	}); err != nil {
		return compContext.Log().ErrorfNewErr("Failed to create or update the cert-manager namespace: %v", err)
	}

	// Get OCI DNS secret from the verrazzano-install namespace
	dns := compContext.EffectiveCR().Spec.Components.DNS
	dnsSecret := v1.Secret{}
	if err := compContext.Client().Get(context.TODO(), client.ObjectKey{Name: dns.OCI.OCIConfigSecret, Namespace: constants.VerrazzanoInstallNamespace}, &dnsSecret); err != nil {
		return compContext.Log().ErrorfNewErr("Failed to find secret %s in the %s namespace: %v", dns.OCI.OCIConfigSecret, constants.VerrazzanoInstallNamespace, err)
	}

	//check if scope value is valid
	scope := dns.OCI.DNSScope
	if scope != dnsGlobal && scope != dnsPrivate && scope != "" {
		return compContext.Log().ErrorfNewErr("Failed, invalid OCI DNS scope value: %s. If set, value can only be 'GLOBAL' or 'PRIVATE", dns.OCI.DNSScope)
	}

	// Attach compartment field to secret and apply it in the external DNS namespace
	externalDNSSecret := v1.Secret{}
	compContext.Log().Debug("Creating the external DNS secret")
	externalDNSSecret.Namespace = ComponentNamespace
	externalDNSSecret.Name = dnsSecret.Name
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &externalDNSSecret, func() error {
		externalDNSSecret.Data = make(map[string][]byte)

		// Verify that the oci secret has one value
		if len(dnsSecret.Data) != 1 {
			return compContext.Log().ErrorNewErr("Failed, OCI secret for OCI DNS should be created from one file")
		}

		// Extract data and create secret in the external DNS namespace
		for k := range dnsSecret.Data {
			externalDNSSecret.Data[ociSecretFileName] = append(dnsSecret.Data[k], []byte(fmt.Sprintf("compartment: %s", dns.OCI.DNSZoneCompartmentOCID))...)
		}

		return nil
	}); err != nil {
		return compContext.Log().ErrorfNewErr("Failed to create or update the external DNS secret: %v", err)
	}
	return nil
}

func isExternalDNSReady(compContext spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", compContext.GetComponent())
	return status.DeploymentsAreReady(compContext.Log(), compContext.Client(), deployments, 1, prefix)
}

// AppendOverrides builds the set of external-dns overrides for the helm install
func AppendOverrides(compContext spi.ComponentContext, releaseName string, namespace string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	dns := compContext.EffectiveCR().Spec.Components.DNS
	// Should never fail the next error checks if IsEnabled() is correct, but can't hurt to check
	if dns == nil {
		return kvs, fmt.Errorf("DNS not configured for component %s", ComponentName)
	}
	oci := dns.OCI
	if oci == nil {
		return kvs, fmt.Errorf("OCI must be configured for component %s", ComponentName)
	}
	// OCI DNS is configured, append all helm overrides for external DNS
	ownerString, err := getOrBuildOwnerID(compContext, releaseName, namespace)
	if err != nil {
		return kvs, err
	}
	txtPrefix, err := getOrBuildRecordPrefix(compContext, ownerString, releaseName, namespace)
	if err != nil {
		return kvs, err
	}
	compContext.Log().Debugf("Owner ID: %s, TXT record prefix: %s", ownerString, txtPrefix)
	arguments := []bom.KeyValue{
		{Key: "domainFilters[0]", Value: oci.DNSZoneName},
		{Key: "zoneIDFilters[0]", Value: oci.DNSZoneOCID},
		{Key: "ociDnsScope", Value: oci.DNSScope},
		{Key: "txtOwnerId", Value: ownerString},
		{Key: "txtPrefix", Value: txtPrefix},
		{Key: "extraVolumes[0].name", Value: "config"},
		{Key: "extraVolumes[0].secret.secretName", Value: oci.OCIConfigSecret},
		{Key: "extraVolumeMounts[0].name", Value: "config"},
		{Key: "extraVolumeMounts[0].mountPath", Value: "/etc/kubernetes/"},
	}
	kvs = append(kvs, arguments...)
	return kvs, nil
}

//getOrBuildOwnerID Get the owner ID from the Helm release if it exists and preserve it, otherwise build a unique ID
func getOrBuildOwnerID(compContext spi.ComponentContext, releaseName string, namespace string) (string, error) {
	value, found, err := helm.GetReleaseStringValue(compContext.Log(), ownerIDHelmKey, releaseName, namespace)
	if err != nil {
		return "", err
	}
	if found {
		return value, nil
	}
	ownerString, err := buildOwnerString(compContext.ActualCR())
	if err != nil {
		return "", err
	}
	return ownerString, nil
}

//getOrBuildRecordPrefix Get the TXT record prefix from the Helm release if it exists and preserve it, otherwise build
//  new one off of the ownerID passed in
func getOrBuildRecordPrefix(compContext spi.ComponentContext, ownerID string, releaseName string, namespace string) (string, error) {
	value, found, err := helm.GetReleaseStringValue(compContext.Log(), prefixKey, releaseName, namespace)
	if err != nil {
		return "", err
	}
	if found {
		return value, nil
	}
	return fmt.Sprintf("_%s-", ownerID), nil
}

//buildOwnerString Builds a unique owner string ID based on the Verrazzano CR UID and namespaced name
func buildOwnerString(cr *vzapi.Verrazzano) (string, error) {
	hash := fnv.New32a()
	_, err := hash.Write([]byte(cr.UID))
	if err != nil {
		return "", err
	}
	sum := hash.Sum32()
	return fmt.Sprintf("v8o-%s-%s-%s", cr.Namespace, cr.Name, strconv.FormatUint(uint64(sum), 10)), nil
}
