// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// getDynamicClientFuncSig defines the signature for a function that returns a k8s dynamic client
type getDynamicClientFuncSig func() (dynamic.Interface, error)

// getDynamicClientFunc is the function for getting a k8s dynamic client - this allows us to override
// the function for unit testing
var getDynamicClientFunc getDynamicClientFuncSig = getDynamicClient

// Constants for Kubernetes resource names
const (
	// note: VZ-5241 In Rancher 2.6.3 the agent was moved from cattle-fleet-system ns
	// to a new cattle-fleet-local-system ns, the rancher-operator-system ns was
	// removed, and the rancher-operator is no longer deployed
	FleetSystemNamespace      = "cattle-fleet-system"
	FleetLocalSystemNamespace = "cattle-fleet-local-system"
	defaultSecretNamespace    = "cert-manager"
	rancherTLSSecretName      = "tls-ca"
	defaultVerrazzanoName     = "verrazzano-ca-certificate-secret"
	fleetAgentDeployment      = "fleet-agent"
	fleetControllerDeployment = "fleet-controller"
	gitjobDeployment          = "gitjob"
	rancherWebhookDeployment  = "rancher-webhook"
)

// Helm Chart setter keys
const (
	ingressTLSSourceKey     = "ingress.tls.source"
	additionalTrustedCAsKey = "additionalTrustedCAs"
	privateCAKey            = "privateCA"

	// Rancher registry Keys
	useBundledSystemChartKey = "useBundledSystemChart"
	systemDefaultRegistryKey = "systemDefaultRegistry"

	// LE Keys
	letsEncryptIngressClassKey = "letsEncrypt.ingress.class"
	letsEncryptEmailKey        = "letsEncrypt.email"
	letsEncryptEnvironmentKey  = "letsEncrypt.environment"
)

const (
	letsEncryptTLSSource       = "letsEncrypt"
	caTLSSource                = "secret"
	caCertsPem                 = "cacerts.pem"
	caCert                     = "ca.crt"
	privateCAValue             = "true"
	useBundledSystemChartValue = "true"
)

const (
	SettingServerURL               = "server-url"
	KontainerDriverOKE             = "oraclecontainerengine"
	NodeDriverOCI                  = "oci"
	ClusterLocal                   = "local"
	AuthConfigLocal                = "local"
	UserVerrazzano                 = "u-verrazzano"
	UserVerrazzanoDescription      = "Verrazzano Admin"
	GlobalRoleBindingVerrazzano    = "grb-" + UserVerrazzano
	SettingUIPL                    = "ui-pl"
	SettingUIPLValueVerrazzano     = "Verrazzano"
	SettingUILogoLight             = "ui-logo-light"
	SettingUILogoLightLogoFilePath = "/usr/share/rancher/ui-dashboard/dashboard/_nuxt/pkg/verrazzano/assets/images/verrazzano-light.svg"
	SettingUILogoDark              = "ui-logo-dark"
	SettingUILogoDarkLogoFilePath  = "/usr/share/rancher/ui-dashboard/dashboard/_nuxt/pkg/verrazzano/assets/images/verrazzano-dark.svg"
	SettingUILogoValueprefix       = "data:image/svg+xml;base64,"
	SettingUIPrimaryColor          = "ui-primary-color"
	SettingUIPrimaryColorValue     = "rgb(48, 99, 142)"
	SettingUILinkColor             = "ui-link-color"
	SettingUILinkColorValue        = "rgb(49, 118, 217)"
)

// auth config
const (
	AuthConfigKeycloakURLPathVerifyAuth           = "/verify-auth"
	AuthConfigKeycloakURLPathIssuer               = "/auth/realms/verrazzano-system"
	AuthConfigKeycloakURLPathAuthEndPoint         = "/auth/realms/verrazzano-system/protocol/openid-connect/auth"
	AuthConfigKeycloakClientIDRancher             = "rancher"
	AuthConfigKeycloakAccessMode                  = "unrestricted"
	AuthConfigKeycloakAttributeAccessMode         = "accessMode"
	AuthConfigKeycloakAttributeClientID           = "clientId"
	AuthConfigAttributeEnabled                    = "enabled"
	AuthConfigKeycloakAttributeGroupSearchEnabled = "groupSearchEnabled"
	AuthConfigKeycloakAttributeAuthEndpoint       = "authEndpoint"
	AuthConfigKeycloakAttributeIssuer             = "issuer"
	AuthConfigKeycloakAttributeRancherURL         = "rancherUrl"
)

