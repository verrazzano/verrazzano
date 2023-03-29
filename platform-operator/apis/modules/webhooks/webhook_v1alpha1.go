// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	ValidateModulesWebhookPath  = "verrazzano-modules"
	ValidateModulesWebhooksPath = "/validate-modules-v1alpha1-install-verrazzano-io"
)

type WebhookV1alpha1 struct{}

func (v *WebhookV1alpha1) Handle(_ context.Context, _ admission.Request) admission.Response {
	zap.S().Infof("Handled module admission request")
	return admission.Allowed("")
}
