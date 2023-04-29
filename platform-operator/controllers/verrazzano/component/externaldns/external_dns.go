// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/namespace"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"hash/fnv"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

	clusterRoleName        = ComponentName
	clusterRoleBindingName = ComponentName
)

var resolvedNamespace string

type isInstalledFuncType func(releaseName string, namespace string) (found bool, err error)

var isInstalledFunc isInstalledFuncType = helm.IsReleaseInstalled

func preInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("external-dns PreInstall dry run")
		return nil
	}

	compContext.Log().Debug("Creating namespace %s namespace if necessary", ComponentNamespace)
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		return nil
	}); err != nil {
		return compContext.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}

	if err := CopyOCIDNSSecret(compContext, ComponentNamespace); err != nil {
		return err
	}
	return nil
}

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
			targetDNSSecret.Data[ociSecretFileName] = append(dnsSecret.Data[k], []byte(fmt.Sprintf("compartment: %s", ociDNS.DNSZoneCompartmentOCID))...)
		}

		return nil
	}); err != nil {
		return compContext.Log().ErrorfNewErr("Failed to create or update the external DNS secret: %v", err)
	}
	return nil
}

func resolveExernalDNSNamespace(_ string) string {
	return ResolveExernalDNSNamespace()
}

func ResolveExernalDNSNamespace() string {
	if len(resolvedNamespace) > 0 {
		// Only resolve the namespace once per instance
		return resolvedNamespace
	}
	resolvedNamespace := ComponentNamespace
	logger := vzlog.DefaultLogger()
	releaseFound, err := isInstalledFunc(ComponentName, legacyNamespace)
	if err != nil {
		logger.ErrorfThrottled("Error listing %s helm release %v", err)
		return ""
	}
	isVerrazzanoManaged, err := namespace.CheckIfVerrazzanoManagedNamespaceExists(legacyNamespace)
	if err != nil {
		logger.ErrorfThrottled("Error listing %s helm release %v", err)
		return ""
	}
	if releaseFound && isVerrazzanoManaged {
		resolvedNamespace = legacyNamespace
	}
	logger.Oncef("Using namespace %s for component %s", resolvedNamespace, ComponentName)
	return resolvedNamespace
}

// postUninstall Clean up the cluster role/bindings
func postUninstall(log vzlog.VerrazzanoLogger, cli client.Client) error {
	log.Progressf("Deleting ClusterRoles and ClusterRoleBindings for external-dns")
	err := resource.Resource{
		Name:   clusterRoleName,
		Client: cli,
		Object: &rbacv1.ClusterRole{},
		Log:    log,
	}.Delete()
	if err != nil {
		return err
	}
	return resource.Resource{
		Name:   clusterRoleBindingName,
		Client: cli,
		Object: &rbacv1.ClusterRoleBinding{},
		Log:    log,
	}.Delete()
}

func getDeploymentNames() []types.NamespacedName {
	return []types.NamespacedName{
		{Name: ComponentName, Namespace: resolveExernalDNSNamespace("")},
	}
}

func (c externalDNSComponent) isExternalDNSReady(compContext spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", compContext.GetComponent())
	return ready.DeploymentsAreReady(compContext.Log(), compContext.Client(), getDeploymentNames(), 1, prefix)
}

// AppendOverrides builds the set of external-dns overrides for the helm install
func AppendOverrides(compContext spi.ComponentContext, releaseName string, namespace string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	effectiveCR := compContext.EffectiveCR()
	oci, err := getOCIDNS(effectiveCR)
	if err != nil {
		return kvs, err
	}
	// OCI DNS is configured, append all helm overrides for external DNS
	ids, err := getOrBuildIDs(compContext, releaseName, namespace)
	if err != nil {
		return kvs, err
	}
	ownerID := ids[0]
	txtPrefix := ids[1]
	compContext.Log().Debugf("Owner ID: %s, TXT record prefix: %s", ownerID, txtPrefix)
	arguments := []bom.KeyValue{
		{Key: "domainFilters[0]", Value: oci.DNSZoneName},
		{Key: "zoneIDFilters[0]", Value: oci.DNSZoneOCID},
		{Key: "ociDnsScope", Value: oci.DNSScope},
		{Key: "txtOwnerId", Value: ownerID},
		{Key: "txtPrefix", Value: txtPrefix},
		{Key: "extraVolumes[0].name", Value: "config"},
		{Key: "extraVolumes[0].secret.secretName", Value: oci.OCIConfigSecret},
		{Key: "extraVolumeMounts[0].name", Value: "config"},
		{Key: "extraVolumeMounts[0].mountPath", Value: "/etc/kubernetes/"},
	}
	for i, source := range getSources(effectiveCR) {
		arguments = append(arguments, bom.KeyValue{
			Key:   fmt.Sprintf("sources[%v]", i),
			Value: source,
		})
	}
	kvs = append(kvs, arguments...)
	return kvs, nil
}

func getSources(vz *vzapi.Verrazzano) []string {
	sources := []string{"ingress", "service"}
	if vzcr.IsIstioEnabled(vz) {
		sources = append(sources, "istio-gateway")
	}

	return sources
}

func getOCIDNS(vz *vzapi.Verrazzano) (*vzapi.OCI, error) {
	dns := vz.Spec.Components.DNS
	// Should never fail the next error checks if IsEnabled() is correct, but can't hurt to check
	if dns == nil {
		return nil, fmt.Errorf("DNS not configured for component %s", ComponentName)
	}
	oci := dns.OCI
	if oci == nil {
		return nil, fmt.Errorf("OCI must be configured for component %s", ComponentName)
	}
	return oci, nil
}

// getOrBuildIDs Get the owner and TXT prefix IDs from the Helm release if they exist and preserve it, otherwise build a new ones
func getOrBuildIDs(compContext spi.ComponentContext, releaseName string, namespace string) ([]string, error) {
	values, err := helm.GetReleaseStringValues(compContext.Log(), []string{ownerIDHelmKey, prefixKey}, releaseName, namespace)
	if err != nil {
		return []string{}, err
	}
	ownerID, ok := values[ownerIDHelmKey]
	if !ok {
		if ownerID, err = buildOwnerString(compContext.ActualCR().UID); err != nil {
			return []string{}, err
		}
	}
	prefixKey, ok := values[prefixKey]
	if !ok {
		prefixKey = buildPrefixKey(ownerID)
	}
	return []string{ownerID, prefixKey}, nil
}

func buildPrefixKey(ownerID string) string {
	return fmt.Sprintf("_%s-", ownerID)
}

// buildOwnerString Builds a unique owner string ID based on the Verrazzano CR UID and namespaced name
func buildOwnerString(uid types.UID) (string, error) {
	hash := fnv.New32a()
	_, err := hash.Write([]byte(fmt.Sprintf("%v", uid)))
	if err != nil {
		return "", err
	}
	sum := hash.Sum32()
	return fmt.Sprintf("v8o-%s", strconv.FormatUint(uint64(sum), 16)), nil
}

// GetOverrides gets the install overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.DNS != nil {
			return effectiveCR.Spec.Components.DNS.ValueOverrides
		}
		return []vzapi.Overrides{}
	}
	effectiveCR := object.(*installv1beta1.Verrazzano)
	if effectiveCR.Spec.Components.DNS != nil {
		return effectiveCR.Spec.Components.DNS.ValueOverrides
	}
	return []installv1beta1.Overrides{}
}