// attributes
const (
	UserAttributeDisplayName                              = "displayName"
	UserAttributeUserName                                 = "username"
	UserAttributePrincipalIDs                             = "principalIds"
	UserAttributeDescription                              = "description"
	GlobalRoleBindingAttributeRoleName                    = "globalRoleName"
	GlobalRoleBindingAttributeUserName                    = "userName"
	ClusterRoleTemplateBindingAttributeClusterName        = "clusterName"
	ClusterRoleTemplateBindingAttributeGroupPrincipalName = "groupPrincipalName"
	ClusterRoleTemplateBindingAttributeRoleTemplateName   = "roleTemplateName"
	RoleTemplateAttributeBuiltin                          = "builtin"
	RoleTemplateAttributeContext                          = "context"
	RoleTemplateAttributeDisplayName                      = "displayName"
	RoleTemplateAttributeExternal                         = "external"
	RoleTemplateAttributeHidden                           = "hidden"
	RoleTemplateAttributeRules                            = "rules"
)

// roles and groups
const (
	AdminRoleName               = "admin"
	VerrazzanoAdminRoleName     = "verrazzano-admin"
	ViewRoleName                = "view"
	VerrazzanoMonitorRoleName   = "verrazzano-monitor"
	ClusterMemberRoleName       = "cluster-member"
	VerrazzanoAdminsGroupName   = "verrazzano-admins"
	VerrazzanoMonitorsGroupName = "verrazzano-monitors"
	GroupKey                    = "group"
	ClusterRoleKey              = "clusterRole"
)

// prefixes
const (
	UserPrincipalKeycloakPrefix  = "keycloakoidc_user://"
	GroupPrincipalKeycloakPrefix = "keycloakoidc_group://"
	UserPrincipalLocalPrefix     = "local://"
)

const (
	rancherChartsClusterRepoName        = "rancher-charts"
	rancherPartnerChartsClusterRepoName = "rancher-partner-charts"
	rancherRke2ChartsClusterRepoName    = "rancher-rke2-charts"

	chartDefaultBranchName = "chart-default-branch"
)

var GVKCluster = common.GetRancherMgmtAPIGVKForKind("Cluster")
var GVKNodeDriver = common.GetRancherMgmtAPIGVKForKind("NodeDriver")
var GVKKontainerDriver = common.GetRancherMgmtAPIGVKForKind("KontainerDriver")
var GVKUser = common.GetRancherMgmtAPIGVKForKind("User")
var GVKGlobalRoleBinding = common.GetRancherMgmtAPIGVKForKind("GlobalRoleBinding")
var GVKClusterRoleTemplateBinding = common.GetRancherMgmtAPIGVKForKind("ClusterRoleTemplateBinding")
var GVKRoleTemplate = common.GetRancherMgmtAPIGVKForKind("RoleTemplate")
var GroupRolePairs = []map[string]string{
	{
		GroupKey:       VerrazzanoAdminsGroupName,
		ClusterRoleKey: AdminRoleName,
	},
	{
		GroupKey:       VerrazzanoAdminsGroupName,
		ClusterRoleKey: VerrazzanoAdminRoleName,
	},
	{
		GroupKey:       VerrazzanoAdminsGroupName,
		ClusterRoleKey: ClusterMemberRoleName,
	},
	{
		GroupKey:       VerrazzanoMonitorsGroupName,
		ClusterRoleKey: ViewRoleName,
	},
	{
		GroupKey:       VerrazzanoMonitorsGroupName,
		ClusterRoleKey: VerrazzanoMonitorRoleName,
	},
	{
		GroupKey:       VerrazzanoMonitorsGroupName,
		ClusterRoleKey: ClusterMemberRoleName,
	},
}

