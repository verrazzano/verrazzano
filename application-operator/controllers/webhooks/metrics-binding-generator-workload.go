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
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	MetricsAnnotation                   = "app.verrazzano.io/metrics"
	MetricsBindingGeneratorWorkloadPath = "/metrics-binding-generator-workload"
)

// GeneratorWorkloadWebhook type for the mutating webhook
type GeneratorWorkloadWebhook struct {
	client.Client
	Decoder    *admission.Decoder
	KubeClient kubernetes.Interface
}

// Handle - handler for the mutating webhook
func (a *GeneratorWorkloadWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	log := zap.S().With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceName, req.Name, vzlog.FieldWebhook, "metrics-binding-generator-workload")
	log.Debugf("group: %s, version: %s, kind: %s", req.Kind.Group, req.Kind.Version, req.Kind.Kind)
	return a.handleWorkloadResource(ctx, req, log)
}

// InjectDecoder injects the decoder.
func (a *GeneratorWorkloadWebhook) InjectDecoder(d *admission.Decoder) error {
	a.Decoder = d
	return nil
}

// handleWorkloadResource decodes the admission request for a workload resource into an unstructured
// and then processes workload resource
func (a *GeneratorWorkloadWebhook) handleWorkloadResource(ctx context.Context, req admission.Request, log *zap.SugaredLogger) admission.Response {
	unst := &unstructured.Unstructured{}
	err := a.Decoder.Decode(req, unst)
	if err != nil {
		log.Errorf("Failed decoding object in admission request: %v", err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Do not handle any workload resources that have owner references.
	// NOTE: this will be revisited.
	if len(unst.GetOwnerReferences()) != 0 {
		return admission.Allowed(constants.StatusReasonSuccess)
	}

	// Get the workload Namespace for annotation processing
	workloadNamespace := &corev1.Namespace{}
	err = a.Client.Get(context.TODO(), types.NamespacedName{Name: unst.GetNamespace()}, workloadNamespace)
	if err != nil {
		log.Errorf("Failed getting workload namespace %s: %v", unst.GetNamespace(), err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// If "none" is specified for annotation "app.verrazzano.io/metrics" then this workload has opted out of metrics.
	if metricsTemplateAnnotation, ok := unst.GetAnnotations()[MetricsAnnotation]; ok {
		if strings.ToLower(metricsTemplateAnnotation) == "none" {
			log.Infof("%s is set to none in the workload - opting out of metrics", MetricsAnnotation)
			return admission.Allowed(constants.StatusReasonSuccess)
		}
	}

	// If "none" is specified for annotation "app.verrazzano.io/metrics" then this namespace has opted out of metrics.
	if metricsTemplateAnnotation, ok := workloadNamespace.GetAnnotations()[MetricsAnnotation]; ok {
		if strings.ToLower(metricsTemplateAnnotation) == "none" {
			log.Infof("%s is set to none in the namespace - opting out of metrics", MetricsAnnotation)
			return admission.Allowed(constants.StatusReasonSuccess)
		}
	}

	// Process the app.verrazzano.io/metrics annotation and get the metrics template, if specified.
	metricsTemplate, err := a.processMetricsAnnotation(unst, workloadNamespace, log)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Workload resource specifies a valid metrics template.
	// We use that metrics template and create/update a metrics binding resource.
	if metricsTemplate != nil {
		err = a.createOrUpdateMetricBinding(ctx, unst, metricsTemplate, log)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
	} else {
		// Workload resource does not specify a metrics template.
		// Look for a matching metrics template workload whose workload selector matches.
		// First, check the namespace of the workload resource and then check the verrazzano-system namespace
		// NOTE: use the first match for now
		var metricsTemplate *vzapp.MetricsTemplate
		found := true
		metricsTemplate, err := a.findMatchingTemplate(ctx, unst, unst.GetNamespace(), log)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
		if metricsTemplate == nil {
			template, err := a.findMatchingTemplate(ctx, unst, constants.VerrazzanoSystemNamespace, log)
			if err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
			if template == nil {
				found = false
			}
			metricsTemplate = template
		}

		// We found a matching metrics template. Create/update a metrics binding.
		if found {
			err = a.createOrUpdateMetricBinding(ctx, unst, metricsTemplate, log)
			if err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
	}

	marshaledWorkloadResource, err := json.Marshal(unst)
	if err != nil {
		log.Errorf("Failed marshalling workload resource: %v", err)
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledWorkloadResource)
}

// processMetricsAnnotation checks the workload resource for the "app.verrazzano.io/metrics" annotation and returns the
// metrics template referenced in the annotation
func (a *GeneratorWorkloadWebhook) processMetricsAnnotation(unst *unstructured.Unstructured, workloadNamespace *corev1.Namespace, log *zap.SugaredLogger) (*vzapp.MetricsTemplate, error) {
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

// createOrUpdateMetricBinding creates/updates a metricsBinding resource and
// adds the apps.verrazzano.io/workload label to the workload resource
func (a *GeneratorWorkloadWebhook) createOrUpdateMetricBinding(ctx context.Context, unst *unstructured.Unstructured, template *vzapp.MetricsTemplate, log *zap.SugaredLogger) error {
	// When the Prometheus target config map was not specified in the metrics template then there is nothing to do.
	if reflect.DeepEqual(template.Spec.PrometheusConfig.TargetConfigMap, vzapp.TargetConfigMap{}) {
		log.Infof("Prometheus target config map %s/%s not specified", template.Namespace, template.Name)
		return nil
	}

	_, err := a.KubeClient.CoreV1().ConfigMaps(template.Spec.PrometheusConfig.TargetConfigMap.Namespace).Get(ctx, template.Spec.PrometheusConfig.TargetConfigMap.Name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Failed getting Prometheus target config map %s/%s: %v", template.Namespace, template.Name, err)
		return err
	}

	// Generate the metricBindings name
	metricsBindingName := generateMetricsBindingName(unst.GetName(), unst.GetAPIVersion(), unst.GetKind())

	metricsBinding := &vzapp.MetricsBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "app.verrazzno.io/v1alpha1",
			Kind:       "metricsBinding"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: unst.GetNamespace(),
			Name:      metricsBindingName,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, a.Client, metricsBinding, func() error {
		return a.mutateMetricsBinding(metricsBinding, template, unst)
	})

	if err != nil {
		log.Errorf("Failed creating/updating metricsBinding resource: %v", err)
		return err
	}

	// Set the app.verrazzano.io/workload to identify the Prometheus config scrape target
	labels := unst.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[constants.MetricsWorkloadLabel] = metricsBindingName
	unst.SetLabels(labels)

	return nil
}

// function called by controllerutil.createOrUpdate to mutate a metricsBinding resource
func (a *GeneratorWorkloadWebhook) mutateMetricsBinding(metricsBinding *vzapp.MetricsBinding, template *vzapp.MetricsTemplate, unst *unstructured.Unstructured) error {
	metricsBinding.Spec.MetricsTemplate.Namespace = template.Namespace
	metricsBinding.Spec.MetricsTemplate.Name = template.Name
	metricsBinding.Spec.PrometheusConfigMap.Namespace = template.Spec.PrometheusConfig.TargetConfigMap.Namespace
	metricsBinding.Spec.PrometheusConfigMap.Name = template.Spec.PrometheusConfig.TargetConfigMap.Name
	metricsBinding.Spec.Workload.Name = unst.GetName()
	metricsBinding.Spec.Workload.TypeMeta = metav1.TypeMeta{APIVersion: unst.GetAPIVersion(), Kind: unst.GetKind()}
	return nil
}

// findMatchingTemplate returns a matching template for a given namespace
func (a *GeneratorWorkloadWebhook) findMatchingTemplate(ctx context.Context, unst *unstructured.Unstructured, namespace string, log *zap.SugaredLogger) (*vzapp.MetricsTemplate, error) {
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
