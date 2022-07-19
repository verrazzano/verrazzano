// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"bytes"
	"context"
	"fmt"
	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/verrazzano/verrazzano/pkg/bom"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"io/fs"
	"io/ioutil"
	adminv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"os"
	controllerruntime "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"text/template"
)

var (
	// For Unit test purposes
	writeFileFunc = ioutil.WriteFile
)

func resetWriteFileFunc() {
	writeFileFunc = ioutil.WriteFile
}

const (
	deploymentName        = "jaeger-operator"
	tmpFilePrefix         = "jaeger-operator-overrides-"
	tmpSuffix             = "yaml"
	tmpFileCreatePattern  = tmpFilePrefix + "*." + tmpSuffix
	tmpFileCleanPattern   = tmpFilePrefix + ".*\\." + tmpSuffix
	jaegerSecName         = "verrazzano-jaeger-secret"
	jaegerCreateField     = "jaeger.create"
	jaegerHostName        = "jaeger"
	jaegerCertificateName = "jaeger-tls"
	openSearchURL         = "http://verrazzano-authproxy-elasticsearch.verrazzano-system.svc.cluster.local:8775"
)

// Define the Jaeger images using extraEnv key.
// We need to replace image using the real image in the bom
const extraEnvValueTemplate = `extraEnv:
  - name: "JAEGER-AGENT-IMAGE"
    value: "{{.AgentImage}}"
  - name: "JAEGER-QUERY-IMAGE"
    value: "{{.QueryImage}}"
  - name: "JAEGER-COLLECTOR-IMAGE"
    value: "{{.CollectorImage}}"
  - name: "JAEGER-INGESTER-IMAGE"
    value: "{{.IngesterImage}}"
  - name: "JAEGER-ES-INDEX-CLEANER-IMAGE"
    value: "{{.IndexCleanerImage}}"
  - name: "JAEGER-ES-ROLLOVER-IMAGE"
    value: "{{.RolloverImage}}"
  - name: "JAEGER-ALL-IN-ONE-IMAGE"
    value: "{{.AllInOneImage}}"
`

// A template to define Jaeger override
// As Jaeger Operator helm-chart does not use tpl in rendering Jaeger spec value, we can not use
// jaeger-operator-values.yaml override file to define Jaeger value referencing other values.
const jaegerValueTemplate = `jaeger:
  create: true
  spec:
    annotations:
      sidecar.istio.io/inject: "true"
      proxy.istio.io/config: '{ "holdApplicationUntilProxyStarts": true }'
    ingress:
      enabled: false
    strategy: production
    storage:
      # Jaeger Elasticsearch storage is compatible with Verrazzano OpenSearch.
      type: elasticsearch
      dependencies:
        enabled: false
      esIndexCleaner:
        enabled: true
        # Number of days to wait before deleting a record
        numberOfDays: 7
        schedule: "55 23 * * *"
      options:
        es:
          server-urls: {{.OpenSearchURL}}
          index-prefix: verrazzano-jaeger
      secretName: {{.SecretName}}
`

// imageData needed for template rendering
type imageData struct {
	AgentImage        string
	QueryImage        string
	CollectorImage    string
	IngesterImage     string
	IndexCleanerImage string
	RolloverImage     string
	AllInOneImage     string
}

// jaegerData needed for template rendering
type jaegerData struct {
	OpenSearchURL string
	SecretName    string
}

// isjaegerOperatorReady checks if the Jaeger Operator deployment is ready
func isJaegerOperatorReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      deploymentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// PreInstall implementation for the Jaeger Operator Component
func preInstall(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Jaeger Operator PreInstall dry run")
		return nil
	}

	// Create the verrazzano-monitoring namespace if not already created
	if err := ensureVerrazzanoMonitoringNamespace(ctx); err != nil {
		return err
	}

	createInstance, err := isCreateJaegerInstance(ctx)
	if err != nil {
		return err
	}
	if createInstance {
		// Create Jaeger secret with the credentials present in the verrazzano-es-internal secret
		return createJaegerSecret(ctx)
	}
	return nil
}

