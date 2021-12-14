// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// ScrapeGeneratorLoadPath specifies the path of scrape-generator webhook
const ScrapeGeneratorLoadPath = "/scrape-generator"

// StatusReasonSuccess
const StatusReasonSuccess = "success"

var scrapeGeneratorLogger = ctrl.Log.WithName("webhooks.scrape-generator")

// ScrapeGeneratorWebhook type for the mutating webhook
type ScrapeGeneratorWebhook struct {
	client.Client
	Decoder       *admission.Decoder
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface
}

// Handle - handler for the mutating webhook
func (a *ScrapeGeneratorWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	scrapeGeneratorLogger.Info(fmt.Sprintf("group: %s, version: %s, kind: %s, namespace: %s", req.Kind.Group, req.Kind.Version, req.Kind.Kind, req.Namespace))

	// Determine what type of resource to handle
	switch strings.ToLower(req.Kind.Kind) {
	case "pod":
		return a.handlePod(ctx, req)
	case "deployment":
		return a.handleDeployment(ctx, req)
	case "replicaset":
		return a.handleReplicaSet(ctx, req)
	case "statefulset":
		return a.handleStatefulSet(ctx, req)
	case "domain":
		return a.handleDomain(ctx, req)
	case "coherence":
		return a.handleCoherence(ctx, req)
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

// handlePod - handle Kind type of Pod
func (a *ScrapeGeneratorWebhook) handlePod(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := a.Decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	return admission.Allowed(StatusReasonSuccess)
}

// handleDeployment - handle Kind type of Deployment
func (a *ScrapeGeneratorWebhook) handleDeployment(ctx context.Context, req admission.Request) admission.Response {
	deployment := &appsv1.Deployment{}
	err := a.Decoder.Decode(req, deployment)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	return admission.Allowed(StatusReasonSuccess)
}

// handleReplicaSet - handle Kind type of ReplicaSet
func (a *ScrapeGeneratorWebhook) handleReplicaSet(ctx context.Context, req admission.Request) admission.Response {
	replicaSet := &appsv1.ReplicaSet{}
	err := a.Decoder.Decode(req, replicaSet)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	return admission.Allowed(StatusReasonSuccess)
}

// handleStatefulSet - handle Kind type of StatefulSet
func (a *ScrapeGeneratorWebhook) handleStatefulSet(ctx context.Context, req admission.Request) admission.Response {
	statefulSet := &appsv1.StatefulSet{}
	err := a.Decoder.Decode(req, statefulSet)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	return admission.Allowed(StatusReasonSuccess)
}

// handleDomain - handle Kind type of Domain
func (a *ScrapeGeneratorWebhook) handleDomain(ctx context.Context, req admission.Request) admission.Response {
	return admission.Allowed(StatusReasonSuccess)
}

// handleCoherence - handle Kind type of Coherence
func (a *ScrapeGeneratorWebhook) handleCoherence(ctx context.Context, req admission.Request) admission.Response {
	return admission.Allowed(StatusReasonSuccess)
}
