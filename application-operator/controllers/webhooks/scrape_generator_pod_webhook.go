// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gertd/go-pluralize"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	ScrapeGeneratorPodPath = "/scrape-generator-pod"
)

var scrapeGeneratorPodLogger = ctrl.Log.WithName("webhooks.scrape-generator-pod")

// ScrapeGeneratorPodWebhook type for the mutating webhook
type ScrapeGeneratorPodWebhook struct {
	client.Client
	Decoder       *admission.Decoder
	DynamicClient dynamic.Interface
}

// Handle is the handler for the mutating webhook
func (a *ScrapeGeneratorPodWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	scrapeGeneratorPodLogger.V(1).Info("scrape-generator-pod webhook called", "namespace", req.Namespace)
	return a.handlePodResource(req)
}

// InjectDecoder injects the decoder.
func (a *ScrapeGeneratorPodWebhook) InjectDecoder(d *admission.Decoder) error {
	a.Decoder = d
	return nil
}

// handlePodResource decodes the admission request for a pod resource into a Pod struct
// and then processes the pod resource
func (a *ScrapeGeneratorPodWebhook) handlePodResource(req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := a.Decoder.Decode(req, pod)
	if err != nil {
		scrapeGeneratorPodLogger.Error(err, "error decoding object in admission request")
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Look for the app.verrazzano.io/metrics-binding label in the owner references
	value, err := a.findMetricsBindingLabel(req.Namespace, pod.OwnerReferences)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	scrapeGeneratorPodLogger.Info(fmt.Sprintf("Setting pod label %s to %s", constants.MetricsBindingLabel, value), "name", pod.GenerateName)

	// Set the app.verrazzano.io/metrics-binding label to the value found in the owner references
	if len(value) != 0 {
		labels := pod.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[constants.MetricsBindingLabel] = value
		pod.SetLabels(labels)
	}

	marshaledPodResource, err := json.Marshal(pod)
	if err != nil {
		scrapeGeneratorPodLogger.Error(err, "error marshalling pod resource")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPodResource)
}

// findMetricsBindingLabel traverses a nested array of owner references and returns the value of
// the app.verrazzano.io/metrics-binding label if found in an owner reference
// resource
func (a *ScrapeGeneratorPodWebhook) findMetricsBindingLabel(namespace string, ownerRefs []metav1.OwnerReference) (string, error) {
	for _, ownerRef := range ownerRefs {
		group, version := controllers.ConvertAPIVersionToGroupAndVersion(ownerRef.APIVersion)
		resource := schema.GroupVersionResource{
			Group:    group,
			Version:  version,
			Resource: pluralize.NewClient().Plural(strings.ToLower(ownerRef.Kind)),
		}

		unst, err := a.DynamicClient.Resource(resource).Namespace(namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
		if err != nil {
			istioLogger.Error(err, "Dynamic API failed")
			return "", err
		}

		labels := unst.GetLabels()
		if value, ok := labels[constants.MetricsBindingLabel]; ok {
			return value, nil
		}

		if len(unst.GetOwnerReferences()) != 0 {
			value, err := a.findMetricsBindingLabel(namespace, unst.GetOwnerReferences())
			if err != nil {
				return "", err
			}
			if len(value) != 0 {
				return value, nil
			}
		}
	}
	return "", nil
}
