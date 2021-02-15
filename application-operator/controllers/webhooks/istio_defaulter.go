// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	pluralize "github.com/gertd/go-pluralize"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// IstioDefaulterPath specifies the path of Istio defaulter webhook
const IstioDefaulterPath = "/istio-defaulter"

// IstioWebhook type for istio defaulter webhook
type IstioWebhook struct {
	//	Client  client.Client
	Decoder       *admission.Decoder
	DynamicClient dynamic.Interface
}

// Handle identifies OAM created pods with a name selecter that matches istio-enabled:true,
// mutates pods and adds additional resources as needed.
func (a *IstioWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	var log = ctrl.Log.WithName("webhooks.istio-defaulter")

	pod := &corev1.Pod{}

	log.Info("Request %s:%s", req.Namespace, req.Name)
	err := a.Decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	u := unstructured.Unstructured{Object: obj}

	log.Info(fmt.Sprintf("Unstructured pod: %s:%s:%s", u.GetNamespace(), u.GetName(), u.GetGenerateName()))

	// Create a new dynamic client.
	//	restConfig, err := clientcmd.BuildConfigFromFlags("", "")
	//	dynamicClient, err := dynamic.NewForConfig(restConfig)
	ownerRefs := u.GetOwnerReferences()
	group, version := convertAPIVersionToGroupAndVersion(ownerRefs[0].APIVersion)
	resource := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: pluralize.NewClient().Plural(strings.ToLower(ownerRefs[0].Kind)),
	}
	log.Info(fmt.Sprintf("resource.Group: %s", resource.Group))
	log.Info(fmt.Sprintf("resource.Version: %s", resource.Version))
	log.Info(fmt.Sprintf("resource.Resource: %s", resource.Resource))
	log.Info(fmt.Sprintf("Namespace: %s", req.Namespace))
	log.Info(fmt.Sprintf("Name: %s", ownerRefs[0].Name))
	unst, err := a.DynamicClient.Resource(resource).Namespace(req.Namespace).Get(context.TODO(), ownerRefs[0].Name, metav1.GetOptions{})
	if err != nil {
		log.Info(fmt.Sprintf("Dynamic API failed: %v", err))
	} else {
		log.Info(fmt.Sprintf("Dynamic client get: %s:%s", unst.GetNamespace(), unst.GetName()))
	}

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
	a.Decoder = d
	return nil
}

// convertAPIVersionToGroupAndVersion splits APIVersion into API and version parts.
// An APIVersion takes the form api/version (e.g. networking.k8s.io/v1)
// If the input does not contain a / the group is defaulted to the empty string.
// apiVersion - The combined api and version to split
func convertAPIVersionToGroupAndVersion(apiVersion string) (string, string) {
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) < 2 {
		// Use empty group for core types.
		return "", parts[0]
	}
	return parts[0], parts[1]
}
