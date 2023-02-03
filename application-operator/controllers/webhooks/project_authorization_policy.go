// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	"strings"

	cluv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"go.uber.org/zap"
	"istio.io/client-go/pkg/apis/security/v1beta1"
	istioversionedclient "istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const verrazzanoIstioLabel = "verrazzano.io/istio"

// AuthorizationPolicy type for fixing up authorization policies for projects
type AuthorizationPolicy struct {
	client.Client
	KubeClient  kubernetes.Interface
	IstioClient istioversionedclient.Interface
}

// cleanupAuthorizationPoliciesForProjects updates authorization policies so that all applications within a project
// are allowed to talk to each other after delete of an appconfig.  This function is called from the appconfig-defaulter
// webhook when an appconfig resource is deleted. This function will fixup the remaining authorization policies to not
// reference the deleted appconfig.
func (ap *AuthorizationPolicy) cleanupAuthorizationPoliciesForProjects(namespace string, appConfigName string, log *zap.SugaredLogger) error {
	// Get the list of defined projects
	projectsList := &cluv1alpha1.VerrazzanoProjectList{}
	listOptions := &client.ListOptions{Namespace: constants.VerrazzanoMultiClusterNamespace}
	err := ap.Client.List(context.TODO(), projectsList, listOptions)
	if err != nil {
		return err
	}

	// Walk the list of projects looking for a project namespace that matches the given namespace
	for _, project := range projectsList.Items {
		namespaceFound := false
		for _, ns := range project.Spec.Template.Namespaces {
			if ns.Metadata.Name == namespace {
				namespaceFound = true
				break
			}
		}

		// Project has a namespace that matches the given namespace
		if namespaceFound {
			// Get the authorization policies for all the namespaces in a project.
			tempAuthzPolicyList, err := ap.getAuthorizationPoliciesForProject(project.Spec.Template.Namespaces)
			if err != nil {
				return err
			}

			// Filter the authorization policies we retrieved.  Do not include authorization policies for the
			// appconfig being deleted.
			authzPolicyList := []*v1beta1.AuthorizationPolicy{}
			for _, policy := range tempAuthzPolicyList {
				if value, ok := policy.Spec.Selector.MatchLabels[verrazzanoIstioLabel]; ok {
					if value != appConfigName {
						authzPolicyList = append(authzPolicyList, policy)
					}
				}
			}

			// After filtering there are no authorization policies so nothing to do - we can return now.
			if len(authzPolicyList) == 0 {
				return nil
			}

			// Get a list of pods for the given namespace
			podList, err := ap.KubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return err
			}

			// Get the list of service accounts remaining in the namespace of where an appconfig delete is taking place.
			// This service account list does not include the service account used by the appconfig being deleted.
			saList := []string{}
			for _, pod := range podList.Items {
				if value, ok := pod.Labels[verrazzanoIstioLabel]; ok {
					if value != appConfigName {
						saList = append(saList, pod.Spec.ServiceAccountName)
					}
				}
			}

			// Create list of unique principals for all authorization policies in a project.
			// This list does not include principals used by the appconfig being deleted.
			uniquePrincipals := make(map[string]bool)
			for _, authzPolicy := range authzPolicyList {
				for _, principal := range authzPolicy.Spec.Rules[0].From[0].Source.Principals {
					// For the namespace passed, only include the service accounts remaining for the namespace
					split := strings.Split(principal, "/")
					if len(split) != 5 {
						return fmt.Errorf("expected format of Istio authorization policy is cluster.local/ns/<namespace>/sa/<service-account>")
					}
					if split[2] == namespace {
						for _, sa := range saList {
							if split[4] == sa {
								uniquePrincipals[principal] = true
							}
						}
						continue
					}
					uniquePrincipals[principal] = true
				}
			}

			// Update all authorization policies in a project.
			err = ap.updateAuthorizationPoliciesForProject(authzPolicyList, uniquePrincipals, log)
			if err != nil {
				return err
			}

			break
		}
	}
	return nil
}

