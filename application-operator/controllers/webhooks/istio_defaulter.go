// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gertd/go-pluralize"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// IstioDefaulterPath specifies the path of Istio defaulter webhook
const IstioDefaulterPath = "/istio-defaulter"

// IstioWebhook type for istio defaulter webhook
type IstioWebhook struct {
	//	Client  client.Client
	Decoder       *admission.Decoder
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface
}

var istioLogger = ctrl.Log.WithName("webhooks.istio-defaulter")

// Handle identifies OAM created pods with a name selecter that matches istio-enabled:true,
// mutates pods and adds additional resources as needed.
func (a *IstioWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {

	pod := &corev1.Pod{}
	err := a.Decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	istioLogger.Info(fmt.Sprintf("Pod serviceAccountName: %s", pod.Spec.ServiceAccountName))

	// TODO: probably not needed
	// Get the unstructured version of the pod
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	u := unstructured.Unstructured{Object: obj}

	istioLogger.Info(fmt.Sprintf("Unstructured pod: %s:%s:%s", u.GetNamespace(), u.GetName(), u.GetGenerateName()))

	ownerRefList := a.flatten(nil, req.Namespace, pod.OwnerReferences[0])
	for _, ref := range ownerRefList {
		istioLogger.Info(fmt.Sprintf("ownerReference: %s:%s", ref.Kind, ref.Name))
	}

	//	marshaledPod, err := json.Marshal(pod)
	//	if err != nil {
	//		return admission.Errored(http.StatusInternalServerError, err)
	//	}

	// TODO: set label that says this is an istio enabled OAM pod
	// TODO: set serviceAccountName is neccessary

	return admission.Allowed("No action required")
	//	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// InjectDecoder injects the decoder.
func (a *IstioWebhook) InjectDecoder(d *admission.Decoder) error {
	a.Decoder = d
	return nil
}

func (a *IstioWebhook) flatten(list []metav1.OwnerReference, namespace string, ownerRef metav1.OwnerReference) []metav1.OwnerReference {

	list = append(list, ownerRef)

	group, version := convertAPIVersionToGroupAndVersion(ownerRef.APIVersion)
	resource := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: pluralize.NewClient().Plural(strings.ToLower(ownerRef.Kind)),
	}
	istioLogger.Info(fmt.Sprintf("resource.Group: %s", resource.Group))
	istioLogger.Info(fmt.Sprintf("resource.Version: %s", resource.Version))
	istioLogger.Info(fmt.Sprintf("resource.Resource: %s", resource.Resource))
	istioLogger.Info(fmt.Sprintf("Namespace: %s", namespace))
	istioLogger.Info(fmt.Sprintf("Name: %s", ownerRef.Name))
	unst, err := a.DynamicClient.Resource(resource).Namespace(namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
	if err != nil {
		istioLogger.Info(fmt.Sprintf("Dynamic API failed: %v", err))
	} else {
		istioLogger.Info(fmt.Sprintf("Dynamic client get: %s:%s", unst.GetNamespace(), unst.GetName()))
	}

	if len(unst.GetOwnerReferences()) != 0 {
		list = a.flatten(list, namespace, unst.GetOwnerReferences()[0])
	}

	return list
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
