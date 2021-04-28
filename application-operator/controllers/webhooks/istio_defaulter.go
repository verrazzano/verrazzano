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
	cluv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers"
	securityv1beta1 "istio.io/api/security/v1beta1"
	"istio.io/api/type/v1beta1"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
	istioversionedclient "istio.io/client-go/pkg/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// IstioDefaulterPath specifies the path of Istio defaulter webhook
const IstioDefaulterPath = "/istio-defaulter"

// IstioAppLabel label to be used for all pods that are istio enabled
const IstioAppLabel = "verrazzano.io/istio"

// IstioWebhook type for istio defaulter webhook
type IstioWebhook struct {
	client.Client
	IstioClient   istioversionedclient.Interface
	Decoder       *admission.Decoder
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface
}

var istioLogger = ctrl.Log.WithName("webhooks.istio-defaulter")

// Handle is the entry point for the mutating webhook.
// This function is called for any pods that are created in a namespace with the label istio-injection=enabled.
func (a *IstioWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := a.Decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Check for the annotation "sidecar.istio.io/inject: false".  No action required if annotation is set to false.
	for key, value := range pod.Annotations {
		if key == "sidecar.istio.io/inject" && value == "false" {
			istioLogger.Info(fmt.Sprintf("Pod labeled with sidecar.istio.io/inject: false: %s:%s:%s", req.Namespace, pod.Name, pod.GenerateName))
			return admission.Allowed("No action required, pod labeled with sidecar.istio.io/inject: false")
		}
	}

	// Get all owner references for this pod
	ownerRefList, err := a.flattenOwnerReferences(nil, req.Namespace, pod.OwnerReferences)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Check if the pod was created from an ApplicationConfiguration resource.
	// We do this by checking for the existence of an ApplicationConfiguration ownerReference resource.
	appConfigOwnerRef := metav1.OwnerReference{}
	for _, ownerRef := range ownerRefList {
		if ownerRef.Kind == "ApplicationConfiguration" {
			appConfigOwnerRef = ownerRef
			break
		}
	}
	// No ApplicationConfiguration ownerReference resource was found so there is no action required.
	if appConfigOwnerRef == (metav1.OwnerReference{}) {
		istioLogger.Info(fmt.Sprintf("Pod is not a child of an ApplicationConfiguration: %s:%s:%s", req.Namespace, pod.Name, pod.GenerateName))
		return admission.Allowed("No action required, pod is not a child of an ApplicationConfiguration resource")
	}

	// If a pod is using the "default" service account then create a app specific service account, if not already
	// created.  A service account is used as a principal in the Istio Authorization policy we create/update.
	serviceAccountName := pod.Spec.ServiceAccountName
	if serviceAccountName == "default" || serviceAccountName == "" {
		serviceAccountName, err = a.createServiceAccount(req.Namespace, appConfigOwnerRef)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
	}

	istioLogger.Info(fmt.Sprintf("Pod name/generated name: %s/%s", pod.Name, pod.GenerateName))

	// Create/update Istio Authorization policy for the given pod.
	err = a.createUpdateAuthorizationPolicy(req.Namespace, serviceAccountName, appConfigOwnerRef)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Fixup Istio Authorization policies within a project
	err = a.fixupAuthorizationPoliciesForProjects(req.Namespace)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Add the label to the pod which is used as the match selector in the authorization policy we created/updated.
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[IstioAppLabel] = appConfigOwnerRef.Name

	// Set the service account name for the pod which is used in the principal portion of the authorization policy we
	// created/updated.
	pod.Spec.ServiceAccountName = serviceAccountName

	// Marshal the mutated pod to send back in the admission review response.
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

// createUpdateAuthorizationPolicy will create/update an Istio authoriztion policy.
func (a *IstioWebhook) createUpdateAuthorizationPolicy(namespace string, serviceAccountName string, ownerRef metav1.OwnerReference) error {
	podPrincipal := fmt.Sprintf("cluster.local/ns/%s/sa/%s", namespace, serviceAccountName)
	gwPrincipal := "cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account"
	promPrincipal := "cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator"

	// Check if authorization policy exist.  The name of the authorization policy is the owner reference name which happens
	// to be the appconfig name.
	authPolicy, err := a.IstioClient.SecurityV1beta1().AuthorizationPolicies(namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})

	// If the authorization policy does not exist then we create it.
	if err != nil && errors.IsNotFound(err) {
		selector := v1beta1.WorkloadSelector{
			MatchLabels: map[string]string{
				IstioAppLabel: ownerRef.Name,
			},
		}
		fromRules := []*securityv1beta1.Rule_From{
			{
				Source: &securityv1beta1.Source{
					Principals: []string{
						podPrincipal,
						gwPrincipal,
						promPrincipal,
					},
				},
			},
		}

		ap := &clisecurity.AuthorizationPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ownerRef.Name,
				Namespace: namespace,
				Labels: map[string]string{
					IstioAppLabel: ownerRef.Name,
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
		if principal == podPrincipal {
			principalFound = true
			break
		}
	}

	// We did not find the principal in the Istio authorization policy so update the policy with the new principal.
	if !principalFound {
		authPolicy.Spec.GetRules()[0].From[0].Source.Principals = append(authPolicy.Spec.GetRules()[0].From[0].Source.Principals, podPrincipal)
		istioLogger.Info(fmt.Sprintf("Updating Istio authorization policy: %s:%s", namespace, ownerRef.Name))
		_, err := a.IstioClient.SecurityV1beta1().AuthorizationPolicies(namespace).Update(context.TODO(), authPolicy, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// createServiceAccount will create a service account to be referenced by the Istio authorization policy
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
					IstioAppLabel: ownerRef.Name,
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
		istioLogger.Info(fmt.Sprintf("Creating service account: %s:%s", namespace, ownerRef.Name))
		serviceAccount, err = a.KubeClient.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), sa, metav1.CreateOptions{})
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	return serviceAccount.Name, nil
}

// flattenOwnerReferences traverses a nested array of owner references and returns a single array of owner references.
func (a *IstioWebhook) flattenOwnerReferences(list []metav1.OwnerReference, namespace string, ownerRefs []metav1.OwnerReference) ([]metav1.OwnerReference, error) {
	for _, ownerRef := range ownerRefs {
		list = append(list, ownerRef)

		group, version := controllers.ConvertAPIVersionToGroupAndVersion(ownerRef.APIVersion)
		resource := schema.GroupVersionResource{
			Group:    group,
			Version:  version,
			Resource: pluralize.NewClient().Plural(strings.ToLower(ownerRef.Kind)),
		}

		unst, err := a.DynamicClient.Resource(resource).Namespace(namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
		if err != nil {
			istioLogger.Error(err, "Dynamic API failed")
			return nil, nil
		}

		if len(unst.GetOwnerReferences()) != 0 {
			list, err = a.flattenOwnerReferences(list, namespace, unst.GetOwnerReferences())
			if err != nil {
				return nil, nil
			}
		}
	}
	return list, nil
}

// fixupAuthorizationPoliciesForProjects updates authorization policies so that all applications within a project
// are allowed to talk to each other
func (a *IstioWebhook) fixupAuthorizationPoliciesForProjects(podNamespace string) error {
	// Get the list of defined projects
	projectsList := &cluv1alpha1.VerrazzanoProjectList{}
	listOptions := &client.ListOptions{Namespace: "verrazzano-mc"}
	err := a.Client.List(context.TODO(), projectsList, listOptions)
	if err != nil {
		return err
	}

	// Walk the list of projects looking for a project namespace that matches the given pod namespace
	for _, project := range projectsList.Items {
		namespaceFound := false
		for _, namespace := range project.Spec.Template.Namespaces {
			if namespace.Metadata.Name == podNamespace {
				namespaceFound = true
				break
			}
		}

		// Project has a namespace that matches the given pod namespace
		if namespaceFound {
			// Get the authorization policies for all the namespaces in a project.
			authzPolicyList, err := a.getAuthorizationPoliciesForProject(project.Spec.Template.Namespaces)
			if err != nil {
				return err
			}

			// Create list of unique principals for all authorization policies in a project.
			uniquePrincipals := make(map[string]bool)
			for _, authzPolicy := range authzPolicyList {
				policy, err := a.IstioClient.SecurityV1beta1().AuthorizationPolicies(authzPolicy.Namespace).Get(context.TODO(), authzPolicy.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}

				for _, principal := range policy.Spec.Rules[0].From[0].Source.Principals {
					uniquePrincipals[principal] = true
				}
			}

			// Update all authorization policies in a project.
			err = a.updateAuthorizationPoliciesForProject(authzPolicyList, uniquePrincipals)
			if err != nil {
				return err
			}

			break
		}
	}
	return nil
}

// getAuthorizationPoliciesForProject returns a list of Istio authorization policies for a given list of namespaces.
// The returned authorization policies must a have an owner reference to an applicationConfiguration resource.
func (a *IstioWebhook) getAuthorizationPoliciesForProject(namespaceList []cluv1alpha1.NamespaceTemplate) ([]clisecurity.AuthorizationPolicy, error) {
	var authzPolicyList = []clisecurity.AuthorizationPolicy{}
	for _, namespace := range namespaceList {
		// Get the list of authorization policy resources in the namespace
		list, err := a.IstioClient.SecurityV1beta1().AuthorizationPolicies(namespace.Metadata.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, authzPolicy := range list.Items {
			// If the owner reference is an appconfig resource then we had to our list of authorization policies.
			if authzPolicy.OwnerReferences[0].Kind == "ApplicationConfiguration" {
				authzPolicyList = append(authzPolicyList, authzPolicy)
			}
		}
	}

	return authzPolicyList, nil
}

// updateAuthorizationPoliciesForProject updates Istio authorization policies for a project, if needed.
func (a *IstioWebhook) updateAuthorizationPoliciesForProject(authzPolicyList []clisecurity.AuthorizationPolicy, uniquePrincipals map[string]bool) error {
	for _, authzPolicy := range authzPolicyList {
		policy, err := a.IstioClient.SecurityV1beta1().AuthorizationPolicies(authzPolicy.Namespace).Get(context.TODO(), authzPolicy.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// If the principals specified for the authorization policy do not have the expected principals then
		// we need to update them.
		if !unorderedEqual(uniquePrincipals, policy.Spec.Rules[0].From[0].Source.Principals) {
			var principals = []string{}
			for principal := range uniquePrincipals {
				principals = append(principals, principal)
			}
			policy.Spec.Rules[0].From[0].Source.Principals = principals
			istioLogger.Info(fmt.Sprintf("Updating project Istio authorization policy: %s:%s", authzPolicy.Namespace, authzPolicy.Name))
			_, err := a.IstioClient.SecurityV1beta1().AuthorizationPolicies(authzPolicy.Namespace).Update(context.TODO(), policy, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// unorderedEqual checks if a map and array have the same string elements
func unorderedEqual(uniquePrincipals map[string]bool, principals []string) bool {
	if len(uniquePrincipals) != len(principals) {
		return false
	}
	for element := range uniquePrincipals {
		if !contains(element, principals) {
			return false
		}
	}
	return true
}

// contains checks that a given string is in an array of strings
func contains(given string, list []string) bool {
	for _, matchValue := range list {
		if matchValue == given {
			return true
		}
	}
	return false
}
