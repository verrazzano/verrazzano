// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/workloadselector"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	MetricsAnnotation                   = "app.verrazzano.io/metrics"
	MetricsBindingGeneratorWorkloadPath = "/metrics-binding-generator-workload"
)

// WorkloadWebhook type for the mutating webhook
type WorkloadWebhook struct {
	client.Client
	Decoder    *admission.Decoder
	KubeClient kubernetes.Interface
}

// Handle - handler for the mutating webhook
func (a *WorkloadWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	log := zap.S().With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceName, req.Name, vzlog.FieldWebhook, "metrics-binding-generator-workload")
	log.Debugf("group: %s, version: %s, kind: %s", req.Kind.Group, req.Kind.Version, req.Kind.Kind)
	return a.handleWorkloadResource(ctx, req, log)
}

// InjectDecoder injects the decoder.
func (a *WorkloadWebhook) InjectDecoder(d *admission.Decoder) error {
	a.Decoder = d
	return nil
}

// handleWorkloadResource decodes the admission request for a workload resource into an unstructured
// and then processes workload resource
func (a *WorkloadWebhook) handleWorkloadResource(ctx context.Context, req admission.Request, log *zap.SugaredLogger) admission.Response {
	unst := &unstructured.Unstructured{}
	if err := a.Decoder.Decode(req, unst); err != nil {
		log.Errorf("Failed decoding object in admission request: %v", err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Do not handle any workload resources that have owner references.
	// NOTE: this will be revisited.
	if len(unst.GetOwnerReferences()) != 0 {
		return admission.Allowed(constants.StatusReasonSuccess)
	}

	// Handle legacy metrics annotations only for _existing_ workloads i.e. if a MetricsBinding
	// already exists
	var existingMetricsBinding *vzapp.MetricsBinding
	var err error
	if existingMetricsBinding, err = a.GetLegacyMetricsBinding(ctx, unst); err != nil {
		log.Errorf("Failed trying to retrieve legacy MetricsBinding for %s workload %s/%s: %v", unst.GetKind(), unst.GetNamespace(), unst.GetName(), err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if existingMetricsBinding == nil {
		// No MetricsBinding exists to be migrated - this is likely a newer app that has not been
		// processed by Verrazzano versions earlier than 1.4
		return admission.Allowed(constants.StatusReasonSuccess)
	}

	// If we got here, this is a pre-Verrazzano 1.4 application - process the annotations and
	// update the existing metrics binding accordingly as before
	// If "none" is specified on workload for annotation "app.verrazzano.io/metrics" then this workload has opted out of metrics.
	if metricsTemplateAnnotation, ok := unst.GetAnnotations()[MetricsAnnotation]; ok {
		if strings.ToLower(metricsTemplateAnnotation) == "none" {
			log.Infof("%s is set to none in the workload - opting out of metrics", MetricsAnnotation)
			return admission.Allowed(constants.StatusReasonSuccess)
		}
	}

	// Get the workload Namespace for annotation processing
	workloadNamespace := &corev1.Namespace{}
	if err = a.Client.Get(context.TODO(), types.NamespacedName{Name: unst.GetNamespace()}, workloadNamespace); err != nil {
		log.Errorf("Failed getting workload namespace %s: %v", unst.GetNamespace(), err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// If "none" is specified on namespace for annotation "app.verrazzano.io/metrics" then this namespace has opted out of metrics.
	if metricsTemplateAnnotation, ok := workloadNamespace.GetAnnotations()[MetricsAnnotation]; ok {
		if strings.ToLower(metricsTemplateAnnotation) == "none" {
			log.Infof("%s is set to none in the namespace - opting out of metrics", MetricsAnnotation)
			return admission.Allowed(constants.StatusReasonSuccess)
		}
	}

	// Get the metrics template from annotation or workload selector
	metricsTemplate, err := a.getMetricsTemplate(ctx, unst, workloadNamespace, log)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Metrics template handling - update the metrics binding as needed

	// Workload resource specifies a valid metrics template or we found one above
	// We use that metrics template to update the existing metrics binding resource. We won't
	// create new MetricsBindings as of Verrazzano 1.4 but we will honor settings for existing apps
	if err = a.updateMetricBinding(ctx, unst, metricsTemplate, existingMetricsBinding, log); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	marshaledWorkloadResource, err := json.Marshal(unst)
	if err != nil {
		log.Errorf("Failed marshalling workload resource: %v", err)
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledWorkloadResource)
}

// getMetricsTemplate processes the app.verrazzano.io/metrics annotation and gets the metrics
// template, if specified. Otherwise it finds the matching metrics template based on workload selector
func (a *WorkloadWebhook) getMetricsTemplate(ctx context.Context, unst *unstructured.Unstructured, workloadNamespace *corev1.Namespace, log *zap.SugaredLogger) (*vzapp.MetricsTemplate, error) {
	metricsTemplate, err := a.processMetricsAnnotation(unst, workloadNamespace, log)
	if err != nil {
		return nil, err
	}

	if metricsTemplate == nil {
		// Workload resource does not specify a metrics template.
		// Look for a matching metrics template workload whose workload selector matches.
		// First, check the namespace of the workload resource and then check the verrazzano-system namespace
		// NOTE: use the first match for now
		// var metricsTemplate *vzapp.MetricsTemplate
		metricsTemplate, err = a.findMatchingTemplate(ctx, unst, unst.GetNamespace(), log)
		if err != nil {
			return nil, err
		}
		if metricsTemplate == nil {
			template, err := a.findMatchingTemplate(ctx, unst, constants.VerrazzanoSystemNamespace, log)
			if err != nil {
				return nil, err
			}
			metricsTemplate = template
		}
	}
	return metricsTemplate, nil
}

// GetLegacyMetricsBinding returns the existing MetricsBinding (legacy resource) for the given
// workload - nil if it does not exist.
func (a *WorkloadWebhook) GetLegacyMetricsBinding(ctx context.Context, unst *unstructured.Unstructured) (*vzapp.MetricsBinding, error) {
	metricsBindingName := generateMetricsBindingName(unst.GetName(), unst.GetAPIVersion(), unst.GetKind())
	metricsBindingKey := types.NamespacedName{Namespace: unst.GetNamespace(), Name: metricsBindingName}
	metricsBinding := vzapp.MetricsBinding{}
	err := a.Client.Get(ctx, metricsBindingKey, &metricsBinding)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	return &metricsBinding, err
}

// processMetricsAnnotation checks the workload resource for the "app.verrazzano.io/metrics" annotation and returns the
// metrics template referenced in the annotation
func (a *WorkloadWebhook) processMetricsAnnotation(unst *unstructured.Unstructured, workloadNamespace *corev1.Namespace, log *zap.SugaredLogger) (*vzapp.MetricsTemplate, error) {
	// Check workload, then namespace for annotation
	metricsTemplate, ok := unst.GetAnnotations()[MetricsAnnotation]
	if !ok {
		metricsTemplate, ok = workloadNamespace.GetAnnotations()[MetricsAnnotation]
		if !ok {
			return nil, nil
		}
	}

	// Look for the metrics template in the namespace of the workload resource
	template := &vzapp.MetricsTemplate{}
	namespacedName := types.NamespacedName{Namespace: unst.GetNamespace(), Name: metricsTemplate}
	err := a.Client.Get(context.TODO(), namespacedName, template)
	if err != nil {
		// If we don't find the metrics template in the namespace of the workload resource then
		// look in the verrazzano-system namespace
		if apierrors.IsNotFound(err) {
			namespacedName := types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: metricsTemplate}
			err := a.Client.Get(context.TODO(), namespacedName, template)
			if err != nil {
				log.Errorf("Failed getting metrics template %s/%s: %v", constants.VerrazzanoSystemNamespace, metricsTemplate, err)
				return nil, err
			}
			log.Infof("Found matching metrics template %s/%s", constants.VerrazzanoSystemNamespace, metricsTemplate)
			return template, nil
		}

		log.Errorf("Failed getting metrics template %s/%s: %v", unst.GetNamespace(), metricsTemplate, err)
		return nil, err
	}

	log.Infof("Found matching metrics template %s/%s", unst.GetNamespace(), metricsTemplate)
	return template, nil
}

// updateMetricBinding updates an existing metricsBinding resource and
// adds the apps.verrazzano.io/workload label to the workload resource
func (a *WorkloadWebhook) updateMetricBinding(ctx context.Context, unst *unstructured.Unstructured, template *vzapp.MetricsTemplate, metricsBinding *vzapp.MetricsBinding, log *zap.SugaredLogger) error {
	if template == nil {
		// nothing to update
		return nil
	}
	// When the Prometheus target config map was not specified in the metrics template then there is nothing to do.
	if reflect.DeepEqual(template.Spec.PrometheusConfig.TargetConfigMap, vzapp.TargetConfigMap{}) {
		log.Infof("Prometheus target config map %s/%s not specified", template.Namespace, template.Name)
		return nil
	}

	// Only look for the config map if it's not the legacy one. The legacy VMI config map will no longer exist, and be replaced
	// with the additional scrape configs secret in the MetricsBinding, so don't look for it.
	if !isLegacyVmiPrometheusConfigMapName(vzapp.NamespaceName{
		Namespace: template.Spec.PrometheusConfig.TargetConfigMap.Namespace, Name: template.Spec.PrometheusConfig.TargetConfigMap.Name}) {
		_, err := a.KubeClient.CoreV1().ConfigMaps(template.Spec.PrometheusConfig.TargetConfigMap.Namespace).Get(ctx, template.Spec.PrometheusConfig.TargetConfigMap.Name, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Failed getting Prometheus target config map %s/%s: %v", template.Namespace, template.Name, err)
			return err
		}
	}

	err := a.mutateMetricsBinding(metricsBinding, template, unst)
	if err != nil {
		log.Errorf("Failed mutating the metricsBinding resource: %v", err)
		return err
	}

	err = a.Client.Update(ctx, metricsBinding)
	if err != nil {
		log.Errorf("Failed updating the metricsBinding resource: %v", err)
		return err
	}

	// Set the app.verrazzano.io/workload to identify the Prometheus config scrape target
	labels := unst.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[constants.MetricsWorkloadLabel] = metricsBinding.GetName()
	unst.SetLabels(labels)

	return nil
}

// mutateMetricsBinding mutates a metricsBinding resource based on the metrics template provided
func (a *WorkloadWebhook) mutateMetricsBinding(metricsBinding *vzapp.MetricsBinding, template *vzapp.MetricsTemplate, unst *unstructured.Unstructured) error {
	metricsBinding.Spec.MetricsTemplate.Namespace = template.Namespace
	metricsBinding.Spec.MetricsTemplate.Name = template.Name
	metricsBinding.Spec.PrometheusConfigMap.Namespace = template.Spec.PrometheusConfig.TargetConfigMap.Namespace
	metricsBinding.Spec.PrometheusConfigMap.Name = template.Spec.PrometheusConfig.TargetConfigMap.Name
	metricsBinding.Spec.Workload.Name = unst.GetName()
	metricsBinding.Spec.Workload.TypeMeta = metav1.TypeMeta{APIVersion: unst.GetAPIVersion(), Kind: unst.GetKind()}

	// If the config map specified is the legacy VMI prometheus config map, modify it to use
	// the additionalScrapeConfigs config map for the Prometheus Operator
	if isLegacyVmiPrometheusConfigMapName(metricsBinding.Spec.PrometheusConfigMap) {
		metricsBinding.Spec.PrometheusConfigMap = vzapp.NamespaceName{}
		metricsBinding.Spec.PrometheusConfigSecret = vzapp.SecretKey{
			Namespace: vzconst.PrometheusOperatorNamespace,
			Name:      vzconst.PromAdditionalScrapeConfigsSecretName,
			Key:       vzconst.PromAdditionalScrapeConfigsSecretKey,
		}
	}

	return nil
}

// isLegacyVmiPrometheusConfigMapName returns true if the given NamespaceName is that of the legacy
// vmi system prometheus config map
func isLegacyVmiPrometheusConfigMapName(configMapName vzapp.NamespaceName) bool {
	return configMapName.Namespace == constants.VerrazzanoSystemNamespace &&
		configMapName.Name == vzconst.VmiPromConfigName
}

// findMatchingTemplate returns a matching template for a given namespace
func (a *WorkloadWebhook) findMatchingTemplate(ctx context.Context, unst *unstructured.Unstructured, namespace string, log *zap.SugaredLogger) (*vzapp.MetricsTemplate, error) {
	// Get the list of metrics templates for the given namespace
	templateList := &vzapp.MetricsTemplateList{}
	err := a.Client.List(ctx, templateList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		log.Errorf("Failed getting list of metrics templates in namespace %s: %v", namespace, err)
		return nil, err
	}

	ws := &workloadselector.WorkloadSelector{
		KubeClient: a.KubeClient,
	}

	// Iterate through the metrics template list and check if we find a matching template for the workload resource
	for _, template := range templateList.Items {
		// If the template workload selector was not specified then don't try to match this template
		if reflect.DeepEqual(template.Spec.WorkloadSelector, vzapp.WorkloadSelector{}) {
			log.Infof("Metrics template %s/%s workloadSelector not specified - no workload match checking performed", template.Namespace, template.Name)
			continue
		}
		found, err := ws.DoesWorkloadMatch(unst,
			&template.Spec.WorkloadSelector.NamespaceSelector,
			&template.Spec.WorkloadSelector.ObjectSelector,
			template.Spec.WorkloadSelector.APIGroups,
			template.Spec.WorkloadSelector.APIVersions,
			template.Spec.WorkloadSelector.Resources)
		if err != nil {
			log.Errorf("Failed looking for a matching metrics template: %v", err)
			return nil, err
		}
		// Found a match, return the matching metrics template
		if found {
			log.Infof("Found matching metrics template %s/%s", namespace, template.Name)
			return &template, nil
		}
	}

	return nil, nil
}

// Generate the metricBindings name
func generateMetricsBindingName(name string, apiVersion string, kind string) string {
	return fmt.Sprintf("%s-%s-%s", name, strings.Replace(apiVersion, "/", "-", 1), strings.ToLower(kind))
}
