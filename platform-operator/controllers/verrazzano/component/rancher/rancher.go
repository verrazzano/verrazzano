// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/verrazzano/verrazzano/platform-operator/constants"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	SettingServerURL                              = "server-url"
	SettingFirstLogin                             = "first-login"
	KontainerDriverOKE                            = "oraclecontainerengine"
	NodeDriverOCI                                 = "oci"
	ClusterLocal                                  = "local"
	AuthConfigLocal                               = "local"
	UserVerrazzano                                = "u-verrazzano"
	UserVerrazzanoDescription                     = "Verrazzano Admin"
	GlobalRoleBindingVerrazzano                   = "grb-" + UserVerrazzano
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
	UserPrincipalKeycloakPrefix                   = "keycloakoidc_user://"
	UserPrincipalLocalPrefix                      = "local://"
	UserAttributeDisplayName                      = "displayName"
	UserAttributeUserName                         = "username"
	UserAttributePrincipalIDs                     = "principalIds"
	UserAttributeDescription                      = "description"
	GlobalRoleBindingAttributeRoleName            = "globalRoleName"
	GlobalRoleBindingRoleName                     = "admin"
	GlobalRoleBindingAttributeUserName            = "userName"
)

var GVKSetting = common.GetRancherMgmtApiGVKForKind("Setting")
var GVKCluster = common.GetRancherMgmtApiGVKForKind("Cluster")
var GVKNodeDriver = common.GetRancherMgmtApiGVKForKind("NodeDriver")
var GVKKontainerDriver = common.GetRancherMgmtApiGVKForKind("KontainerDriver")
var GVKAuthConfig = common.GetRancherMgmtApiGVKForKind("AuthConfig")
var GVKUser = common.GetRancherMgmtApiGVKForKind("User")
var GVKGlobalRoleBinding = common.GetRancherMgmtApiGVKForKind("GlobalRoleBinding")

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
func GetOverrides(effectiveCR *vzapi.Verrazzano) []vzapi.Overrides {
	if effectiveCR.Spec.Components.Rancher != nil {
		return effectiveCR.Spec.Components.Rancher.ValueOverrides
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

func configureKeycloakOIDC(ctx spi.ComponentContext) error {
	log := ctx.Log()
	c := ctx.Client()

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
	return common.UpdateKeycloakOIDCAuthConfig(ctx, authConfig)
}

func createOrUpdateRancherVerrazzanoUser(ctx spi.ComponentContext) error {
	log := ctx.Log()
	c := ctx.Client()
	vzRancherUser := unstructured.Unstructured{}
	vzRancherUser.SetGroupVersionKind(GVKUser)
	vzRancherUserName := types.NamespacedName{Name: UserVerrazzano}
	err := c.Get(context.Background(), vzRancherUserName, &vzRancherUser)
	createUser := false
	if err != nil {
		if errors.IsNotFound(err) {
			createUser = true
			log.Debug("Rancher user verrazzano does not exist")
		} else {
			return log.ErrorfThrottledNewErr("failed configuring verrazzano rancher user, unable to fetch verrazzano user: %s", err.Error())
		}
	}

	vzUser, err := keycloak.GetVerrazzanoUserFromKeycloak(ctx)
	if err != nil {
		return log.ErrorfThrottledNewErr("failed configuring verrazzano rancher user, unable to fetch verrazzano user id from keycloak: %s", err.Error())
	}

	userData := vzRancherUser.UnstructuredContent()
	userData[UserAttributeUserName] = vzUser.Username
	userData[UserAttributeDisplayName] = strings.Title(vzUser.Username)
	userData[UserAttributeDescription] = strings.Title(UserVerrazzanoDescription)
	userData[UserAttributePrincipalIDs] = []interface{}{UserPrincipalKeycloakPrefix + vzUser.ID, UserPrincipalLocalPrefix + UserVerrazzano}

	if createUser {
		vzRancherUser.SetName(UserVerrazzano)
		err = c.Create(context.Background(), &vzRancherUser, &client.CreateOptions{})
	} else {
		err = c.Update(context.Background(), &vzRancherUser, &client.UpdateOptions{})
	}

	if err != nil {
		return log.ErrorfThrottledNewErr("failed configuring verrazzano rancher user: %s", err.Error())
	}

	return nil
}

func createOrUpdateRancherVerrazzanoUserGlobalRoleBinding(ctx spi.ComponentContext) error {
	log := ctx.Log()
	c := ctx.Client()
	vzRancherGlobalRoleBinding := unstructured.Unstructured{}
	vzRancherGlobalRoleBinding.SetGroupVersionKind(GVKGlobalRoleBinding)
	vzRancherGRBName := types.NamespacedName{Name: GlobalRoleBindingVerrazzano}
	err := c.Get(context.Background(), vzRancherGRBName, &vzRancherGlobalRoleBinding)
	createNew := false
	if err != nil {
		if errors.IsNotFound(err) {
			createNew = true
			log.Debug("Rancher GlobalRoleBinding for verrazzano user does not exist")
		} else {
			return log.ErrorfThrottledNewErr("failed configuring verrazzano rancher user GlobalRoleBinding, unable to fetch GlobalRoleBinding: %s", err.Error())
		}
	}

	globalRoleBindingData := vzRancherGlobalRoleBinding.UnstructuredContent()
	globalRoleBindingData[GlobalRoleBindingAttributeRoleName] = GlobalRoleBindingRoleName
	globalRoleBindingData[GlobalRoleBindingAttributeUserName] = UserVerrazzano

	if createNew {
		vzRancherGlobalRoleBinding.SetName(GlobalRoleBindingVerrazzano)
		err = c.Create(context.Background(), &vzRancherGlobalRoleBinding, &client.CreateOptions{})
	} else {
		err = c.Update(context.Background(), &vzRancherGlobalRoleBinding, &client.UpdateOptions{})
	}

	if err != nil {
		return log.ErrorfThrottledNewErr("failed configuring verrazzano rancher user GlobalRoleBinding: %s", err.Error())
	}

	return nil
}

func disableOrEnableAuthProvider(ctx spi.ComponentContext, name string, enable bool) error {
	log := ctx.Log()
	c := ctx.Client()
	authConfig := unstructured.Unstructured{}
	authConfig.SetGroupVersionKind(GVKAuthConfig)
	authConfigName := types.NamespacedName{Name: name}
	err := c.Get(context.Background(), authConfigName, &authConfig)
	if err != nil {
		return log.ErrorfThrottledNewErr("failed to set enabled to %v for authconfig %s, unable to fetch, error: %s", enable, name, err.Error())
	}

	authConfigData := authConfig.UnstructuredContent()
	authConfigData[AuthConfigAttributeEnabled] = enable
	err = c.Update(context.Background(), &authConfig, &client.UpdateOptions{})
	if err != nil {
		return log.ErrorfThrottledNewErr("failed to set enabled to %v for authconfig %s, error: %s", enable, name, err.Error())
	}

	return nil
}

func disableFirstLogin(ctx spi.ComponentContext) error {
	log := ctx.Log()
	c := ctx.Client()
	firstLoginSetting := unstructured.Unstructured{}
	firstLoginSetting.SetGroupVersionKind(GVKSetting)
	firstLoginSettingName := types.NamespacedName{Name: SettingFirstLogin}
	err := c.Get(context.Background(), firstLoginSettingName, &firstLoginSetting)
	if err != nil {
		return log.ErrorfThrottledNewErr("Failed getting first-login Setting: %s", err.Error())
	}

	firstLoginSetting.UnstructuredContent()["value"] = "false"
	err = c.Update(context.Background(), &firstLoginSetting)
	if err != nil {
		return log.ErrorfThrottledNewErr("Failed updating first-login Setting: %s", err.Error())
	}

	return nil
}
