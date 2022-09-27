// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	cluv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"go.uber.org/zap"
	securityv1beta1 "istio.io/api/security/v1beta1"
	"istio.io/api/type/v1beta1"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
	istiofake "istio.io/client-go/pkg/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestDeleteOnePolicyOneNamespace tests when an authorization policy is cleaned up
// GIVEN a single project with one namespace and a single authorization policy
// WHEN cleanupAuthorizationPoliciesForProjects is called
// THEN the cleanupAuthorizationPoliciesForProjects should return success
func TestDeleteOnePolicyOneNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	err := cluv1alpha1.AddToScheme(scheme)
	assert.NoError(t, err, "Unexpected error adding to scheme")
	client := ctrlfake.NewClientBuilder().WithScheme(scheme).Build()

	ap := &AuthorizationPolicy{
		Client:      client,
		KubeClient:  fake.NewSimpleClientset(),
		IstioClient: istiofake.NewSimpleClientset(),
	}

	// Create a project in the verrazzano-mc namespace
	project := &cluv1alpha1.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project",
			Namespace: "verrazzano-mc",
		},
		Spec: cluv1alpha1.VerrazzanoProjectSpec{
			Template: cluv1alpha1.ProjectTemplate{
				Namespaces: []cluv1alpha1.NamespaceTemplate{
					{Metadata: metav1.ObjectMeta{
						Name: "appconfig-namespace",
					}},
				},
			},
			Placement: cluv1alpha1.Placement{
				Clusters: []cluv1alpha1.Cluster{
					{
						Name: constants.DefaultClusterName,
					},
				},
			},
		},
	}
	err = ap.Client.Create(context.TODO(), project)
	assert.NoError(t, err, "Unexpected error creating Verrazzano project")

	// Create a Istio authorization policy in the projects namespace
	authzPolicy := &clisecurity.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "appconfig-name",
			Namespace: "appconfig-namespace",
			Labels: map[string]string{
				IstioAppLabel: "appconfig-name",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "appconfig-name",
					Kind: "ApplicationConfiguration",
				},
			},
		},
		Spec: securityv1beta1.AuthorizationPolicy{
			Selector: &v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{
					IstioAppLabel: "appconfig-name",
				},
			},
		},
	}

	_, err = ap.IstioClient.SecurityV1beta1().AuthorizationPolicies("appconfig-namespace").Create(context.TODO(), authzPolicy, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating authorization policies")

	err = ap.cleanupAuthorizationPoliciesForProjects("appconfig-namespace", "appconfig-name", zap.S())
	assert.NoError(t, err, "Unexpected error cleaning up authorization policies")
}