// AppendOverrides appends Helm value overrides for the Jaeger Operator component's Helm chart
// A go template is used to specify the Jaeger images using extraEnv key.
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Create a Bom and get the Key Value overrides
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}

	// Get jaeger-agent image
	agentImages, err := bomFile.BuildImageOverrides("jaeger-agent")
	if err != nil {
		return nil, err
	}
	if len(agentImages) != 1 {
		return nil, fmt.Errorf("component Jaeger Operator failed, expected 1 image for Jaeger Agent, found %v", len(agentImages))
	}

	// Get jaeger-collector image
	collectorImages, err := bomFile.BuildImageOverrides("jaeger-collector")
	if err != nil {
		return nil, err
	}
	if len(collectorImages) != 1 {
		return nil, fmt.Errorf("component Jaeger Operator failed, expected 1 image for Jaeger Collector, found %v", len(collectorImages))
	}

	// Get jaeger-query image
	queryImages, err := bomFile.BuildImageOverrides("jaeger-query")
	if err != nil {
		return nil, err
	}
	if len(queryImages) != 1 {
		return nil, fmt.Errorf("component Jaeger Operator failed, expected 1 image for Jaeger Query, found %v", len(queryImages))
	}

	// Get jaeger-ingester image
	ingesterImages, err := bomFile.BuildImageOverrides("jaeger-ingester")
	if err != nil {
		return nil, err
	}
	if len(ingesterImages) != 1 {
		return nil, fmt.Errorf("component Jaeger Operator failed, expected 1 image for Jaeger Ingester, found %v", len(ingesterImages))
	}

	// Get jaeger-es-index-cleaner image
	indexCleanerImages, err := bomFile.BuildImageOverrides("jaeger-es-index-cleaner")
	if err != nil {
		return nil, err
	}
	if len(indexCleanerImages) != 1 {
		return nil, fmt.Errorf("component Jaeger Operator failed, expected 1 image for Jaeger Elasticsearch Index Cleaner, found %v", len(indexCleanerImages))
	}

	// Get jaeger-es-rollover image
	rolloverImages, err := bomFile.BuildImageOverrides("jaeger-es-rollover")
	if err != nil {
		return nil, err
	}
	if len(rolloverImages) != 1 {
		return nil, fmt.Errorf("component Jaeger Operator failed, expected 1 image for Jaeger Elasticsearch Rollover, found %v", len(rolloverImages))
	}

	// Get jaeger-es-rollover image
	allInOneImages, err := bomFile.BuildImageOverrides("jaeger-all-in-one")
	if err != nil {
		return nil, err
	}
	if len(allInOneImages) != 1 {
		return nil, fmt.Errorf("component Jaeger Operator failed, expected 1 image for Jaeger AllInOne, found %v", len(allInOneImages))
	}

	// use template to populate Jaeger images
	var b bytes.Buffer
	t, err := template.New("images").Parse(extraEnvValueTemplate)
	if err != nil {
		return nil, err
	}

	// Render the template
	data := imageData{AgentImage: agentImages[0].Value, CollectorImage: collectorImages[0].Value,
		QueryImage: queryImages[0].Value, IngesterImage: ingesterImages[0].Value,
		IndexCleanerImage: indexCleanerImages[0].Value, RolloverImage: rolloverImages[0].Value,
		AllInOneImage: allInOneImages[0].Value}
	err = t.Execute(&b, data)
	if err != nil {
		return nil, err
	}

	createInstance, err := isCreateJaegerInstance(compContext)
	if err != nil {
		return nil, err
	}
	if createInstance {
		// use template to populate Jaeger spec data
		template, err := template.New("jaeger").Parse(jaegerValueTemplate)
		if err != nil {
			return nil, err
		}
		data := jaegerData{OpenSearchURL: openSearchURL, SecretName: jaegerSecName}
		err = template.Execute(&b, data)
		if err != nil {
			return nil, err
		}
	}

	// Write the overrides file to a temp dir and add a helm file override argument
	overridesFileName, err := generateOverridesFile(compContext, b.Bytes())
	if err != nil {
		return kvs, fmt.Errorf("failed generating Jaeger Operator overrides file: %v", err)
	}

	// Append any installArgs overrides
	kvs = append(kvs, bom.KeyValue{Value: overridesFileName, IsFile: true})
	return kvs, nil
}

