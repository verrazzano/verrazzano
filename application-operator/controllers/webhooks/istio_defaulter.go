// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// IstioDefaulterPath specifies the path of Istio defaulter webhook
const IstioDefaulterPath = "/istio-defaulter"

// IstioWebhook type for istio defaulter webhook
type IstioWebhook struct {
	Client  client.Client
	decoder *admission.Decoder
}

// Handle identifies OAM created pods with a name selecter that matches istio-enabled:true,
// mutates pods and adds additional resources as needed.
func (a *IstioWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	var log = ctrl.Log.WithName("webhooks.istio-defaulter")

	pod := &corev1.Pod{}

	log.Info("Request %s:%s", req.Namespace, req.Name)
	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	u := unstructured.Unstructured{Object: obj}

	log.Info(fmt.Sprintf("Unstructured pod: %s:%s:%s", u.GetNamespace(), u.GetName(), u.GetGenerateName()))

	//	marshaledPod, err := json.Marshal(pod)
	//	if err != nil {
	//		return admission.Errored(http.StatusInternalServerError, err)
	//	}

	// TODO: set label that says this is an istio enabled OAM pod

	return admission.Allowed("No action required")
	//	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// InjectDecoder injects the decoder.
func (a *IstioWebhook) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
