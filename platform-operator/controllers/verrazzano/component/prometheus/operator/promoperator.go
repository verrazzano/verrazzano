// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	"path"
	"strconv"

	vmoconst "github.com/verrazzano/verrazzano-monitoring-operator/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/kustomize/kyaml/sliceutil"
)

const (
	deploymentName  = "prometheus-operator-kube-p-operator"
	istioVolumeName = "istio-certs-dir"
	serviceAccount  = "cluster.local/ns/verrazzano-monitoring/sa/prometheus-operator-kube-p-prometheus"
)

// isPrometheusOperatorReady checks if the Prometheus operator deployment is ready
func isPrometheusOperatorReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      deploymentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// PreInstall implementation for the Prometheus Operator Component
func preInstall(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Prometheus Operator PreInstall dry run")
		return nil
	}

	// Create the verrazzano-monitoring namespace
	ctx.Log().Debugf("Creating namespace %s for the Prometheus Operator", ComponentNamespace)
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), prometheus.GetVerrazzanoMonitoringNamespace(), func() error {
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}

	// Create an empty secret for the additional scrape configs - this secret gets populated with scrape jobs for managed clusters
	return ensureAdditionalScrapeConfigsSecret(ctx)
}

// ensureAdditionalScrapeConfigsSecret creates an empty secret for additional scrape configurations loaded by Prometheus, if the secret
// does not already exist. Initially this secret is empty but when managed clusters are created, the federated scrape configuration
// is added to this secret.
func ensureAdditionalScrapeConfigsSecret(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Creating or updating secret %s for Prometheus additional scrape configs", vzconst.PromAdditionalScrapeConfigsSecretName)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vzconst.PromAdditionalScrapeConfigsSecretName,
			Namespace: ComponentNamespace,
		},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		if _, exists := secret.Data[vzconst.PromAdditionalScrapeConfigsSecretKey]; !exists {
			secret.Data[vzconst.PromAdditionalScrapeConfigsSecretKey] = []byte{}
		}
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s secret: %v", vzconst.PromAdditionalScrapeConfigsSecretName, err)
	}
	return nil
}

// AppendOverrides appends install overrides for the Prometheus Operator Helm chart
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Append custom images from the subcomponents in the bom
	ctx.Log().Debug("Appending the image overrides for the Prometheus Operator components")
	subcomponents := []string{"prometheus-config-reloader", "alertmanager", "prometheus"}
	kvs, err := appendCustomImageOverrides(ctx, kvs, subcomponents)
	if err != nil {
		return kvs, err
	}

	// Replace default images for subcomponents Alertmanager and Prometheus
	defaultImages := map[string]string{
		// format "subcomponentName": "helmDefaultKey"
		"alertmanager": "prometheusOperator.alertmanagerDefaultBaseImage",
		"prometheus":   "prometheusOperator.prometheusDefaultBaseImage",
	}
	kvs, err = appendDefaultImageOverrides(ctx, kvs, defaultImages)
	if err != nil {
		return kvs, err
	}

	// If the cert-manager component is enabled, use it for webhook certificates, otherwise Prometheus Operator
	// will use the kube-webhook-certgen image
	kvs = append(kvs, bom.KeyValue{
		Key:   "prometheusOperator.admissionWebhooks.certManager.enabled",
		Value: strconv.FormatBool(vzconfig.IsCertManagerEnabled(ctx.EffectiveCR())),
	})

	// If we specify a storage or the prod is used, create a PVC for Prometheus
	resourceRequest, err := common.FindStorageOverride(ctx.EffectiveCR())
	if err != nil {
		return kvs, err
	}
	if resourceRequest != nil {
		storage := resourceRequest.Storage
		memory := resourceRequest.Memory
		kvs = append(kvs, []bom.KeyValue{
			{
				Key:   "prometheusOperator.prometheusSpec.storageSpec.volumeClaimTemplate.spec.storageClassName.resources.requests.storage",
				Value: storage,
			},
			{
				Key:   "prometheusOperator.prometheusSpec.storageSpec.volumeClaimTemplate.spec.storageClassName.resources.requests.memory",
				Value: memory,
			},
		}...)
	}

	// Append the Istio Annotations for Prometheus
	kvs, err = appendIstioOverrides("prometheus.prometheusSpec.podMetadata.annotations",
		"prometheus.prometheusSpec.volumeMounts",
		"prometheus.prometheusSpec.volumes",
		kvs)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed applying the Istio Overrides for Prometheus")
	}

	// Disable HTTP2 to allow mTLS communication with the application Istio sidecars
	kvs = append(kvs, []bom.KeyValue{
		{Key: "prometheus.prometheusSpec.containers[0].name", Value: "prometheus"},
		{Key: "prometheus.prometheusSpec.containers[0].env[0].name", Value: "PROMETHEUS_COMMON_DISABLE_HTTP2"},
		{Key: "prometheus.prometheusSpec.containers[0].env[0].value", Value: `"1"`},
	}...)

	kvs, err = appendAdditionalVolumeOverrides(ctx,
		"prometheus.prometheusSpec.volumeMounts",
		"prometheus.prometheusSpec.volumes",
		kvs)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed applying additional volume overrides for Prometheus")
	}

	// Add a label to Prometheus Operator resources to distinguish Verrazzano resources
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("commonLabels.%s", constants.VerrazzanoComponentLabelKey), Value: ComponentName})

	// Add label to the Prometheus Operator pod to avoid a sidecar injection
	kvs = append(kvs, bom.KeyValue{Key: `prometheusOperator.podAnnotations.sidecar\.istio\.io/inject`, Value: `"false"`})

	return kvs, nil
}

