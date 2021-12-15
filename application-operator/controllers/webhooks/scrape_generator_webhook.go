// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
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

var scrapeGeneratorLogger = ctrl.Log.WithName("webhooks.scrape-generator")

// ScrapeGeneratorWebhook type for the mutating webhook
type ScrapeGeneratorWebhook struct {
	client.Client
	Decoder    *admission.Decoder
	KubeClient kubernetes.Interface
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

	// For the time being, do not handle any workload resources that have owner references.
	// NOTE: this will be revisited.
	if len(unst.GetOwnerReferences()) != 0 {
		return admission.Allowed(StatusReasonSuccess)
	}

	// Namespace of workload resource must be labeled with "verrazzano-managed": "true"
	namespace, err := a.KubeClient.CoreV1().Namespaces().Get(ctx, unst.GetNamespace(), metav1.GetOptions{})
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if _, ok := namespace.Labels[constants.LabelVerrazzanoManaged]; !ok {
		return admission.Allowed(StatusReasonSuccess)
	}

	// Process the app.verrazzano.io/metrics annotation
	metricsTemplate, err := a.processMetricsAnnotation(unst)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Workload resource has a valid metric template
	if metricsTemplate != nil {

	}

	marshaledWorkloadResource, err := json.Marshal(unst)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledWorkloadResource)
}

func (a *ScrapeGeneratorWebhook) processMetricsAnnotation(unst *unstructured.Unstructured) (*vzapp.MetricsTemplate, error) {
	if metricsTemplate, ok := unst.GetAnnotations()["app.verrazzano.io/metrics"]; ok {
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
