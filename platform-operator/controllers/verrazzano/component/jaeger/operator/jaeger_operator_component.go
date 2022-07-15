// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"

	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	// ComponentName is the name of the component
	ComponentName = "jaeger-operator"
	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoMonitoringNamespace
	// ComponentJSONName is the json name of the component in the CRD
	ComponentJSONName = "jaegerOperator"
	// ChartDir is the relative directory path for Jaeger Operator chart
	ChartDir = "jaegertracing/jaeger-operator"
	//ComponentServiceName is the name of the service.
	ComponentServiceName = "jaeger-operator-metrics"
	//ComponentWebhookServiceName is the name of the webhook service.
	ComponentWebhookServiceName = "jaeger-operator-webhook-service"
	//ComponentMutatingWebhookConfigName is the name of the mutating webhook config.
	ComponentMutatingWebhookConfigName = "jaeger-operator-mutating-webhook-configuration"
	//ComponentMutatingWebhookConfigName is the name of the mutating webhook config.
	ComponentValidatingWebhookConfigName = "jaeger-operator-validating-webhook-configuration"
	//ComponentMutatingWebhookConfigName is the name of the Certificate.
	ComponentCertificateName = "jaeger-operator-serving-cert"
	//ComponentMutatingWebhookConfigName is the name of the secret.
	ComponentSecretName = "jaeger-operator-service-cert"
	//JaegerCollectorDeploymentName is the name of the Jaeger instance collector deployment.
	JaegerCollectorDeploymentName = "jaeger-operator-jaeger-collector"
	//JaegerQueryDeploymentName is the name of the Jaeger instance query deployment.
	JaegerQueryDeploymentName = "jaeger-operator-jaeger-query"
	//JaegerInstanceName is the name of the jaeger instance
	JaegerInstanceName = "jaeger-operator-jaeger"
)

type jaegerOperatorComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return jaegerOperatorComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ChartDir),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_3_0,
			ImagePullSecretKeyname:    "image.imagePullSecrets[0].name",
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "jaeger-operator-values.yaml"),
			Dependencies:              []string{certmanager.ComponentName},
			AppendOverridesFunc:       AppendOverrides,
			GetInstallOverridesFunc:   GetOverrides,
		},
	}
}

// IsEnabled returns true only if the Jaeger Operator is explicitly enabled
// in the Verrazzano CR.
func (c jaegerOperatorComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.JaegerOperator
	if comp == nil || comp.Enabled == nil {
		return false
	}
	return *comp.Enabled
}

// IsReady checks if the Jaeger Operator deployment is ready
func (c jaegerOperatorComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isJaegerOperatorReady(ctx)
	}
	return false
}

// MonitorOverrides checks whether monitoring is enabled for install overrides sources
func (c jaegerOperatorComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.JaegerOperator == nil {
		return false
	}
	if ctx.EffectiveCR().Spec.Components.JaegerOperator.MonitorChanges != nil {
		return *ctx.EffectiveCR().Spec.Components.JaegerOperator.MonitorChanges
	}
	return true
}

// PreInstall updates resources necessary for the Jaeger Operator Component installation
func (c jaegerOperatorComponent) PreInstall(ctx spi.ComponentContext) error {
	return preInstall(ctx)
}

// ValidateInstall verifies the installation of the Verrazzano object
func (c jaegerOperatorComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	return c.validateJaegerOperator(vz)
}

// ValidateUpgrade verifies the upgrade of the Verrazzano object
func (c jaegerOperatorComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	return c.validateJaegerOperator(new)
}

// PreUpgrade Jaeger component pre-upgrade processing
func (c jaegerOperatorComponent) PreUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Jaeger pre-upgrade")
	// Create the verrazzano-monitoring namespace if not already created
	if err := ensureVerrazzanoMonitoringNamespace(ctx); err != nil {
		return err
	}
	createInstance, err := isCreateJaegerInstance(ctx)
	if err != nil {
		return err
	}
	installed, err := helmcli.IsReleaseInstalled(ComponentName, ComponentNamespace)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed searching for Jaeger release: %v", err)
	}
	if !installed && DoNonDefaultJaegerInstanceDeploymentsExists(ctx) {
		return ctx.Log().ErrorfNewErr("The jaeger resource %s/%s need to be removed or moved to a different namespace before continuing the upgrade, After the upgrade by default an Jaeger instance will be created in %s namespace: %v", ComponentNamespace, JaegerInstanceName, ComponentNamespace, err)
	}
	if err := RemoveDeploymentAndService(ctx); err != nil {
		return err
	}
	if err := RemoveMutatingWebhookConfig(ctx); err != nil {
		return err
	}
	if err := RemoveValidatingWebhookConfig(ctx); err != nil {
		return err
	}
	if err := RemoveJaegerWebhookService(ctx); err != nil {
		return err
	}
	if err := RemoveOldCertAndSecret(ctx); err != nil {
		return err
	}
	if createInstance {
		// Create Jaeger secret with the credentials present in the verrazzano-es-internal secret
		return createJaegerSecret(ctx)
	}
	return nil
}

//Uprade jaegeroperator component for upgrade processing.
func (c jaegerOperatorComponent) Upgrade(ctx spi.ComponentContext) error {
	return c.HelmComponent.Install(ctx)
}

// IsInstalled checks if jaeger is installed
func (c jaegerOperatorComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, deployment)
	if errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		ctx.Log().Errorf("Failed to get %s/%s deployment: %v", ComponentNamespace, ComponentName, err)
		return false, err
	}
	return true, nil
}