// appendCustomImageOverrides takes a list of subcomponent image names and appends it to the given Helm overrides
func appendCustomImageOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue, subcomponents []string) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, ctx.Log().ErrorNewErr("Failed to get the bom file for the Prometheus Operator image overrides: ", err)
	}

	for _, subcomponent := range subcomponents {
		imageOverrides, err := bomFile.BuildImageOverrides(subcomponent)
		if err != nil {
			return kvs, ctx.Log().ErrorfNewErr("Failed to build the Prometheus Operator image overrides for subcomponent %s: ", subcomponent, err)
		}
		kvs = append(kvs, imageOverrides...)
	}

	return kvs, nil
}

func appendDefaultImageOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue, subcomponents map[string]string) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, ctx.Log().ErrorNewErr("Failed to get the bom file for the Prometheus Operator image overrides: ", err)
	}

	for subcomponent, helmKey := range subcomponents {
		images, err := bomFile.GetImageNameList(subcomponent)
		if err != nil {
			return kvs, ctx.Log().ErrorfNewErr("Failed to get the image for subcomponent %s from the bom: ", subcomponent, err)
		}
		if len(images) > 0 {
			kvs = append(kvs, bom.KeyValue{Key: helmKey, Value: images[0]})
		}
	}

	return kvs, nil
}

// validatePrometheusOperator checks scenarios in which the Verrazzano CR violates install verification due to Prometheus Operator specifications
func (c prometheusComponent) validatePrometheusOperator(vz *vzapi.Verrazzano) error {
	// Validate if Prometheus is enabled, Prometheus Operator should be enabled
	if !c.IsEnabled(vz) && vzconfig.IsPrometheusEnabled(vz) {
		return fmt.Errorf("Prometheus cannot be enabled if the Prometheus Operator is disabled")
	}
	// Validate install overrides
	if vz.Spec.Components.PrometheusOperator != nil {
		if err := vzapi.ValidateInstallOverrides(vz.Spec.Components.PrometheusOperator.ValueOverrides); err != nil {
			return err
		}
	}
	return nil
}

// appendIstioOverrides appends Istio annotations necessary for Prometheus in Istio
// Istio is required on the Prometheus for mTLS between it and Verrazzano applications
func appendIstioOverrides(annotationsKey, volumeMountKey, volumeKey string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Istio annotations that will copy the volume mount for the Istio certs to the envoy sidecar
	// The last annotation allows envoy to intercept only requests from the Keycloak Service IP
	annotations := map[string]string{
		`proxy\.istio\.io/config`:                             `{"proxyMetadata":{ "OUTPUT_CERTS": "/etc/istio-output-certs"}}`,
		`sidecar\.istio\.io/userVolumeMount`:                  `[{"name": "istio-certs-dir", "mountPath": "/etc/istio-output-certs"}]`,
		`traffic\.sidecar\.istio\.io/excludeOutboundIPRanges`: "0.0.0.0/0",
	}

	for key, value := range annotations {
		kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s.%s", annotationsKey, key), Value: value})
	}

	// Volume mount on the Prometheus container to mount the Istio-generated certificates
	vm := corev1.VolumeMount{
		Name:      istioVolumeName,
		MountPath: vmoconst.IstioCertsMountPath,
	}
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[0].name", volumeMountKey), Value: vm.Name})
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[0].mountPath", volumeMountKey), Value: vm.MountPath})

	// Volume annotation to enable an in-memory location for Istio to place and serve certificates
	vol := corev1.Volume{
		Name: istioVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
	}
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[0].name", volumeKey), Value: vol.Name})
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[0].emptyDir.medium", volumeKey), Value: string(vol.VolumeSource.EmptyDir.Medium)})

	return kvs, nil
}