// fixupAuthorizationPoliciesForProjects updates authorization policies so that all applications within a project
// are allowed to talk to each other. This function is called by the istio-defaulter webhook when authorization
// policies are created.
func (ap *AuthorizationPolicy) fixupAuthorizationPoliciesForProjects(namespace string, log *zap.SugaredLogger) error {
	// Get the list of defined projects
	projectsList := &cluv1alpha1.VerrazzanoProjectList{}
	listOptions := &client.ListOptions{Namespace: constants.VerrazzanoMultiClusterNamespace}
	err := ap.Client.List(context.TODO(), projectsList, listOptions)
	if err != nil {
		return err
	}

	// Walk the list of projects looking for a project namespace that matches the given namespace
	for _, project := range projectsList.Items {
		namespaceFound := false
		for _, ns := range project.Spec.Template.Namespaces {
			if ns.Metadata.Name == namespace {
				namespaceFound = true
				break
			}
		}

		// Project has a namespace that matches the given namespace
		if namespaceFound {
			// Get the authorization policies for all the namespaces in a project.
			tempAuthzPolicyList, err := ap.getAuthorizationPoliciesForProject(project.Spec.Template.Namespaces)
			if err != nil {
				return err
			}

			// Filter the authorization policies we retrieved
			authzPolicyList := []*v1beta1.AuthorizationPolicy{}
			for _, policy := range tempAuthzPolicyList {
				if _, ok := policy.Spec.Selector.MatchLabels[verrazzanoIstioLabel]; ok {
					authzPolicyList = append(authzPolicyList, policy)
				}
			}

			// Create list of unique principals for all authorization policies in a project.
			uniquePrincipals := make(map[string]bool)
			for _, authzPolicy := range authzPolicyList {
				for _, principal := range authzPolicy.Spec.Rules[0].From[0].Source.Principals {
					uniquePrincipals[principal] = true
				}
			}

			// Update all authorization policies in a project.
			err = ap.updateAuthorizationPoliciesForProject(authzPolicyList, uniquePrincipals, log)
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
func (ap *AuthorizationPolicy) getAuthorizationPoliciesForProject(namespaceList []cluv1alpha1.NamespaceTemplate) ([]*v1beta1.AuthorizationPolicy, error) {
	var authzPolicyList = []*v1beta1.AuthorizationPolicy{}
	for _, namespace := range namespaceList {
		// Get the list of authorization policy resources in the namespace
		list, err := ap.IstioClient.SecurityV1beta1().AuthorizationPolicies(namespace.Metadata.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, authzPolicy := range list.Items {
			// If the owner reference is an appconfig resource then
			// we add the authorization policy to our list of authorization policies
			if authzPolicy.OwnerReferences[0].Kind == "ApplicationConfiguration" {
				authzPolicyList = append(authzPolicyList, authzPolicy)
			}
		}
	}

	return authzPolicyList, nil
}

// updateAuthorizationPoliciesForProject updates Istio authorization policies for a project, if needed.
func (ap *AuthorizationPolicy) updateAuthorizationPoliciesForProject(authzPolicyList []*v1beta1.AuthorizationPolicy, uniquePrincipals map[string]bool, log *zap.SugaredLogger) error {
	for i, authzPolicy := range authzPolicyList {
		// If the principals specified for the authorization policy do not have the expected principals then
		// we need to update them.
		if !vzstring.UnorderedEqual(uniquePrincipals, authzPolicy.Spec.Rules[0].From[0].Source.Principals) {
			var principals = []string{}
			for principal := range uniquePrincipals {
				principals = append(principals, principal)
			}
			authzPolicy.Spec.Rules[0].From[0].Source.Principals = principals
			log.Debugf("Updating project Istio authorization policy: %s:%s", authzPolicy.Namespace, authzPolicy.Name)
			_, err := ap.IstioClient.SecurityV1beta1().AuthorizationPolicies(authzPolicy.Namespace).Update(context.TODO(), authzPolicyList[i], metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