// TestDeleteTwoPoliciesOneNamespace tests when an authorization policy is cleaned up
// GIVEN a single projects with one namespace and two authorization policies
// WHEN cleanupAuthorizationPoliciesForProjects is called
// THEN the cleanupAuthorizationPoliciesForProjects should return success and cleanup the authorization policy of
// the remaining authorization policy
func TestDeleteTwoPoliciesOneNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	err := cluv1alpha1.AddToScheme(scheme)
	assert.NoError(t, err, "Unexpected error adding to scheme")
	client := ctrlfake.NewFakeClientWithScheme(scheme)

	ap := &AuthorizationPolicy{
		Client:      client,
		KubeClient:  fake.NewSimpleClientset(),
		IstioClient: istiofake.NewSimpleClientset(),
	}

	// Create a project in the verrazzano-mc namespace
	project := &cluv1alpha1.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project",
			Namespace: "verrazzano-mc",
		},
		Spec: cluv1alpha1.VerrazzanoProjectSpec{
			Template: cluv1alpha1.ProjectTemplate{
				Namespaces: []cluv1alpha1.NamespaceTemplate{
					{Metadata: metav1.ObjectMeta{
						Name: "appconfig-namespace",
					}},
				},
			},
			Placement: cluv1alpha1.Placement{
				Clusters: []cluv1alpha1.Cluster{
					{
						Name: constants.DefaultClusterName,
					},
				},
			},
		},
	}
	err = ap.Client.Create(context.TODO(), project)
	assert.NoError(t, err, "Unexpected error creating Verrazzano project")

	// Create a pod for appconfig-name1 in the projects namespace
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod1",
			Namespace: "appconfig-namespace",
			Labels: map[string]string{
				IstioAppLabel: "appconfig-name1",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "appconfig-name1",
					Kind:       "ApplicationConfiguration",
					APIVersion: "core.oam.dev/v1alpha2",
				},
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "appconfig-name1",
		},
	}
	_, err = ap.KubeClient.CoreV1().Pods("appconfig-namespace").Create(context.TODO(), pod, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	// Create an authorization policy for appconfig-name1 in the projects namespace
	authzPolicy := &clisecurity.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "appconfig-name1",
			Namespace: "appconfig-namespace",
			Labels: map[string]string{
				IstioAppLabel: "appconfig-name1",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "appconfig-name1",
					Kind: "ApplicationConfiguration",
				},
			},
		},
		Spec: securityv1beta1.AuthorizationPolicy{
			Selector: &v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{
					IstioAppLabel: "appconfig-name1",
				},
			},
			Rules: []*securityv1beta1.Rule{
				{
					From: []*securityv1beta1.Rule_From{
						{
							Source: &securityv1beta1.Source{
								Principals: []string{
									"cluster.local/ns/appconfig-namespace/sa/appconfig-name1",
									"cluster.local/ns/appconfig-namespace/sa/appconfig-name2",
									"cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account",
									"cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator",
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = ap.IstioClient.SecurityV1beta1().AuthorizationPolicies("appconfig-namespace").Create(context.TODO(), authzPolicy, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating authorization policies")

	// Create a pod for appconfig-name2 in the projects namespace
	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod2",
			Namespace: "appconfig-namespace",
			Labels: map[string]string{
				IstioAppLabel: "appconfig-name2",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "appconfig-name2",
					Kind:       "ApplicationConfiguration",
					APIVersion: "core.oam.dev/v1alpha2",
				},
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "appconfig-name2",
		},
	}
	_, err = ap.KubeClient.CoreV1().Pods("appconfig-namespace").Create(context.TODO(), pod, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	// Create an authorization policy for appconfig-name2 in the projects namespace
	authzPolicy2 := &clisecurity.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "appconfig-name2",
			Namespace: "appconfig-namespace",
			Labels: map[string]string{
				IstioAppLabel: "appconfig-name2",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "appconfig-name2",
					Kind: "ApplicationConfiguration",
				},
			},
		},
		Spec: securityv1beta1.AuthorizationPolicy{
			Selector: &v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{
					IstioAppLabel: "appconfig-name2",
				},
			},
			Rules: []*securityv1beta1.Rule{
				{
					From: []*securityv1beta1.Rule_From{
						{
							Source: &securityv1beta1.Source{
								Principals: []string{
									"cluster.local/ns/appconfig-namespace/sa/appconfig-name1",
									"cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account",
									"cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator",
									"cluster.local/ns/appconfig-namespace/sa/appconfig-name2",
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = ap.IstioClient.SecurityV1beta1().AuthorizationPolicies("appconfig-namespace").Create(context.TODO(), authzPolicy2, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating authorization policies")

	err = ap.cleanupAuthorizationPoliciesForProjects("appconfig-namespace", "appconfig-name1", zap.S())
	assert.NoError(t, err, "Unexpected error cleaning up authorization policies")

	updatedPolicy, err := ap.IstioClient.SecurityV1beta1().AuthorizationPolicies("appconfig-namespace").Get(context.TODO(), "appconfig-name2", metav1.GetOptions{})
	assert.NoError(t, err, "Unexpected error getting authorization policies")
	assert.Equal(t, len(updatedPolicy.Spec.Rules[0].From[0].Source.Principals), 3)
	assert.Contains(t, updatedPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account")
	assert.Contains(t, updatedPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/appconfig-namespace/sa/appconfig-name2")
	assert.Contains(t, updatedPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator")
}

// TestDeleteThreePoliciesTwoNamespace tests when an authorization policy is cleaned up
// GIVEN a single projects with two namespace and three authorization policies
// WHEN cleanupAuthorizationPoliciesForProjects is called
// THEN the cleanupAuthorizationPoliciesForProjects should return success and cleanup the authorization policy of
// the remaining authorization policies
func TestDeleteThreePoliciesTwoNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	err := cluv1alpha1.AddToScheme(scheme)
	assert.NoError(t, err, "Unexpected error adding to scheme")
	client := ctrlfake.NewFakeClientWithScheme(scheme)

	ap := &AuthorizationPolicy{
		Client:      client,
		KubeClient:  fake.NewSimpleClientset(),
		IstioClient: istiofake.NewSimpleClientset(),
	}

	// Create a project in the verrazzano-mc namespace with two namespaces
	project := &cluv1alpha1.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project",
			Namespace: "verrazzano-mc",
		},
		Spec: cluv1alpha1.VerrazzanoProjectSpec{
			Template: cluv1alpha1.ProjectTemplate{
				Namespaces: []cluv1alpha1.NamespaceTemplate{
					{Metadata: metav1.ObjectMeta{
						Name: "appconfig-namespace1",
					}},
					{Metadata: metav1.ObjectMeta{
						Name: "appconfig-namespace2",
					}},
				},
			},
			Placement: cluv1alpha1.Placement{
				Clusters: []cluv1alpha1.Cluster{
					{
						Name: constants.DefaultClusterName,
					},
				},
			},
		},
	}
	err = ap.Client.Create(context.TODO(), project)
	assert.NoError(t, err, "Unexpected error creating Verrazzano project")

	// Create a pod for appconfig-name1 in the project namespace appconfig-namespace1
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod1",
			Namespace: "appconfig-namespace1",
			Labels: map[string]string{
				IstioAppLabel: "appconfig-name1",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "appconfig-name1",
					Kind:       "ApplicationConfiguration",
					APIVersion: "core.oam.dev/v1alpha2",
				},
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "appconfig-name1",
		},
	}
	_, err = ap.KubeClient.CoreV1().Pods("appconfig-namespace1").Create(context.TODO(), pod, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	// Create an authorization policy for appconfig-name1 in the project namespace appconfig-namespace1
	authzPolicy := &clisecurity.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "appconfig-name1",
			Namespace: "appconfig-namespace1",
			Labels: map[string]string{
				IstioAppLabel: "appconfig-name1",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "appconfig-name1",
					Kind: "ApplicationConfiguration",
				},
			},
		},
		Spec: securityv1beta1.AuthorizationPolicy{
			Selector: &v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{
					IstioAppLabel: "appconfig-name1",
				},
			},
			Rules: []*securityv1beta1.Rule{
				{
					From: []*securityv1beta1.Rule_From{
						{
							Source: &securityv1beta1.Source{
								Principals: []string{
									"cluster.local/ns/appconfig-namespace1/sa/appconfig-name1",
									"cluster.local/ns/appconfig-namespace1/sa/appconfig-name2",
									"cluster.local/ns/appconfig-namespace2/sa/appconfig-name3",
									"cluster.local/ns/appconfig-namespace2/sa/random-sa",
									"cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account",
									"cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator",
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = ap.IstioClient.SecurityV1beta1().AuthorizationPolicies("appconfig-namespace1").Create(context.TODO(), authzPolicy, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating authorization policies")

	// Create a pod for appconfig-name2 in the project namespace appconfig-namespace1
	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod2",
			Namespace: "appconfig-namespace1",
			Labels: map[string]string{
				IstioAppLabel: "appconfig-name2",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "appconfig-name2",
					Kind:       "ApplicationConfiguration",
					APIVersion: "core.oam.dev/v1alpha2",
				},
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "appconfig-name2",
		},
	}
	_, err = ap.KubeClient.CoreV1().Pods("appconfig-namespace1").Create(context.TODO(), pod, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	// Create an authorization policy for appconfig-name2 in the project namespace appconfig-namespace1
	authzPolicy = &clisecurity.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "appconfig-name2",
			Namespace: "appconfig-namespace1",
			Labels: map[string]string{
				IstioAppLabel: "appconfig-name2",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "appconfig-name2",
					Kind: "ApplicationConfiguration",
				},
			},
		},
		Spec: securityv1beta1.AuthorizationPolicy{
			Selector: &v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{
					IstioAppLabel: "appconfig-name2",
				},
			},
			Rules: []*securityv1beta1.Rule{
				{
					From: []*securityv1beta1.Rule_From{
						{
							Source: &securityv1beta1.Source{
								Principals: []string{
									"cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account",
									"cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator",
									"cluster.local/ns/appconfig-namespace1/sa/appconfig-name1",
									"cluster.local/ns/appconfig-namespace1/sa/appconfig-name2",
									"cluster.local/ns/appconfig-namespace2/sa/appconfig-name3",
									"cluster.local/ns/appconfig-namespace2/sa/random-sa",
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = ap.IstioClient.SecurityV1beta1().AuthorizationPolicies("appconfig-namespace1").Create(context.TODO(), authzPolicy, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating authorization policies")

	// Create a pod for appconfig-name3 in the project namespace appconfig-namespace2
	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod3",
			Namespace: "appconfig-namespace2",
			Labels: map[string]string{
				IstioAppLabel: "appconfig-name3",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "appconfig-name3",
					Kind:       "ApplicationConfiguration",
					APIVersion: "core.oam.dev/v1alpha2",
				},
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "appconfig-name3",
		},
	}
	_, err = ap.KubeClient.CoreV1().Pods("appconfig-namespace2").Create(context.TODO(), pod, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating pod")

	// Create an authorization policy for appconfig-name3 in the project namespace appconfig-namespace2
	authzPolicy2 := &clisecurity.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "appconfig-name3",
			Namespace: "appconfig-namespace2",
			Labels: map[string]string{
				IstioAppLabel: "appconfig-name3",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "appconfig-name3",
					Kind: "ApplicationConfiguration",
				},
			},
		},
		Spec: securityv1beta1.AuthorizationPolicy{
			Selector: &v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{
					IstioAppLabel: "appconfig-name3",
				},
			},
			Rules: []*securityv1beta1.Rule{
				{
					From: []*securityv1beta1.Rule_From{
						{
							Source: &securityv1beta1.Source{
								Principals: []string{
									"cluster.local/ns/appconfig-namespace1/sa/appconfig-name1",
									"cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account",
									"cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator",
									"cluster.local/ns/appconfig-namespace1/sa/appconfig-name2",
									"cluster.local/ns/appconfig-namespace2/sa/appconfig-name3",
									"cluster.local/ns/appconfig-namespace2/sa/random-sa",
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = ap.IstioClient.SecurityV1beta1().AuthorizationPolicies("appconfig-namespace2").Create(context.TODO(), authzPolicy2, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating authorization policies")

	err = ap.cleanupAuthorizationPoliciesForProjects("appconfig-namespace1", "appconfig-name1", zap.S())
	assert.NoError(t, err, "Unexpected error cleaning up authorization policies")

	updatedPolicy, err := ap.IstioClient.SecurityV1beta1().AuthorizationPolicies("appconfig-namespace1").Get(context.TODO(), "appconfig-name2", metav1.GetOptions{})
	assert.NoError(t, err, "Unexpected error getting authorization policies")
	assert.Equal(t, len(updatedPolicy.Spec.Rules[0].From[0].Source.Principals), 5)
	assert.Contains(t, updatedPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account")
	assert.Contains(t, updatedPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/appconfig-namespace1/sa/appconfig-name2")
	assert.Contains(t, updatedPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator")
	assert.Contains(t, updatedPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/appconfig-namespace2/sa/appconfig-name3")
	assert.Contains(t, updatedPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/appconfig-namespace2/sa/random-sa")

	updatedPolicy, err = ap.IstioClient.SecurityV1beta1().AuthorizationPolicies("appconfig-namespace2").Get(context.TODO(), "appconfig-name3", metav1.GetOptions{})
	assert.NoError(t, err, "Unexpected error getting authorization policies")
	assert.Equal(t, len(updatedPolicy.Spec.Rules[0].From[0].Source.Principals), 5)
	assert.Contains(t, updatedPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account")
	assert.Contains(t, updatedPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/appconfig-namespace1/sa/appconfig-name2")
	assert.Contains(t, updatedPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator")
	assert.Contains(t, updatedPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/appconfig-namespace2/sa/appconfig-name3")
	assert.Contains(t, updatedPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/appconfig-namespace2/sa/random-sa")
}
