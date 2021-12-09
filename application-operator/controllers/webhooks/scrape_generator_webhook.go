// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/application-operator/internal/certificates"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// ScrapeGeneratorloadPath specifies the path of scrape-generator webhook
const ScrapeGeneratorloadPath = "/scrape-generator"

var workloadLogger = ctrl.Log.WithName("webhooks.scrape-generator")

// WorkloadWebhook type for workload mutating webhook
type WorkloadWebhook struct {
	client.Client
	Decoder       *admission.Decoder
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface
}

// Handle - handler for the mutating webhook
func (a *WorkloadWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	workloadLogger.Info(fmt.Sprintf("entered %s webhook for group: %s, version: %s, kind: %s, namespace: %s, count: %d", certificates.ScrapeGeneratorWebhookName, req.Kind.Group, req.Kind.Version, req.Kind.Kind, req.Namespace))

	return admission.Allowed("not implemented yet")
}