// validateJaegerOperator checks scenarios in which the Verrazzano CR violates install verification
// due to Jaeger Operator specifications
func (c jaegerOperatorComponent) validateJaegerOperator(vz *vzapi.Verrazzano) error {
	// Validate install overrides
	if vz.Spec.Components.JaegerOperator != nil {
		if err := vzapi.ValidateInstallOverrides(vz.Spec.Components.JaegerOperator.ValueOverrides); err != nil {
			return err
		}
	}
	return nil
}

// GetOverrides returns the list of install overrides for a component
func GetOverrides(effectiveCR *vzapi.Verrazzano) []vzapi.Overrides {
	if effectiveCR.Spec.Components.JaegerOperator != nil {
		return effectiveCR.Spec.Components.JaegerOperator.ValueOverrides
	}
	return []vzapi.Overrides{}
}

func ensureVerrazzanoMonitoringNamespace(ctx spi.ComponentContext) error {
	// Create the verrazzano-monitoring namespace
	ctx.Log().Debugf("Creating namespace %s for the Jaeger Operator", ComponentNamespace)
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &namespace, func() error {
		MutateVerrazzanoMonitoringNamespace(ctx, &namespace)
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
}

// MutateVerrazzanoMonitoringNamespace modifies the given namespace for the Monitoring subcomponents
// with the appropriate labels, in one location. If the provided namespace is not the Verrazzano
// monitoring namespace, it is ignored.
func MutateVerrazzanoMonitoringNamespace(ctx spi.ComponentContext, namespace *corev1.Namespace) {
	if namespace.Name != constants.VerrazzanoMonitoringNamespace {
		return
	}
	if namespace.Labels == nil {
		namespace.Labels = map[string]string{}
	}
	namespace.Labels[globalconst.LabelVerrazzanoNamespace] = constants.VerrazzanoMonitoringNamespace

	istio := ctx.EffectiveCR().Spec.Components.Istio
	if istio != nil && istio.IsInjectionEnabled() {
		namespace.Labels[globalconst.LabelIstioInjection] = "enabled"
	}
}

func generateOverridesFile(ctx spi.ComponentContext, contents []byte) (string, error) {
	file, err := os.CreateTemp(os.TempDir(), tmpFileCreatePattern)
	if err != nil {
		return "", err
	}

	overridesFileName := file.Name()
	if err := writeFileFunc(overridesFileName, contents, fs.ModeAppend); err != nil {
		return "", err
	}
	ctx.Log().Debugf("Verrazzano jaeger-operator install overrides file %s contents: %s", overridesFileName,
		string(contents))
	return overridesFileName, nil
}

// createJaegerSecret creates a Jaeger secret for storing credentials needed to access OpenSearch.
func createJaegerSecret(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Creating secret %s required by Jaeger instance to access storage", jaegerSecName)
	esInternalSecret, err := getESInternalSecret(ctx)
	if err != nil {
		return err
	}
	if esInternalSecret.Data == nil {
		return nil
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jaegerSecName,
			Namespace: ComponentNamespace,
		},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		if _, exists := esInternalSecret.Data["username"]; exists {
			secret.Data["ES_USERNAME"] = esInternalSecret.Data["username"]
		}
		if _, exists := esInternalSecret.Data["password"]; exists {
			secret.Data["ES_PASSWORD"] = esInternalSecret.Data["password"]
		}
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s secret: %v", jaegerSecName, err)
	}
	return nil
}

