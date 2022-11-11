// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"

	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	SettingFirstLogin          = "first-login"
)

const (
	rancherChartsClusterRepoName        = "rancher-charts"
	rancherPartnerChartsClusterRepoName = "rancher-partner-charts"
	rancherRke2ChartsClusterRepoName    = "rancher-rke2-charts"

	chartDefaultBranchName = "chart-default-branch"
)

var GVKSetting = common.GetRancherMgmtAPIGVKForKind("Setting")

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
	rancherHostname := fmt.Sprintf("%s.%s.%s", common.RancherName, vz.Spec.EnvironmentName, dnsSuffix)
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
	return status.DeploymentsAreReady(log, c, deployments, 1, prefix)
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

// GetOverrides returns install overrides for a component
func GetOverrides(effectiveCR *vzapi.Verrazzano) []vzapi.Overrides {
	if effectiveCR.Spec.Components.Rancher != nil {
		return effectiveCR.Spec.Components.Rancher.ValueOverrides
	}
	return []vzapi.Overrides{}
}