//Verifies if User created Jaeger instance deployments exists
func DoNonDefaultJaegerInstanceDeploymentsExists(ctx spi.ComponentContext) bool {
	client := ctx.Client()
	deployments := []types.NamespacedName{
		{
			Name:      JaegerCollectorDeploymentName,
			Namespace: ComponentNamespace,
		},
		{
			Name:      JaegerQueryDeploymentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	//jaegerInstance := status.DoDeploymentsExist(ctx.Log(), client, deployments, 1, prefix)
	return status.DoDeploymentsExist(ctx.Log(), client, deployments, 1, prefix)
	/*	if jaegerInstance && {

		}*/
}

func RemoveMutatingWebhookConfig(ctx spi.ComponentContext) error {
	config, err := ctrl.GetConfig()
	if err != nil {
		ctx.Log().ErrorfNewErr("Failed to get kubeconfig with error: %v", err)
		return err
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		ctx.Log().ErrorfNewErr("Failed to get kubeClient with error: %v", err)
		return err
	}
	_, err = kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), ComponentMutatingWebhookConfigName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return ctx.Log().ErrorfNewErr("Failed to get mutatingwebhookconfiguration %s: %v", ComponentMutatingWebhookConfigName, err)
	}
	err = kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.TODO(), ComponentMutatingWebhookConfigName, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return ctx.Log().ErrorfNewErr("Failed to delete mutatingwebhookconfiguration %s: %v", ComponentMutatingWebhookConfigName, err)
	}
	return nil
}

func RemoveValidatingWebhookConfig(ctx spi.ComponentContext) error {
	config, err := ctrl.GetConfig()
	if err != nil {
		ctx.Log().ErrorfNewErr("Failed to get kubeconfig with error: %v", err)
		return err
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		ctx.Log().ErrorfNewErr("Failed to get kubeClient with error: %v", err)
		return err
	}
	_, err = kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), ComponentValidatingWebhookConfigName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return ctx.Log().ErrorfNewErr("Failed to get validatingwebhookconfiguration %s: %v", ComponentValidatingWebhookConfigName, err)
	}
	err = kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.TODO(), ComponentValidatingWebhookConfigName, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return ctx.Log().ErrorfNewErr("Failed to delete validatingwebhookconfiguration %s: %v", ComponentValidatingWebhookConfigName, err)
	}
	return nil
}

// removeDeploymentAndService removes the Jaeger deployment during pre-upgrade.
// The match selector for jaeger operator deployment was changed in 1.34.1 from the previous jaeger version (1.32.0) that Verrazzano installed.
// The match selector is an immutable field so this was a workaround to avoid a failure during jaeger upgrade.
func RemoveDeploymentAndService(ctx spi.ComponentContext) error {
	deployment := &appsv1.Deployment{}
	if err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, deployment); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to get deployment %s/%s: %v", ComponentNamespace, ComponentName, err)
	}
	// Remove the jaeger deployment only if the match selector is not what is expected.
	if deployment.Spec.Selector != nil && len(deployment.Spec.Selector.MatchExpressions) == 0 && len(deployment.Spec.Selector.MatchLabels) == 2 {
		instance, ok := deployment.Spec.Selector.MatchLabels["app.kubernetes.io/instance"]
		if ok && instance == ComponentName {
			name, ok := deployment.Spec.Selector.MatchLabels["app.kubernetes.io/name"]
			if ok && name == ComponentName {
				return nil
			}
		}
	}
	service := &corev1.Service{}
	if err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentServiceName}, service); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to get service %s/%s: %v", ComponentNamespace, ComponentServiceName, err)
	}
	if err := ctx.Client().Delete(context.TODO(), service); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to delete service %s/%s: %v", ComponentNamespace, ComponentServiceName, err)
	}
	if err := ctx.Client().Delete(context.TODO(), deployment); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to delete deployment %s/%s: %v", ComponentNamespace, ComponentName, err)
	}
	return nil
}

func RemoveJaegerWebhookService(ctx spi.ComponentContext) error {

	service := &corev1.Service{}
	if err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentWebhookServiceName}, service); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to get webhook service %s/%s: %v", ComponentNamespace, ComponentWebhookServiceName, err)
	}
	if err := ctx.Client().Delete(context.TODO(), service); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to delete webhook service %s/%s: %v", ComponentNamespace, ComponentWebhookServiceName, err)
	}
	return nil
}

func RemoveOldCertAndSecret(ctx spi.ComponentContext) error {
	cert := &certv1.Certificate{}
	ctx.Log().Info("Removing old jaeger certificate if it exists %s/%s: %v", ComponentNamespace, ComponentCertificateName)
	if err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentCertificateName}, cert); err == nil {
		if err := ctx.Client().Delete(context.TODO(), cert); err != nil {
			return ctx.Log().ErrorfNewErr("Failed to delete Jaeger cert %s/%s: %v", ComponentNamespace, ComponentCertificateName, err)
		}
	}
	secret := &corev1.Secret{}
	ctx.Log().Info("Removing old secret if it exists %s/%s: %v", ComponentNamespace, ComponentSecretName)
	if err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentSecretName}, secret); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to get secret %s/%s: %v", ComponentNamespace, ComponentSecretName, err)
	}
	if err := ctx.Client().Delete(context.TODO(), secret); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to delete secret %s/%s: %v", ComponentNamespace, ComponentSecretName, err)
	}
	return nil
}