// getESInternalSecret checks whether verrazzano-es-internal secret exists. Return error if the secret does not exist.
func getESInternalSecret(ctx spi.ComponentContext) (corev1.Secret, error) {
	secret := corev1.Secret{}
	if vzconfig.IsKeycloakEnabled(ctx.EffectiveCR()) {
		// Check verrazzano-es-internal Secret. return error which will cause requeue
		err := ctx.Client().Get(context.TODO(), clipkg.ObjectKey{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      globalconst.VerrazzanoESInternal,
		}, &secret)

		if err != nil {
			if errors.IsNotFound(err) {
				ctx.Log().Progressf("Component Jaeger Operator waiting for the secret %s/%s to exist",
					constants.VerrazzanoSystemNamespace, globalconst.VerrazzanoESInternal)
				return secret, ctrlerrors.RetryableError{Source: ComponentName}
			}
			ctx.Log().Errorf("Component Jaeger Operator failed to get the secret %s/%s: %v",
				constants.VerrazzanoSystemNamespace, globalconst.VerrazzanoESInternal, err)
			return secret, err
		}
		return secret, nil
	}
	return secret, nil
}

// isCreateJaegerInstance determines if the default Jaeger instance has to be created or not.
func isCreateJaegerInstance(ctx spi.ComponentContext) (bool, error) {
	if vzconfig.IsElasticsearchEnabled(ctx.EffectiveCR()) && vzconfig.IsKeycloakEnabled(ctx.EffectiveCR()) {
		// Check if jaeger instance creation is disabled in the user defined Helm overrides
		overrides, err := common.GetInstallOverridesYAML(ctx, GetOverrides(ctx.EffectiveCR()))
		if err != nil {
			return false, err
		}
		for _, override := range overrides {
			jaegerCreate, err := common.ExtractValueFromOverrideString(override, jaegerCreateField)
			if err != nil {
				return false, err
			}
			if jaegerCreate != nil && !jaegerCreate.(bool) {
				return false, nil
			}
		}
		return true, nil
	}
	return false, nil
}

// ReassociateResources updates the resources to ensure they are managed by this release/component.  The resource policy
// annotation is removed to ensure that helm manages the lifecycle of the resources (the resource policy annotation is
// added to ensure the resources are disassociated from the VZ chart which used to manage these resources)
func ReassociateResources(cli clipkg.Client) error {
	namespacedName := types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}
	name := types.NamespacedName{Name: ComponentName}
	objects := []clipkg.Object{
		&corev1.ServiceAccount{},
		&corev1.Service{},
		&appsv1.Deployment{},
	}
	noNamespaceObjects := []clipkg.Object{
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
	}
	// namespaced resources
	for _, obj := range objects {
		if _, err := common.RemoveResourcePolicyAnnotation(cli, obj, namespacedName); err != nil {
			return err
		}
	}
	// additional namespaced resources managed by this helm chart
	helmManagedResources := GetHelmManagedResources()
	for _, managedResource := range helmManagedResources {
		if _, err := common.RemoveResourcePolicyAnnotation(cli, managedResource.Obj, managedResource.NamespacedName); err != nil {
			return err
		}
	}
	// cluster resources
	for _, obj := range noNamespaceObjects {
		if _, err := common.RemoveResourcePolicyAnnotation(cli, obj, name); err != nil {
			return err
		}
	}
	return nil
}

// Verifies if User created Jaeger instance deployments exists
func doDefaultJaegerInstanceDeploymentsExists(ctx spi.ComponentContext) bool {
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
	return status.DoDeploymentsExist(ctx.Log(), client, deployments, 1, prefix)
}

// removeMutatingWebhookConfig removes the  jaeger-operator-mutating-webhook-configuration resource during the pre-upgrade
// The jaeger-operator-mutating-webhook-configuration injects the old cert and fails the webhook service handshake during the upgrade.
// On deleting, the webhook will be created by the helm and thus injects a new cert which enables a successful handshake with the service during the upgrade.
func removeMutatingWebhookConfig(ctx spi.ComponentContext) error {
	config, err := controllerruntime.GetConfig()
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
	if err != nil && !errors.IsNotFound(err) {
		return ctx.Log().ErrorfNewErr("Failed to get mutatingwebhookconfiguration %s: %v", ComponentMutatingWebhookConfigName, err)
	}
	err = kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.TODO(), ComponentMutatingWebhookConfigName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return ctx.Log().ErrorfNewErr("Failed to delete mutatingwebhookconfiguration %s: %v", ComponentMutatingWebhookConfigName, err)
	}
	return nil
}

