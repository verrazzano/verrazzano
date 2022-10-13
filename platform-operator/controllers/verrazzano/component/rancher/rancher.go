// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"bufio"
	"context"
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"reflect"
	"strings"

	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/verrazzano/verrazzano/pkg/k8s/status"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// checkRancherUpgradeFailureSig is a function needed for unit test override
type checkRancherUpgradeFailureSig func(c client.Client, log vzlog.VerrazzanoLogger) (err error)

// checkRancherUpgradeFailureFunc is the default checkRancherUpgradeFailure function
var checkRancherUpgradeFailureFunc checkRancherUpgradeFailureSig = checkRancherUpgradeFailure

// fakeCheckRancherUpgradeFailure is the fake checkRancherUpgradeFailure function needed for unit testing
func fakeCheckRancherUpgradeFailure(_ client.Client, _ vzlog.VerrazzanoLogger) (err error) {
	return nil
}

func SetFakeCheckRancherUpgradeFailureFunc() {
	checkRancherUpgradeFailureFunc = fakeCheckRancherUpgradeFailure
}

func SetDefaultCheckRancherUpgradeFailureFunc() {
	checkRancherUpgradeFailureFunc = checkRancherUpgradeFailure
}

// Constants for Kubernetes resource names
const (
	// note: VZ-5241 In Rancher 2.6.3 the agent was moved from cattle-fleet-system ns
	// to a new cattle-fleet-local-system ns, the rancher-operator-system ns was
	// removed, and the rancher-operator is no longer deployed
	FleetSystemNamespace      = "cattle-fleet-system"
	FleetLocalSystemNamespace = "cattle-fleet-local-system"
	defaultSecretNamespace    = "cert-manager"
	namespaceLabelKey         = "verrazzano.io/namespace"
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
	SettingFirstLogin              = "first-login"
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

var GVKSetting = common.GetRancherMgmtAPIGVKForKind("Setting")
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

	// Temporary work around for Rancher issue 36914
	err := checkRancherUpgradeFailureFunc(c, log)
	if err != nil {
		log.ErrorfThrottled("Error checking Rancher pod logs: %s", err.Error())
		return false
	}

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
	return status.DeploymentsAreReady(log, c, deployments, 1, prefix)
}

// checkRancherUpgradeFailure - temporary work around for Rancher issue 36914. During an upgrade, the Rancher pods
// are recycled.  When the leader pod is restarted, it is possible that a Rancher 2.5.9 pod could
// acquire leader and recreate the downloaded helm charts it requires.
//
// If one of the Rancher pods is failing to find the rancher-webhook, recycle that pod.
func checkRancherUpgradeFailure(c client.Client, log vzlog.VerrazzanoLogger) error {
	ctx := context.TODO()

	// Get the Rancher pods
	podList := &corev1.PodList{}
	err := c.List(ctx, podList, client.InNamespace(ComponentNamespace), client.MatchingLabels{"app": "rancher"})
	if err != nil {
		return err
	}
	if len(podList.Items) == 0 {
		return nil
	}

	config, err := ctrl.GetConfig()
	if err != nil {
		return err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// Check the logs of each pod
	podsRestarted := false
	for i, pod := range podList.Items {
		// Skip pods that are already being deleted
		if pod.DeletionTimestamp != nil {
			continue
		}

		// Skip pods that are not ready, they will get checked again in another call to isReady.
		if !isPodReady(pod) {
			continue
		}

		// Get the pod log stream
		logStream, err := clientSet.CoreV1().Pods(ComponentNamespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: "rancher"}).Stream(ctx)
		if err != nil {
			return err
		}
		defer logStream.Close()

		// Search the stream for the expected text
		restartPod := false
		scanner := bufio.NewScanner(logStream)
		for scanner.Scan() {
			token := scanner.Text()
			if strings.Contains(token, "[ERROR] available chart version") {
				log.Infof("Rancher IsReady: Failed to find system chart for pod %s: %s", pod.Name, token)
				restartPod = true
				break
			}
		}

		// If the pod is failing to find the system chart for rancher-webhook, the wrong helm charts are
		// being used by the Rancher pod. Restart the pod. This will cause another Rancher pod to become the leader,
		// and, if needed, will recreate the custom resources related to the helm charts.
		if restartPod {
			// Delete custom resources containing helm charts to use
			err := deleteClusterRepos(log)
			if err != nil {
				return err
			}
			log.Infof("Rancher IsReady: Restarting pod %s", pod.Name)
			err = c.Delete(ctx, &podList.Items[i])
			if err != nil {
				return err
			}
			podsRestarted = true
		}
	}

	// If any pods were restarted, return an error so that the IsReady check will not continue
	// any further.  Checks will resume again after the pod is ready again.
	if podsRestarted {
		return fmt.Errorf("Rancher IsReady: pods were restarted, waiting for them to be ready again")
	}

	return nil
}

