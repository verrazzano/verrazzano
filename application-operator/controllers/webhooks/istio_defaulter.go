// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gertd/go-pluralize"
	securityv1beta1 "istio.io/api/security/v1beta1"
	v1beta1 "istio.io/api/type/v1beta1"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
	istioversionedclient "istio.io/client-go/pkg/clientset/versioned"
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

const istioAppLabel = "istio-app"

// IstioWebhook type for istio defaulter webhook
type IstioWebhook struct {
	IstioClient   *istioversionedclient.Clientset
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

	// Check for the annotation "sidecar.istio.io/inject: false".  No action required if annotation is set to false.
	// TODO: is the correct thing to do for Coherence pods?
	for key, value := range pod.Annotations {
		if key == "sidecar.istio.io/inject" && value == "false" {
			return admission.Allowed("No action required, pod labeled with sidecar.istio.io/inject: false")
		}
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
	// No OAM ApplicationConfiguration ownerReference resource found so there is no action required.
	if appConfigOwnerRef == (metav1.OwnerReference{}) {
		istioLogger.Info(fmt.Sprintf("Not a OAM created pod: %s:%s:%s", req.Namespace, pod.Name, pod.GenerateName))
		return admission.Allowed("No action required, pod was not created from an ApplicationConfiguration resource")
	}

	// If a pod is using the "default" service account then create a appconfig specific service account, if not already
	// created. This service account will be referenced in an Istio authorization policy.
	istioLogger.Info(fmt.Sprintf("Pod serviceAccountName: %s", pod.Spec.ServiceAccountName))
	serviceAccountName := pod.Spec.ServiceAccountName
	if serviceAccountName == "default" {
		serviceAccountName, err = a.createServiceAccount(req.Namespace, appConfigOwnerRef)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
	}
	istioLogger.Info(fmt.Sprintf("Pod serviceAccountName to use: %s", serviceAccountName))

	// Create/update Istio Authorization policy
	err = a.createUpdateAuthorizationPolicy(req.Namespace, serviceAccountName, appConfigOwnerRef)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Add the label to the pod that will be used as the match selector in the authorization policy.
	pod.Labels[istioAppLabel] = appConfigOwnerRef.Name

	// Set the service account name for the pod.
	pod.Spec.ServiceAccountName = serviceAccountName

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// InjectDecoder injects the decoder.
func (a *IstioWebhook) InjectDecoder(d *admission.Decoder) error {
	a.Decoder = d
	return nil
}

func (a *IstioWebhook) createUpdateAuthorizationPolicy(namespace string, serviceAccountName string, ownerRef metav1.OwnerReference) error {
	sourcePrincipal := fmt.Sprintf("cluster.local/ns/%s/sa/%s", namespace, serviceAccountName)

	// Check if authorization policy exist.  The name of the authorization policy is the owner reference name which happens
	// to be the appconfig name.
	authPolicy, err := a.IstioClient.SecurityV1beta1().AuthorizationPolicies(namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})

	// If the authorization policy does not exist then we create it.
	if err != nil && errors.IsNotFound(err) {
		selector := v1beta1.WorkloadSelector{
			MatchLabels: map[string]string{
				istioAppLabel: ownerRef.Name,
			},
		}
		fromRules := []*securityv1beta1.Rule_From{
			{
				Source: &securityv1beta1.Source{
					Principals: []string{
						sourcePrincipal,
					},
				},
			},
		}

		ap := &clisecurity.AuthorizationPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ownerRef.Name,
				Namespace: namespace,
				Labels: map[string]string{
					istioAppLabel: ownerRef.Name,
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
			Spec: securityv1beta1.AuthorizationPolicy{
				Selector: &selector,
				Rules: []*securityv1beta1.Rule{
					{
						From: fromRules,
					},
				},
			},
		}

		istioLogger.Info(fmt.Sprintf("Creating Istio authorization policy: %s:%s", namespace, ownerRef.Name))
		_, err := a.IstioClient.SecurityV1beta1().AuthorizationPolicies(namespace).Create(context.TODO(), ap, metav1.CreateOptions{})
		return err
	} else if err != nil {
		return err
	}

	// Check if we need to add a principal to an existing Istio authorization policy.
	principalFound := false
	for _, principal := range authPolicy.Spec.GetRules()[0].From[0].Source.Principals {
		if principal == sourcePrincipal {
			principalFound = true
			break
		}
	}

	// We did not find the principal in the Istio authorization policy so update the policy with the new principal.
	if !principalFound {
		authPolicy.Spec.GetRules()[0].From[0].Source.Principals = append(authPolicy.Spec.GetRules()[0].From[0].Source.Principals, sourcePrincipal)
		istioLogger.Info(fmt.Sprintf("Updating Istio authorization policy: %s:%s", namespace, ownerRef.Name))
		_, err := a.IstioClient.SecurityV1beta1().AuthorizationPolicies(namespace).Update(context.TODO(), authPolicy, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *IstioWebhook) createServiceAccount(namespace string, ownerRef metav1.OwnerReference) (string, error) {
	// Check if service account exist.  The name of the service account is the owner reference name which happens
	// to be the appconfig name.
	serviceAccount, err := a.KubeClient.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})

	// If the service account does not exist then we create it.
	if err != nil && errors.IsNotFound(err) {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ownerRef.Name,
				Namespace: namespace,
				Labels: map[string]string{
					istioAppLabel: ownerRef.Name,
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
		istioLogger.Info(fmt.Sprintf("Creating service account: %s:%s", namespace, serviceAccount.Name))
		serviceAccount, err = a.KubeClient.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), sa, metav1.CreateOptions{})
		if err != nil {
			return "", err
		}
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