// removeValidatingWebhookConfig removes the  jaeger-operator-validating-webhook-configuration resource during the pre-upgrade
// The jaeger-operator-validating-webhook-configuration injects the old cert and fails the webhook service handshake during the upgrade.
// On deleting, the webhook will be created by the helm and thus injects a new cert which enables a successful handshake with the service during the upgrade.
func removeValidatingWebhookConfig(ctx spi.ComponentContext) error {
	config, err := controllerruntime.GetConfig()
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
	if err != nil && !errors.IsNotFound(err) {
		return ctx.Log().ErrorfNewErr("Failed to get validatingwebhookconfiguration %s: %v", ComponentValidatingWebhookConfigName, err)
	}
	err = kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.TODO(), ComponentValidatingWebhookConfigName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return ctx.Log().ErrorfNewErr("Failed to delete validatingwebhookconfiguration %s: %v", ComponentValidatingWebhookConfigName, err)
	}
	return nil
}

// removeDeploymentAndService removes the Jaeger deployment during pre-upgrade.
// The match selector for jaeger operator deployment was changed in 1.34.1 from the previous jaeger version (1.32.0) that Verrazzano installed.
// The match selector is an immutable field so this was a workaround to avoid a failure during jaeger upgrade.
func removeDeploymentAndService(ctx spi.ComponentContext) error {
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

// removeJaegerWebhookService removes the jaeger-operator-webhook-service  during the upgrade
// After removing the mutating and validating webhook configs, the webhook service is removed and replaced by helm during the upgrade.
func removeJaegerWebhookService(ctx spi.ComponentContext) error {
	service := &corev1.Service{}
	if err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentWebhookServiceName}, service); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to get webhook service %s/%s: %v", ComponentNamespace, ComponentWebhookServiceName, err)
	}
	if err := ctx.Client().Delete(context.TODO(), service); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to delete webhook service %s/%s: %v", ComponentNamespace, ComponentWebhookServiceName, err)
	}
	return nil
}

// Jaeger yaml based installation creates jaeger-operator-serving-cert which is different from helm based installation
// But both create same secret jaeger-operator-service-cert, After jaeger is upgraded, jaeger webhook uses old secret which isn't valid, so had to be removed.
func removeOldCertAndSecret(ctx spi.ComponentContext) error {
	cert := &certv1.Certificate{}
	ctx.Log().Info("Removing old jaeger certificate if it exists %s/%s", ComponentNamespace, ComponentCertificateName)
	if err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentCertificateName}, cert); err == nil {
		if err := ctx.Client().Delete(context.TODO(), cert); err != nil {
			return ctx.Log().ErrorfNewErr("Failed to delete Jaeger cert %s/%s: %v", ComponentNamespace, ComponentCertificateName, err)
		}
	}
	secret := &corev1.Secret{}
	ctx.Log().Info("Removing old secret if it exists %s/%s", ComponentNamespace, ComponentSecretName)
	if err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentSecretName}, secret); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to get secret %s/%s: %v", ComponentNamespace, ComponentSecretName, err)
	}
	if err := ctx.Client().Delete(context.TODO(), secret); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to delete secret %s/%s: %v", ComponentNamespace, ComponentSecretName, err)
	}
	return nil
}

// GetHelmManagedResources returns a list of extra resource types and their namespaced names that are managed by the
// jaeger helm chart
func GetHelmManagedResources() []common.HelmManagedResource {
	return []common.HelmManagedResource{
		{Obj: &corev1.Service{}, NamespacedName: types.NamespacedName{Name: "jaeger-operator-metrics", Namespace: ComponentNamespace}},
		{Obj: &corev1.Service{}, NamespacedName: types.NamespacedName{Name: "jaeger-operator-webhook-service", Namespace: ComponentNamespace}},
		{Obj: &certv1.Issuer{}, NamespacedName: types.NamespacedName{Name: "jaeger-operator-selfsigned-issuer", Namespace: ComponentNamespace}},
		{Obj: &adminv1.ValidatingWebhookConfiguration{}, NamespacedName: types.NamespacedName{Name: "jaeger-operator-validating-webhook-configuration", Namespace: ComponentNamespace}},
		{Obj: &adminv1.MutatingWebhookConfiguration{}, NamespacedName: types.NamespacedName{Name: "jaeger-operator-mutating-webhook-configuration", Namespace: ComponentNamespace}},
	}
}