// GetOverrides appends Helm value overrides for the Prometheus Operator Helm chart
func GetOverrides(effectiveCR *vzapi.Verrazzano) []vzapi.Overrides {
	if effectiveCR.Spec.Components.PrometheusOperator != nil {
		return effectiveCR.Spec.Components.PrometheusOperator.ValueOverrides
	}
	return []vzapi.Overrides{}
}

// appendAdditionalVolumeOverrides adds a volume and volume mount so we can mount managed cluster TLS certs from a secret in the Prometheus pod.
// Initially the secret does not exist. When managed clusters are created, the secret is created and Prometheus TLS certs for the managed
// clusters are added to the secret.
func appendAdditionalVolumeOverrides(ctx spi.ComponentContext, volumeMountKey, volumeKey string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[1].name", volumeMountKey), Value: "managed-cluster-ca-certs"})
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[1].mountPath", volumeMountKey), Value: "/etc/prometheus/managed-cluster-ca-certs"})
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[1].readOnly", volumeMountKey), Value: "true"})

	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[1].name", volumeKey), Value: "managed-cluster-ca-certs"})
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[1].secret.secretName", volumeKey), Value: constants.PromManagedClusterCACertsSecretName})
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[1].secret.optional", volumeKey), Value: "true"})

	return kvs, nil
}

// applySystemMonitors applies templatized PodMonitor and ServiceMonitor custom resources for Verrazzano system
// components to the cluster
func applySystemMonitors(ctx spi.ComponentContext) error {
	// create template key/value map
	args := make(map[string]interface{})
	args["systemNamespace"] = constants.VerrazzanoSystemNamespace
	args["monitoringNamespace"] = constants.VerrazzanoMonitoringNamespace
	args["nginxNamespace"] = constants.IngressNginxNamespace
	args["istioNamespace"] = constants.IstioSystemNamespace

	// substitute template values to all files in the directory and apply the resulting YAML
	dir := path.Join(config.GetThirdPartyManifestsDir(), "prometheus-operator")
	yamlApplier := k8sutil.NewYAMLApplier(ctx.Client(), "")
	return yamlApplier.ApplyDT(dir, args)
}

func updateApplicationAuthorizationPolicies(ctx spi.ComponentContext) error {
	// Get the Application namespaces by filtering the label verrazzano-managed=true
	nsList := corev1.NamespaceList{}
	err := ctx.Client().List(context.TODO(), &nsList, &client.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set{vzconst.VerrazzanoManagedLabelKey: "true"})})
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to list namespaces with the label %s=true: %v", vzconst.VerrazzanoManagedLabelKey, err)
	}

	// For each namespace, if an authorization policy exists, add the prometheus operator service account as a principal
	for _, ns := range nsList.Items {
		authPolicyList := istioclisec.AuthorizationPolicyList{}
		err = ctx.Client().List(context.TODO(), &authPolicyList, &client.ListOptions{Namespace: ns.Name})
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed to list Authorization Policies in namespace %s: %v", ns.Name, err)
		}
		// Parse the authorization policy list for the Verrazzano Istio label and apply the service account to the first rule
		for i := range authPolicyList.Items {
			authPolicy := authPolicyList.Items[i]
			if _, ok := authPolicy.Labels[constants.IstioAppLabel]; !ok {
				continue
			}
			_, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client(), &authPolicy, func() error {
				rules := authPolicy.Spec.Rules
				if len(rules) <= 0 || rules[0] == nil {
					return nil
				}
				targetRule := rules[0]
				if len(targetRule.From) <= 0 || targetRule.From[0] == nil {
					return nil
				}
				targetFrom := targetRule.From[0]
				if targetFrom.Source == nil {
					return nil
				}
				// Update the object principal with the Prometheus Operator service account if not found
				if !sliceutil.Contains(targetFrom.Source.Principals, serviceAccount) {
					authPolicy.Spec.Rules[0].From[0].Source.Principals = append(targetFrom.Source.Principals, serviceAccount)
				}
				return nil
			})
			if err != nil {
				return ctx.Log().ErrorfNewErr("Failed to update the Authorization Policy %s in namespace %s: %v", authPolicy.Name, ns.Name, err)
			}
		}
	}
	return nil
}
