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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// Get all owner references for this pod
	ownerRefList := a.flatten(nil, req.Namespace, pod.OwnerReferences)

	// Check if the pod was created from an OAM ApplicationConfiguration resource.
	// We do this by checking for the existence of an OAM ApplicationConfiguration ownerReference resource.
	appConfigOwnerRef := metav1.OwnerReference{}
	for _, ownerRef := range ownerRefList {
		istioLogger.Info(fmt.Sprintf("ownerReference: %s:%s", ownerRef.Kind, ownerRef.Name))
		if ownerRef.Kind == "ApplicationConfiguration" {
			istioLogger.Info(fmt.Sprintf("OAM created pod: %s:%s:%s", req.Namespace, pod.Name, pod.GenerateName))
			appConfigOwnerRef = ownerRef
			break
		}
	}
	// No OAM ApplicationConfiguration ownerReference resource found
	if appConfigOwnerRef == (metav1.OwnerReference{}) {
		istioLogger.Info(fmt.Sprintf("Not a OAM created pod: %s:%s:%s", req.Namespace, pod.Name, pod.GenerateName))
		return admission.Allowed("No action required, pod was not created from an ApplicationConfiguration resource")
	}

	// Create a service account if the pod is using the default service account.  The service account will
	// be referenced in an Istio authorization policy.
	istioLogger.Info(fmt.Sprintf("Pod serviceAccountName: %s", pod.Spec.ServiceAccountName))
	serviceAccountName := pod.Spec.ServiceAccountName
	if serviceAccountName == "default" {
		serviceAccountName, err = a.createServiceAccountIfNeeded(req.Namespace, appConfigOwnerRef)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
	}
	istioLogger.Info(fmt.Sprintf("Pod serviceAccountName to use: %s", serviceAccountName))

	// TODO: create/update authz policy

	// TODO: set application specific label for OAM pod

	// TODO: set serviceAccountName for OAM pod, if needed

	//	marshaledPod, err := json.Marshal(pod)
	//	if err != nil {
	//		return admission.Errored(http.StatusInternalServerError, err)
	//	}

	return admission.Allowed("No action required")
	//	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// InjectDecoder injects the decoder.
func (a *IstioWebhook) InjectDecoder(d *admission.Decoder) error {
	a.Decoder = d
	return nil
}

func (a *IstioWebhook) createServiceAccountIfNeeded(namespace string, ownerRef metav1.OwnerReference) (string, error) {
	// Check if service account exists.  If it does not exist, create it.
	serviceAccount, err := a.KubeClient.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ownerRef.Name,
				Namespace: namespace,
				Labels: map[string]string{
					"istio-app": ownerRef.Name,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						Name:       ownerRef.Name,
						Kind:       ownerRef.Kind,
						APIVersion: ownerRef.APIVersion,
						UID:        ownerRef.UID,
					},
				},
			},
		}
		serviceAccount, err = a.KubeClient.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), sa, metav1.CreateOptions{})
		if err != nil {
			return "", err
		}
		istioLogger.Info(fmt.Sprintf("Created service account: %s", serviceAccount.Name))
	} else if err != nil {
		return "", err
	}

	return serviceAccount.Name, nil
}

func (a *IstioWebhook) flatten(list []metav1.OwnerReference, namespace string, ownerRefs []metav1.OwnerReference) []metav1.OwnerReference {
	for _, ownerRef := range ownerRefs {
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
			istioLogger.Error(err, "Dynamic API failed")
		}

		//  TODO: Handle error dynamic client api call
		if len(unst.GetOwnerReferences()) != 0 {
			list = a.flatten(list, namespace, unst.GetOwnerReferences())
		}
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