var cattleSettingsGVR = schema.GroupVersionResource{
	Group:    "management.cattle.io",
	Version:  "v3",
	Resource: "settings",
}

var cattleClusterReposGVR = schema.GroupVersionResource{
	Group:    "catalog.cattle.io",
	Version:  "v1",
	Resource: "clusterrepos",
}

func useAdditionalCAs(acme vzapi.Acme) bool {
	return acme.Environment != "production"
}

func getRancherHostname(c client.Client, vz *vzapi.Verrazzano) (string, error) {
	dnsSuffix, err := vzconfig.GetDNSSuffix(c, vz)
	if err != nil {
		return "", err
	}
	env := vz.Spec.EnvironmentName
	if len(env) == 0 {
		env = constants.DefaultEnvironmentName
	}
	rancherHostname := fmt.Sprintf("%s.%s.%s", common.RancherName, env, dnsSuffix)
	return rancherHostname, nil
}

// isRancherReady checks that the Rancher component is in a 'Ready' state, as defined
// in the body of this function
func isRancherReady(ctx spi.ComponentContext) bool {
	log := ctx.Log()
	c := ctx.Client()

	deployments := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
		{
			Name:      rancherWebhookDeployment,
			Namespace: ComponentNamespace,
		},
		{
			Name:      fleetAgentDeployment,
			Namespace: FleetLocalSystemNamespace,
		},
		{
			Name:      fleetControllerDeployment,
			Namespace: FleetSystemNamespace,
		},
		{
			Name:      gitjobDeployment,
			Namespace: FleetSystemNamespace,
		},
	}

	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return ready.DeploymentsAreReady(log, c, deployments, 1, prefix)
}

// chartsNotUpdatedWorkaround - workaround for VZ-7053, where some of the Helm charts are not
// getting updated after Rancher upgrade. This workaround will scale the Rancher deployment down
// and then delete the ClusterRepo resources. This must be done in PreUpgrade. Scaling down is
// necessary to prevent an old pod from updating the ClusterRepo with an old commit. When Rancher
// is upgraded, new Rancher pods will start and create new ClusterRepo resources with the correct
// chart data.
func chartsNotUpdatedWorkaround(ctx spi.ComponentContext) error {
	if err := scaleDownRancherDeployment(ctx.Client(), ctx.Log()); err != nil {
		return err
	}
	return deleteClusterRepos(ctx.Log())
}

// scaleDownRancherDeployment scales the Rancher deployment down to zero replicas
func scaleDownRancherDeployment(c client.Client, log vzlog.VerrazzanoLogger) error {
	deployment := appsv1.Deployment{}
	namespacedName := types.NamespacedName{Name: common.RancherName, Namespace: common.CattleSystem}
	if err := c.Get(context.TODO(), namespacedName, &deployment); err != nil {
		return client.IgnoreNotFound(err)
	}

	if deployment.Status.AvailableReplicas == 0 {
		// deployment is scaled down, we're done
		return nil
	}

	if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas > 0 {
		log.Infof("Scaling down Rancher deployment %v", namespacedName)
		zero := int32(0)
		deployment.Spec.Replicas = &zero
		if err := c.Update(context.TODO(), &deployment); err != nil {
			return log.ErrorfNewErr("Failed to scale Rancher deployment %v to zero replicas: %v", namespacedName, err)
		}
	}

	// return RetryableError so we come back through this function again and check the replicas - repeat
	// until there are no available replicas
	log.Progressf("Waiting for Rancher deployment %v to scale down", namespacedName)
	return ctrlerrors.RetryableError{Source: ComponentName}
}

// getDynamicClient returns a dynamic k8s client
func getDynamicClient() (dynamic.Interface, error) {
	config, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return dynamicClient, nil
}

