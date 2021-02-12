// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// PodDefaulterPath specifies the path of PodDefaulter
const PodDefaulterPath = "/pod-defaulter"

// PodWebhook type for pod defaulter webhook
type PodWebhook struct {
	Client  client.Client
	decoder *admission.Decoder
}

// Handle identifies OAM created pods and mutates pods and adds additional resources as needed.
func (a *PodWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	var log = ctrl.Log.WithName("webhooks.pod-defaulter")

	pod := &corev1.Pod{}

	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	log.Info(fmt.Sprintf("Created pod: %s:%s", pod.Namespace, pod.Name))

	//	marshaledPod, err := json.Marshal(pod)
	//	if err != nil {
	//		return admission.Errored(http.StatusInternalServerError, err)
	//	}

	return admission.Allowed("No action required")
	//	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// InjectDecoder injects the decoder.
func (a *PodWebhook) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
