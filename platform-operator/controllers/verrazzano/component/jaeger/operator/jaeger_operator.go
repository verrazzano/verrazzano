// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"bytes"
	"context"
	"fmt"
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
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	controllerruntime "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"text/template"
)

var (
	// For Unit test purposes
	writeFileFunc              = ioutil.WriteFile
	getControllerRuntimeClient = getClient
	disallowedOverrides        = []string{
		"nameOverride",
		"fullnameOverride",
		"serviceAccount.name",
		"ingress.enabled",
	}
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
	if vz.Spec.Components.JaegerOperator != nil && vz.Spec.Components.JaegerOperator.Enabled != nil &&
		*vz.Spec.Components.JaegerOperator.Enabled {
		overrides := vz.Spec.Components.JaegerOperator.ValueOverrides
		if err := vzapi.ValidateInstallOverrides(overrides); err != nil {
			return err
		}

		client, err := getControllerRuntimeClient()
		if err != nil {
			return err
		}
		overrideYAMLs, err := common.GetInstallOverridesYAMLUsingClient(client, overrides, ComponentNamespace)
		if err != nil {
			return err
		}
		for _, overrideYAML := range overrideYAMLs {
			// Check if there are any Helm chart values that are not allowed to be overridden by the user
			for _, disallowedOverride := range disallowedOverrides {
				value, err := common.ExtractValueFromOverrideString(overrideYAML, disallowedOverride)
				if err != nil {
					return err
				}
				if value != nil {
					return fmt.Errorf("the Jaeger Operator Helm chart value %s cannot be overridden in the "+
						"Verrazzano CR.", disallowedOverride)
				}
			}
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
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
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

//getESInternalSecret checks whether verrazzano-es-internal secret exists. Return error if the secret does not exist.
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

// getClient returns a controller runtime client for the Verrazzano resource
func getClient() (clipkg.Client, error) {
	runtimeConfig, err := controllerruntime.GetConfig()
	if err != nil {
		return nil, err
	}
	return clipkg.New(runtimeConfig, clipkg.Options{Scheme: newScheme()})
}

// newScheme creates a new scheme that includes this package's object for use by client
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = vzapi.AddToScheme(scheme)
	_ = clientgoscheme.AddToScheme(scheme)
	return scheme
}