// deleteClusterRepos - temporary work around for Rancher issue 36914. On upgrade of Rancher
// the setting of useBundledSystemChart does not appear to be honored, and the downloaded
// helm charts for the previous release of Rancher are used (instead of the charts on the Rancher
// container image).
func deleteClusterRepos(log vzlog.VerrazzanoLogger) error {
	dynamicClient, err := getDynamicClientFunc()
	if err != nil {
		log.Errorf("Rancher deleteClusterRepos: Failed creating dynamic client: %v", err)
		return err
	}

	// Get the name of the default branch for the helm charts
	chartDefaultBranch, err := dynamicClient.Resource(cattleSettingsGVR).Get(context.TODO(), chartDefaultBranchName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		log.Errorf("Rancher deleteClusterRepos: Failed getting settings.management.cattle.io %s: %v", chartDefaultBranchName, err)
		return err
	}

	// Obtain the name of the default branch from the custom resource
	defaultBranch, _, err := unstructured.NestedString(chartDefaultBranch.Object, "default")
	if err != nil {
		log.Errorf("Rancher deleteClusterRepos: Failed to find default branch value in settings.management.cattle.io %s: %v", chartDefaultBranchName, err)
		return err
	}

	log.Infof("Rancher deleteClusterRepos: The default release branch is currently set to %s", defaultBranch)

	// Delete settings.management.cattle.io chart-default-branch
	err = dynamicClient.Resource(cattleSettingsGVR).Delete(context.TODO(), chartDefaultBranchName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Errorf("Rancher deleteClusterRepos: Failed deleting settings.management.cattle.io %s: %v", chartDefaultBranchName, err)
		return err
	}
	log.Infof("Rancher deleteClusterRepos: Deleted settings.management.cattle.io %s", chartDefaultBranchName)

	// List of clusterrepos to delete
	names := []string{rancherChartsClusterRepoName, rancherPartnerChartsClusterRepoName, rancherRke2ChartsClusterRepoName}
	for _, name := range names {
		err = dynamicClient.Resource(cattleClusterReposGVR).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Errorf("Rancher deleteClusterRepos: Failed deleting clusterrepos.catalog.cattle.io %s: %v", name, err)
			return err
		}
		log.Infof("Rancher deleteClusterRepos: Deleted clusterrepos.catalog.cattle.io %s", name)
	}

	return nil
}

