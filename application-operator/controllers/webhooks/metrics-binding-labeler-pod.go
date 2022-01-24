// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/gertd/go-pluralize"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	MetricsBindingLabelerPodPath = "/metrics-binding-labeler-pod"
)

// LabelerPodWebhook type for the mutating webhook
type LabelerPodWebhook struct {
	client.Client
	Decoder       *admission.Decoder
	DynamicClient dynamic.Interface
}

// Handle is the handler for the mutating webhook
func (a *LabelerPodWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	log := zap.S().With("webhooks.metrics-binding-labeler-pod")

	log.Info("metrics-binding-labeler-pod webhook called", "namespace", req.Namespace, "name", req.Name)
	return a.handlePodResource(req, log)
}

// InjectDecoder injects the decoder.
func (a *LabelerPodWebhook) InjectDecoder(d *admission.Decoder) error {
	a.Decoder = d
	return nil
}

// handlePodResource decodes the admission request for a pod resource into a Pod struct
// and then processes the pod resource
func (a *LabelerPodWebhook) handlePodResource(req admission.Request, log *zap.SugaredLogger) admission.Response {
	pod := &corev1.Pod{}
	err := a.Decoder.Decode(req, pod)
	if err != nil {
		log.Errorf("Failed decoding object in admission request: %v", err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	var workloadLabel string
	// Get the workload resource for the given pod if there are owner references
	if len(pod.OwnerReferences) != 0 {
		workloads, err := a.getWorkloadResource(nil, req.Namespace, pod.OwnerReferences, log)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
		if len(workloads) > 1 {
			err = fmt.Errorf("multiple workload resources found for %s, Verrazzano metrics cannot be enabled", pod.Name)
			log.Errorf("Failed identifying workload resource: %v", err)
			return admission.Errored(http.StatusInternalServerError, err)
		}
		workloadLabel = generateMetricsBindingName(workloads[0].GetName(), workloads[0].GetAPIVersion(), workloads[0].GetKind())
	} else {
		workloadLabel = generateMetricsBindingName(pod.Name, pod.APIVersion, pod.Kind)
	}

	// Set the app.verrazzano.io/workload to identify the Prometheus config scrape target
	labels := pod.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[constants.MetricsWorkloadLabel] = workloadLabel
	pod.SetLabels(labels)
	log.Debugw(fmt.Sprintf("Setting pod label %s to %s", constants.MetricsWorkloadLabel, workloadLabel), "namespace", req.Namespace, "name", req.Name)

	marshaledPodResource, err := json.Marshal(pod)
	if err != nil {
		log.Errorf("Failed marshalling pod resource: %v", err)
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPodResource)
}

// getWorkloadResource traverses a nested array of owner references and returns a list of resources
// that have no owner references.  Most likely, the list will have only one resource
func (a *LabelerPodWebhook) getWorkloadResource(resources []*unstructured.Unstructured, namespace string, ownerRefs []metav1.OwnerReference, log *zap.SugaredLogger) ([]*unstructured.Unstructured, error) {
	for _, ownerRef := range ownerRefs {
		group, version := controllers.ConvertAPIVersionToGroupAndVersion(ownerRef.APIVersion)
		resource := schema.GroupVersionResource{
			Group:    group,
			Version:  version,
			Resource: pluralize.NewClient().Plural(strings.ToLower(ownerRef.Kind)),
		}

		unst, err := a.DynamicClient.Resource(resource).Namespace(namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Failed getting the Dynamic API: %v", err)
			return nil, err
		}

		if len(unst.GetOwnerReferences()) == 0 {
			resources = append(resources, unst)
		} else {
			resources, err = a.getWorkloadResource(resources, namespace, unst.GetOwnerReferences(), log)
			if err != nil {
				return nil, err
			}
		}
	}

	return resources, nil
}