//Remove old Jaeger resources such as Deployment, services, certs, and webhooks
func removeOldJaegerResources(ctx spi.ComponentContext) error {
	if err := removeDeploymentAndService(ctx); err != nil {
		return err
	}
	if err := removeMutatingWebhookConfig(ctx); err != nil {
		return err
	}
	if err := removeValidatingWebhookConfig(ctx); err != nil {
		return err
	}
	if err := removeJaegerWebhookService(ctx); err != nil {
		return err
	}
	if err := removeOldCertAndSecret(ctx); err != nil {
		return err
	}
	return nil
}

// createOrUpdateJaegerIngress Creates or updates the Jaeger authproxy ingress
func createOrUpdateJaegerIngress(ctx spi.ComponentContext, namespace string) error {
	ingress := networkv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: constants.JaegerIngress, Namespace: namespace},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &ingress, func() error {
		dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed building DNS domain name: %v", err)
		}

		jaegerHostName := buildJaegerHostnameForDomain(dnsSubDomain)
		ingressClassName := vzconfig.GetIngressClassName(ctx.EffectiveCR())
		// Overwrite the existing Jaeger service definition to point to the Verrazzano authproxy
		pathType := networkv1.PathTypeImplementationSpecific
		ingRule := networkv1.IngressRule{
			Host: jaegerHostName,
			IngressRuleValue: networkv1.IngressRuleValue{
				HTTP: &networkv1.HTTPIngressRuleValue{
					Paths: []networkv1.HTTPIngressPath{
						{
							Path:     "/()(.*)",
							PathType: &pathType,
							Backend: networkv1.IngressBackend{
								Service: &networkv1.IngressServiceBackend{
									Name: constants.VerrazzanoAuthProxyServiceName,
									Port: networkv1.ServiceBackendPort{
										Number: constants.VerrazzanoAuthProxyServicePort,
									},
								},
								Resource: nil,
							},
						},
					},
				},
			},
		}
		ingress.Spec.TLS = []networkv1.IngressTLS{
			{
				Hosts:      []string{jaegerHostName},
				SecretName: "jaeger-tls",
			},
		}
		ingress.Spec.Rules = []networkv1.IngressRule{ingRule}
		ingress.Spec.IngressClassName = &ingressClassName
		if ingress.Annotations == nil {
			ingress.Annotations = make(map[string]string)
		}
		ingress.Annotations["kubernetes.io/tls-acme"] = "true"
		ingress.Annotations["nginx.ingress.kubernetes.io/proxy-body-size"] = "6M"
		ingress.Annotations["nginx.ingress.kubernetes.io/rewrite-target"] = "/$2"
		ingress.Annotations["nginx.ingress.kubernetes.io/secure-backends"] = "false"
		ingress.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTP"
		ingress.Annotations["nginx.ingress.kubernetes.io/service-upstream"] = "true"
		ingress.Annotations["nginx.ingress.kubernetes.io/upstream-vhost"] = "${service_name}.${namespace}.svc.cluster.local"
		ingress.Annotations["cert-manager.io/common-name"] = jaegerHostName
		if vzconfig.IsExternalDNSEnabled(ctx.EffectiveCR()) {
			ingressTarget := fmt.Sprintf("verrazzano-ingress.%s", dnsSubDomain)
			ingress.Annotations["external-dns.alpha.kubernetes.io/target"] = ingressTarget
			ingress.Annotations["external-dns.alpha.kubernetes.io/ttl"] = "60"
		}
		return nil
	})
	if ctrlerrors.ShouldLogKubenetesAPIError(err) {
		return ctx.Log().ErrorfNewErr("Failed create/update Jaeger ingress: %v", err)
	}
	return err
}

func buildJaegerHostnameForDomain(dnsDomain string) string {
	return fmt.Sprintf("%s.%s", jaegerHostName, dnsDomain)
}
