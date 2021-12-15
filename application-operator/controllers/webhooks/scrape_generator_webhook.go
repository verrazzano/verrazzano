// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"k8s.io/client-go/dynamic"

	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/workloadselector"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// ScrapeGeneratorLoadPath specifies the path of scrape-generator webhook
const ScrapeGeneratorLoadPath = "/scrape-generator"

// StatusReasonSuccess constant for successful response
const StatusReasonSuccess = "success"

const (
	MetricsAnnotation                 = "app.verrazzano.io/metrics"
	MetricsWorkloadUidAnnotation      = "app.verrazzano.io/metrics-workload-uid"
	MetricsTemplateUidAnnotation      = "app.verrazzano.io/metrics-template-uid"
	MetricsPromConfigMapUidAnnotation = "app.verrazzano.io/metrics-prometheus-configmap-uid"
)

var scrapeGeneratorLogger = ctrl.Log.WithName("webhooks.scrape-generator")

// ScrapeGeneratorWebhook type for the mutating webhook
type ScrapeGeneratorWebhook struct {
	client.Client
	Decoder       *admission.Decoder
	DynamicClient dynamic.Interface
	KubeClient    kubernetes.Interface
}

// Handle - handler for the mutating webhook
func (a *ScrapeGeneratorWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	scrapeGeneratorLogger.Info(fmt.Sprintf("group: %s, version: %s, kind: %s, namespace: %s", req.Kind.Group, req.Kind.Version, req.Kind.Kind, req.Namespace))

	// Check the type of resource in the admission request
	switch strings.ToLower(req.Kind.Kind) {
	case "pod", "deployment", "replicaset", "statefulset", "domain", "coherence":
		return a.handleWorkloadResource(ctx, req)
	default:
		scrapeGeneratorLogger.Info(fmt.Sprintf("unsupported kind %s", req.Kind.Kind))
		return admission.Allowed("not implemented yet")
	}
}

// InjectDecoder injects the decoder.
func (a *ScrapeGeneratorWebhook) InjectDecoder(d *admission.Decoder) error {
	a.Decoder = d
	return nil
}

func (a *ScrapeGeneratorWebhook) handleWorkloadResource(ctx context.Context, req admission.Request) admission.Response {
	unst := &unstructured.Unstructured{}
	err := a.Decoder.Decode(req, unst)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Do not handle any workload resources that have owner references.
	// NOTE: this will be revisited.
	if len(unst.GetOwnerReferences()) != 0 {
		return admission.Allowed(StatusReasonSuccess)
	}

	// Namespace of workload resource must be labeled with "verrazzano-managed": "true".
	// If not labeled this way there is nothing to do.
	namespace, err := a.KubeClient.CoreV1().Namespaces().Get(ctx, unst.GetNamespace(), metav1.GetOptions{})
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if _, ok := namespace.Labels[constants.LabelVerrazzanoManaged]; !ok {
		return admission.Allowed(StatusReasonSuccess)
	}

	// Process the app.verrazzano.io/metrics annotation and get the metrics template if specified.
	metricsTemplate, err := a.processMetricsAnnotation(unst)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Workload resource specifies a valid metrics template.
	// We use that metrics template and add the required annotations.
	if metricsTemplate != nil {
		err = a.populateAnnotations(ctx, unst, metricsTemplate)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
	} else {
		// Workload resource does not specify a metrics template.
		// Look for a matching metrics template workload whose workload selector matches.
		// First, check the namepsace of the workload resource and then check the verrazzano-system namespace
		// NOTE: use the first match for now
		found := true
		metricsTemplate, err := a.findMatchingTemplate(ctx, unst, unst.GetNamespace())
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
		if metricsTemplate == nil {
			metricsTemplate, err := a.findMatchingTemplate(ctx, unst, constants.VerrazzanoSystemNamespace)
			if err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
			if metricsTemplate == nil {
				found = false
			}
		}

		// We found a matching metrics template so add the required annotations.
		if found {
			err = a.populateAnnotations(ctx, unst, metricsTemplate)
			if err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
	}

	marshaledWorkloadResource, err := json.Marshal(unst)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledWorkloadResource)
}

// processMetricsAnnotation checks the workload resource for the "app.verrazzano.io/metrics" annotation and returns the
// metrics template referenced in the annotation
func (a *ScrapeGeneratorWebhook) processMetricsAnnotation(unst *unstructured.Unstructured) (*vzapp.MetricsTemplate, error) {
	if metricsTemplate, ok := unst.GetAnnotations()[MetricsAnnotation]; ok {
		if metricsTemplate == "none" {
			return nil, nil
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
					return nil, err
				}
				return template, nil
			}
			return nil, err
		}

		return template, nil
	}

	return nil, nil
}

// populateAnnotations adds metrics annotations to the workload resource
func (a *ScrapeGeneratorWebhook) populateAnnotations(ctx context.Context, unst *unstructured.Unstructured, template *vzapp.MetricsTemplate) error {
	configMap, err := a.KubeClient.CoreV1().ConfigMaps(template.Spec.PrometheusConfig.TargetConfigMap.Namespace).Get(ctx, template.Spec.PrometheusConfig.TargetConfigMap.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	annotations := unst.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[MetricsWorkloadUidAnnotation] = string(unst.GetUID())
	annotations[MetricsTemplateUidAnnotation] = string(template.UID)
	annotations[MetricsPromConfigMapUidAnnotation] = string(configMap.UID)

	return nil
}

// findMatchingTemplate returns a matching template for a given namespace
func (a *ScrapeGeneratorWebhook) findMatchingTemplate(ctx context.Context, unst *unstructured.Unstructured, namespace string) (*vzapp.MetricsTemplate, error) {
	// Get the list of metrics templates for the given namespace
	templateList := &vzapp.MetricsTemplateList{}
	err := a.Client.List(ctx, templateList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, err
	}

	ws := &workloadselector.WorkloadSelector{
		DynamicClient: a.DynamicClient,
		KubeClient:    a.KubeClient,
	}

	// Iterate through the metrics template list and check if we find a matching template for the workload resource
	for _, template := range templateList.Items {
		found, err := ws.DoesWorkloadMatch(unst,
			&template.Spec.WorkloadSelector.NamespaceSelector,
			&template.Spec.WorkloadSelector.ObjectSelector,
			template.Spec.WorkloadSelector.APIGroups,
			template.Spec.WorkloadSelector.APIVersions,
			template.Spec.WorkloadSelector.Resources)
		if err != nil {
			return nil, err
		}
		// Found a match, return the matching metrics template
		if found {
			return &template, nil
		}
	}

	return nil, nil
}