// GetOverrides returns install overrides for a component
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.Rancher != nil {
			return effectiveCR.Spec.Components.Rancher.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.Rancher != nil {
			return effectiveCR.Spec.Components.Rancher.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

// Delete the local cluster
func DeleteLocalCluster(log vzlog.VerrazzanoLogger, c client.Client) {
	log.Once("Deleting Rancher local cluster")

	localCluster := unstructured.Unstructured{}
	localCluster.SetGroupVersionKind(GVKCluster)
	localClusterName := types.NamespacedName{Name: ClusterLocal}
	err := c.Get(context.Background(), localClusterName, &localCluster)
	if err != nil {
		log.Oncef("Failed getting local Cluster: %s", err.Error())
		return
	}

	err = c.Delete(context.Background(), &localCluster)
	if err != nil {
		log.Oncef("Failed deleting local cluster: %s", err.Error())
		return
	}

	log.Once("Successfully deleted Rancher local cluster")
}

// activateOCIDriver activates the OCI nodeDriver
func activateOCIDriver(log vzlog.VerrazzanoLogger, c client.Client) error {
	ociDriver := unstructured.Unstructured{}
	ociDriver.SetGroupVersionKind(GVKNodeDriver)
	ociDriverName := types.NamespacedName{Name: NodeDriverOCI}
	err := c.Get(context.Background(), ociDriverName, &ociDriver)
	if err != nil {
		return log.ErrorfThrottledNewErr("Failed getting OCI Driver: %s", err.Error())
	}

	ociDriverMerge := client.MergeFrom(ociDriver.DeepCopy())
	ociDriver.UnstructuredContent()["spec"].(map[string]interface{})["active"] = true
	err = c.Patch(context.Background(), &ociDriver, ociDriverMerge)
	if err != nil {
		return log.ErrorfThrottledNewErr("Failed patching OCI Driver: %s", err.Error())
	}

	return nil
}

// activateDrivers activates the oraclecontainerengine kontainerDriver
func activatOKEDriver(log vzlog.VerrazzanoLogger, c client.Client) error {
	okeDriver := unstructured.Unstructured{}
	okeDriver.SetGroupVersionKind(GVKKontainerDriver)
	okeDriverName := types.NamespacedName{Name: KontainerDriverOKE}
	err := c.Get(context.Background(), okeDriverName, &okeDriver)
	if err != nil {
		return log.ErrorfThrottledNewErr("Failed getting OKE Driver: %s", err.Error())
	}

	okeDriverMerge := client.MergeFrom(okeDriver.DeepCopy())
	okeDriver.UnstructuredContent()["spec"].(map[string]interface{})["active"] = true
	err = c.Patch(context.Background(), &okeDriver, okeDriverMerge)
	if err != nil {
		return log.ErrorfThrottledNewErr("Failed patching OKE Driver: %s", err.Error())
	}

	return nil
}

// putServerURL updates the server-url Setting
func putServerURL(log vzlog.VerrazzanoLogger, c client.Client, serverURL string) error {
	serverURLSetting := unstructured.Unstructured{}
	serverURLSetting.SetGroupVersionKind(common.GVKSetting)
	serverURLSettingName := types.NamespacedName{Name: SettingServerURL}
	err := c.Get(context.Background(), serverURLSettingName, &serverURLSetting)
	if err != nil {
		return log.ErrorfThrottledNewErr("Failed getting server-url Setting: %s", err.Error())
	}

	serverURLSetting.UnstructuredContent()["value"] = serverURL
	err = c.Update(context.Background(), &serverURLSetting)
	if err != nil {
		return log.ErrorfThrottledNewErr("Failed updating server-url Setting: %s", err.Error())
	}

	return nil
}

// configureKeycloakOIDC configures Keycloak as an OIDC provider in Rancher
func configureKeycloakOIDC(ctx spi.ComponentContext) error {
	log := ctx.Log()
	c := ctx.Client()
	keycloakAuthConfig := unstructured.Unstructured{}
	keycloakAuthConfig.SetGroupVersionKind(common.GVKAuthConfig)
	keycloakAuthConfigName := types.NamespacedName{Name: common.AuthConfigKeycloak}
	err := c.Get(context.Background(), keycloakAuthConfigName, &keycloakAuthConfig)
	if err != nil {
		return log.ErrorfThrottledNewErr("failed configuring keycloak as OIDC provider for rancher, unable to fetch keycloak authConfig: %s", err.Error())
	}

	keycloakURL, err := k8sutil.GetURLForIngress(c, "keycloak", "keycloak", "https")
	if err != nil {
		return log.ErrorfThrottledNewErr("failed configuring keycloak as OIDC provider for rancher, unable to fetch keycloak url: %s", err.Error())
	}

	rancherURL, err := k8sutil.GetURLForIngress(c, "rancher", "cattle-system", "https")
	if err != nil {
		log.Oncef("skipping configuring keycloak as OIDC provider for rancher, unable to fetch rancher url: %s", err.Error())
		return nil
	}

	clientSecret, err := keycloak.GetRancherClientSecretFromKeycloak(ctx)
	if err != nil {
		return log.ErrorfThrottledNewErr("failed configuring keycloak as OIDC provider for rancher, unable to fetch rancher client secret: %s", err.Error())
	}

	authConfig := make(map[string]interface{})
	authConfig[AuthConfigKeycloakAttributeAccessMode] = AuthConfigKeycloakAccessMode
	authConfig[AuthConfigKeycloakAttributeClientID] = AuthConfigKeycloakClientIDRancher
	authConfig[AuthConfigKeycloakAttributeGroupSearchEnabled] = true
	authConfig[AuthConfigKeycloakAttributeAuthEndpoint] = keycloakURL + AuthConfigKeycloakURLPathAuthEndPoint
	authConfig[common.AuthConfigKeycloakAttributeClientSecret] = clientSecret
	authConfig[AuthConfigKeycloakAttributeIssuer] = keycloakURL + AuthConfigKeycloakURLPathIssuer
	authConfig[AuthConfigKeycloakAttributeRancherURL] = rancherURL + AuthConfigKeycloakURLPathVerifyAuth
	authConfig[AuthConfigAttributeEnabled] = true

	return common.UpdateKeycloakOIDCAuthConfig(ctx, authConfig)
}

// createOrUpdateResource creates or updates a Rancher resource
func createOrUpdateResource(ctx spi.ComponentContext, nsn types.NamespacedName, gvk schema.GroupVersionKind, attributes map[string]interface{}) error {
	log := ctx.Log()
	c := ctx.Client()
	resource := unstructured.Unstructured{}
	resource.SetGroupVersionKind(gvk)
	resource.SetName(nsn.Name)
	resource.SetNamespace(nsn.Namespace)
	_, err := controllerutil.CreateOrUpdate(context.Background(), c, &resource, func() error {
		if len(attributes) > 0 {
			data := resource.UnstructuredContent()
			for k, v := range attributes {
				data[k] = v
			}
		}
		return nil
	})

	if err != nil {
		return log.ErrorfThrottledNewErr("failed configuring %s %s: %s", gvk.Kind, nsn.Name, err.Error())
	}

	return nil
}

// createOrUpdateRancherVerrazzanoUser creates or updates the verrazzano user in Rancher
func createOrUpdateRancherVerrazzanoUser(ctx spi.ComponentContext) error {
	log := ctx.Log()

	nsn := types.NamespacedName{Name: UserVerrazzano}

	vzUser, err := keycloak.GetVerrazzanoUserFromKeycloak(ctx)
	if err != nil {
		return log.ErrorfThrottledNewErr("failed configuring verrazzano rancher user, unable to fetch verrazzano user id from keycloak: %s", err.Error())
	}

	data := map[string]interface{}{}
	data[UserAttributeUserName] = vzUser.Username
	caser := cases.Title(language.English)
	data[UserAttributeDisplayName] = caser.String(vzUser.Username)
	data[UserAttributeDescription] = caser.String(UserVerrazzanoDescription)
	data[UserAttributePrincipalIDs] = []interface{}{UserPrincipalKeycloakPrefix + vzUser.ID, UserPrincipalLocalPrefix + UserVerrazzano}

	return createOrUpdateResource(ctx, nsn, GVKUser, data)
}

// createOrUpdateRancherVerrazzanoUserGlobalRoleBinding used to make the verrazzano user admin
func createOrUpdateRancherVerrazzanoUserGlobalRoleBinding(ctx spi.ComponentContext) error {
	nsn := types.NamespacedName{Name: GlobalRoleBindingVerrazzano}

	data := map[string]interface{}{}
	data[GlobalRoleBindingAttributeRoleName] = AdminRoleName
	data[GlobalRoleBindingAttributeUserName] = UserVerrazzano

	return createOrUpdateResource(ctx, nsn, GVKGlobalRoleBinding, data)
}

// createOrUpdateRoleTemplate creates or updates RoleTemplates used to add Keycloak groups to the Rancher cluster
func createOrUpdateRoleTemplate(ctx spi.ComponentContext, role string) error {
	log := ctx.Log()
	c := ctx.Client()

	nsn := types.NamespacedName{Name: role}

	clusterRole := &rbacv1.ClusterRole{}
	err := c.Get(context.Background(), nsn, clusterRole)
	if err != nil {
		return log.ErrorfThrottledNewErr("failed creating RoleTemplate, unable to fetch ClusterRole %s: %s", role, err.Error())
	}

	data := map[string]interface{}{}
	data[RoleTemplateAttributeBuiltin] = false
	data[RoleTemplateAttributeContext] = "cluster"
	caser := cases.Title(language.English)
	data[RoleTemplateAttributeDisplayName] = caser.String(strings.Replace(role, "-", " ", 1))
	data[RoleTemplateAttributeExternal] = true
	data[RoleTemplateAttributeHidden] = true
	if clusterRole.Rules != nil && len(clusterRole.Rules) > 0 {
		data[RoleTemplateAttributeRules] = clusterRole.Rules
	}

	return createOrUpdateResource(ctx, nsn, GVKRoleTemplate, data)
}

// createOrUpdateClusterRoleTemplateBinding creates or updates ClusterRoleTemplateBinding used to add Keycloak groups to the Rancher cluster
func createOrUpdateClusterRoleTemplateBinding(ctx spi.ComponentContext, clusterRole string, group string) error {
	name := fmt.Sprintf("crtb-%s-%s", clusterRole, group)
	nsn := types.NamespacedName{Name: name, Namespace: ClusterLocal}

	data := map[string]interface{}{}
	data[ClusterRoleTemplateBindingAttributeClusterName] = ClusterLocal
	data[ClusterRoleTemplateBindingAttributeGroupPrincipalName] = GroupPrincipalKeycloakPrefix + group
	data[ClusterRoleTemplateBindingAttributeRoleTemplateName] = clusterRole

	return createOrUpdateResource(ctx, nsn, GVKClusterRoleTemplateBinding, data)
}

// disableFirstLogin disables the verrazzano user first log in
func disableFirstLogin(ctx spi.ComponentContext) error {
	log := ctx.Log()
	c := ctx.Client()
	firstLoginSetting := unstructured.Unstructured{}
	firstLoginSetting.SetGroupVersionKind(common.GVKSetting)
	firstLoginSettingName := types.NamespacedName{Name: common.SettingFirstLogin}
	err := c.Get(context.Background(), firstLoginSettingName, &firstLoginSetting)
	if err != nil {
		return log.ErrorfThrottledNewErr("Failed getting first-login setting: %s", err.Error())
	}

	firstLoginSetting.UnstructuredContent()["value"] = "false"
	err = c.Update(context.Background(), &firstLoginSetting)
	if err != nil {
		return log.ErrorfThrottledNewErr("Failed updating first-login setting: %s", err.Error())
	}

	return nil
}

// createOrUpdateUIPlSetting creates/updates the ui-pl setting with value Verrazzano
func createOrUpdateUIPlSetting(ctx spi.ComponentContext) error {
	return createOrUpdateResource(ctx, types.NamespacedName{Name: SettingUIPL}, common.GVKSetting, map[string]interface{}{"value": SettingUIPLValueVerrazzano})
}

// createOrUpdateUILogoSetting updates the ui-logo-* settings
func createOrUpdateUILogoSetting(ctx spi.ComponentContext, settingName string, logoPath string) error {
	log := ctx.Log()
	c := ctx.Client()
	pod, err := k8sutil.GetRunningPodForLabel(c, "app=rancher", "cattle-system", log)
	if err != nil {
		return err
	}

	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return err
	}

	logoCommand := []string{"/bin/sh", "-c", fmt.Sprintf("cat %s | base64", logoPath)}
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, pod, "rancher", logoCommand)
	if err != nil {
		return log.ErrorfThrottledNewErr("Failed execing into Rancher pod %s: %v", stderr, err)
	}

	if len(stdout) == 0 {
		return log.ErrorfThrottledNewErr("Invalid empty output from Rancher pod")
	}

	return createOrUpdateResource(ctx, types.NamespacedName{Name: settingName}, common.GVKSetting, map[string]interface{}{"value": fmt.Sprintf("%s%s", SettingUILogoValueprefix, stdout)})
}

// createOrUpdateUIColorSettings creates/updates the ui-primary-color and ui-link-color settings
func createOrUpdateUIColorSettings(ctx spi.ComponentContext) error {
	err := createOrUpdateResource(ctx, types.NamespacedName{Name: SettingUIPrimaryColor}, common.GVKSetting, map[string]interface{}{"value": SettingUIPrimaryColorValue})
	if err != nil {
		return err
	}

	return createOrUpdateResource(ctx, types.NamespacedName{Name: SettingUILinkColor}, common.GVKSetting, map[string]interface{}{"value": SettingUILinkColorValue})
}
