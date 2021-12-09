// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// IstioDefaulterPath specifies the path of Istio defaulter webhook
const PokoWorkloadPath = "/poko-workload"

var workloadLogger = ctrl.Log.WithName("webhooks.poko-workload")

var count = 0

// WorkloadWebhook type for workload mutating webhook
type WorkloadWebhook struct {
	client.Client
	Decoder       *admission.Decoder
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface
}

// Handle - handler for the mutating webhook
func (a *WorkloadWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	count++
	workloadLogger.Info(fmt.Sprintf("entered mutating webhook for group: %s, version: %s, kind: %s, namespace: %s, count: %d", req.Kind.Group, req.Kind.Version, req.Kind.Kind, req.Namespace, count))

	return admission.Allowed("work in progress")
}
