// Copyright (c) 2021, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/namespace"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

// ComponentName is the name of the component
const (
	imagePullSecretHelmKey = "global.imagePullSecrets[0]"
	ownerIDHelmKey         = "txtOwnerId"
	prefixKey              = "txtPrefix"

	clusterRoleName                 = ComponentName
	clusterRoleBindingName          = ComponentName
	externalDNSIngressAnnotationKey = "external-dns.alpha.kubernetes.io/target"
)

type isInstalledFuncType func(releaseName string, namespace string) (found bool, err error)

var isLegacyNamespaceInstalledFunc isInstalledFuncType = helm.IsReleaseInstalled

func preInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("external-dns PreInstall dry run")
		return nil
	}

	compContext.Log().Debug("Creating namespace %s namespace if necessary", ComponentNamespace)
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		return nil
	}); err != nil {
		return compContext.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}

	if err := common.CopyOCIDNSSecret(compContext, ComponentNamespace); err != nil {
		return err
	}
	return nil
}

// resolveExernalDNSNamespace implements the HelmComponent contract to resolve a component namespace dynamically
func resolveExernalDNSNamespace(_ string) string {
	return ResolveExernalDNSNamespace()
}

// ResolveExernalDNSNamespace Resolves the namespace for External DNS based on an existing legacy instance vs a new one;
// if External-DNS exists in the cert-manager namespace AND the namespace is owned by Verrazzano, use the existing
// installed namespace; otherwise we will install it into the verrazzano-system namespace
//
// HelmComponent will cache the results when called from that context
func ResolveExernalDNSNamespace() string {
	resolvedNamespace := ComponentNamespace
	logger := vzlog.DefaultLogger()
	releaseFound, err := isLegacyNamespaceInstalledFunc(ComponentName, legacyNamespace)
	if err != nil {
		logger.ErrorfThrottled("Error listing %s helm release %v", err)
		return ""
	}
	isVerrazzanoManaged, err := namespace.CheckIfVerrazzanoManagedNamespaceExists(legacyNamespace)
	if err != nil {
		logger.ErrorfThrottled("Error checking if namespace %s is Verrazzano-managed: %v", legacyNamespace, err)
		return ""
	}
	if releaseFound && isVerrazzanoManaged {
		resolvedNamespace = legacyNamespace
	}
	logger.Oncef("Using namespace %s for component %s", resolvedNamespace, ComponentName)
	return resolvedNamespace
}

// preUninstall checks to make sure that all ingresses/services in the Verrazzano namespaces which
// have the external DNS target containing the word verrazzano have already been uninstalled, and
// also that the VMO has been uninstalled.
// If not, they may require external-dns to clean up the corresponding
// DNS records, so we cannot start external-dns uninstall yet.
func preUninstall(log vzlog.VerrazzanoLogger, cli client.Client) error {
	log.Progressf("Checking for leftover ingresses in Verrazzano component namespaces before uninstalling %s", ComponentName)

	namespaceList, err := listVerrazzanoNamespaces(cli)
	if err != nil {
		return err
	}
	for _, ns := range namespaceList.Items {
		ingressList := netv1.IngressList{}
		if err := cli.List(context.TODO(), &ingressList, client.InNamespace(ns.Name)); err != nil {
			log.Errorf("Failed to list ingresses in namespace %s: %v", ns.Name, err)
			return err
		}
		for _, ing := range ingressList.Items {
			if strings.Contains(ing.Annotations[externalDNSIngressAnnotationKey], constants.VzIngress) {
				log.Progressf("Component %s pre-uninstall is waiting for ingress %s in namespace %s to be uninstalled", ComponentName, ing.Name, ns.Name)
				return ctrlerrors.RetryableError{Source: ComponentName}
			}
		}
	}

	vmoUninstalled := verifyVMOUninstalled(log, cli)
	ingressNginxUninstalled := verifyIngressNginxUninstalled(cli)
	if !vmoUninstalled {
		log.Progressf("Component %s pre-uninstall is waiting for %s to be uninstalled", ComponentName, common.VMOComponentName)
		return ctrlerrors.RetryableError{Source: ComponentName}
	}
	if !ingressNginxUninstalled {
		log.Progressf("Component %s pre-uninstall is waiting for the %s namespace to be deleted", ComponentName, constants.IngressNginxNamespace)
		return ctrlerrors.RetryableError{Source: ComponentName}
	}
	return nil
}

// listVerrazzanoNamespaces lists all Verrazzano labeled namespaces
func listVerrazzanoNamespaces(cli client.Client) (*corev1.NamespaceList, error) {
	// []string{constants.VerrazzanoSystemNamespace, constants.KeycloakNamespace, vzconst.RancherSystemNamespace, vzconst.VerrazzanoMonitoringNamespace}
	nsList := corev1.NamespaceList{}
	if err := cli.List(context.TODO(), &nsList, client.HasLabels{vzconst.LabelVerrazzanoNamespace}); client.IgnoreNotFound(err) != nil {
		return nil, err
	}
	return &nsList, nil
}

func verifyIngressNginxUninstalled(cli client.Client) bool {
	ns := corev1.Namespace{}
	// The ingress nginx namespace would be deleted if the ingress nginx component has been uninstalled.
	err := cli.Get(context.TODO(), types.NamespacedName{Name: constants.IngressNginxNamespace}, &ns)
	return err != nil && errors.IsNotFound(err)
}

func verifyVMOUninstalled(log vzlog.VerrazzanoLogger, cli client.Client) bool {
	vmoDeployment := appsv1.Deployment{}
	err := cli.Get(context.TODO(),
		types.NamespacedName{Namespace: common.VMOComponentNamespace, Name: common.VMOComponentName},
		&vmoDeployment)

	if err != nil {
		// VMO is uninstalled if the VMO deployment does not exist
		if errors.IsNotFound(err) {
			return true
		}
		log.Errorf("Failed to get VMO deployment: %v", err)
	}
	return false
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

func (c *externalDNSComponent) getDeploymentNames(ctx spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{
		{Name: ComponentName, Namespace: c.HelmComponent.ResolveNamespace(ctx)},
	}
}

func (c *externalDNSComponent) isExternalDNSReady(compContext spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", compContext.GetComponent())
	return ready.DeploymentsAreReady(compContext.Log(), compContext.Client(), c.getDeploymentNames(compContext), 1, prefix)
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