// deleteClusterRepos - temporary work around for Rancher issue 36914. On upgrade of Rancher
// the setting of useBundledSystemChart does not appear to be honored, and the downloaded
// helm charts for the previous release of Rancher are used (instead of the charts on the Rancher
// container image).
func deleteClusterRepos(log vzlog.VerrazzanoLogger) error {

	config, err := ctrl.GetConfig()
	if err != nil {
		log.Debugf("Rancher IsReady: Failed getting config: %v", err)
		return err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Debugf("Rancher IsReady: Failed creating dynamic client: %v", err)
		return err
	}

	// Configure the GVR
	gvr := schema.GroupVersionResource{
		Group:    "management.cattle.io",
		Version:  "v3",
		Resource: "settings",
	}

	// Get the name of the default branch for the helm charts
	name := "chart-default-branch"
	chartDefaultBranch, err := dynamicClient.Resource(gvr).Get(context.TODO(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		log.Debugf("Rancher IsReady: Failed getting settings.management.cattle.io %s: %v", name, err)
		return err
	}

	// Obtain the name of the default branch from the custom resource
	defaultBranch, _, err := unstructured.NestedString(chartDefaultBranch.Object, "default")
	if err != nil {
		log.Debugf("Rancher IsReady: Failed to find default branch value in settings.management.cattle.io %s: %v", name, err)
		return err
	}

	log.Infof("Rancher IsReady: The default release branch is currently set to %s", defaultBranch)

	// Delete settings.management.cattle.io chart-default-branch
	err = dynamicClient.Resource(gvr).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Debugf("Rancher IsReady: Failed deleting settings.management.cattle.io %s: %v", name, err)
		return err
	}
	log.Infof("Rancher IsReady: Deleted settings.management.cattle.io %s", name)

	// Reconfigure the GVR
	gvr = schema.GroupVersionResource{
		Group:    "catalog.cattle.io",
		Version:  "v1",
		Resource: "clusterrepos",
	}

	// List of clusterrepos to delete
	names := []string{"rancher-charts", "rancher-rke2-charts", "rancher-partner-charts"}
	for _, name := range names {
		err = dynamicClient.Resource(gvr).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Debugf("Rancher IsReady: Failed deleting clusterrepos.catalog.cattle.io %s: %v", name, err)
			return err
		}
		log.Infof("Rancher IsReady: Deleted clusterrepos.catalog.cattle.io %s", name)
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
	serverURLSetting.SetGroupVersionKind(GVKSetting)
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
	existingUser, err := checkRancherVerrazzanoUser(ctx, vzUser)
	if err != nil {
		return log.ErrorfThrottledNewErr("failed to check if verrazzano rancher user exists: %s", err.Error())
	} else if existingUser != "" {
		return log.ErrorfThrottledNewErr("Failed to create rancher user %s as another rancher user %s exists that "+
			"is mapped to verrazzano user id from keycloak.", UserVerrazzano, existingUser)
	}
	data := map[string]interface{}{}
	data[UserAttributeUserName] = vzUser.Username
	caser := cases.Title(language.English)
	data[UserAttributeDisplayName] = caser.String(vzUser.Username)
	data[UserAttributeDescription] = caser.String(UserVerrazzanoDescription)
	data[UserAttributePrincipalIDs] = []interface{}{UserPrincipalKeycloakPrefix + vzUser.ID, UserPrincipalLocalPrefix + UserVerrazzano}

	return createOrUpdateResource(ctx, nsn, GVKUser, data)
}

// checkRancherVerrazzanoUser checks whether any rancher user exists other than u-verrazzano mapped to the ID of  key-clock user verrazzano
func checkRancherVerrazzanoUser(ctx spi.ComponentContext, vzUser *keycloak.KeycloakUser) (string, error) {
	c := ctx.Client()
	resource := unstructured.UnstructuredList{}
	resource.SetGroupVersionKind(GVKUser)
	err := c.List(context.TODO(), &resource, &client.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, user := range resource.Items {
		if user.GetName() == UserVerrazzano {
			continue
		}
		data := user.UnstructuredContent()
		principleIDs := data[UserAttributePrincipalIDs]
		switch reflect.TypeOf(principleIDs).Kind() {
		case reflect.Slice:
			principleID := reflect.ValueOf(principleIDs)
			for i := 0; i < principleID.Len(); i++ {
				if strings.Contains(principleID.Index(i).String(), UserPrincipalKeycloakPrefix+vzUser.ID) {
					return user.GetName(), nil
				}
			}
		}
	}
	return "", nil
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
	firstLoginSetting.SetGroupVersionKind(GVKSetting)
	firstLoginSettingName := types.NamespacedName{Name: SettingFirstLogin}
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
	return createOrUpdateResource(ctx, types.NamespacedName{Name: SettingUIPL}, GVKSetting, map[string]interface{}{"value": SettingUIPLValueVerrazzano})
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

	return createOrUpdateResource(ctx, types.NamespacedName{Name: settingName}, GVKSetting, map[string]interface{}{"value": fmt.Sprintf("%s%s", SettingUILogoValueprefix, stdout)})
}

// createOrUpdateUIColorSettings creates/updates the ui-primary-color and ui-link-color settings
func createOrUpdateUIColorSettings(ctx spi.ComponentContext) error {
	err := createOrUpdateResource(ctx, types.NamespacedName{Name: SettingUIPrimaryColor}, GVKSetting, map[string]interface{}{"value": SettingUIPrimaryColorValue})
	if err != nil {
		return err
	}

	return createOrUpdateResource(ctx, types.NamespacedName{Name: SettingUILinkColor}, GVKSetting, map[string]interface{}{"value": SettingUILinkColorValue})
}
