// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"encoding/json"

	v1alpha12 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var testManagedCluster = v1alpha1.VerrazzanoManagedCluster{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-managed-cluster-name",
		Namespace: constants.VerrazzanoMultiClusterNamespace,
	},
	Spec: v1alpha1.VerrazzanoManagedClusterSpec{
		CASecret:                     "test-secret",
		ManagedClusterManifestSecret: "test-cluster-manifest-secret",
		ServiceAccount:               "test-service-account",
	},
}

var testProject = v1alpha12.VerrazzanoProject{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: constants.VerrazzanoMultiClusterNamespace,
	},
	Spec: v1alpha12.VerrazzanoProjectSpec{
		Placement: v1alpha12.Placement{
			Clusters: []v1alpha12.Cluster{{Name: "test-managed-cluster-name"}},
		},
		Template: v1alpha12.ProjectTemplate{
			Namespaces: []v1alpha12.NamespaceTemplate{
				{
					Metadata: metav1.ObjectMeta{
						Name: "newNS1",
					},
				},
			},
		},
	},
}

var testNetworkPolicy = v1alpha12.VerrazzanoProject{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: constants.VerrazzanoMultiClusterNamespace,
	},
	Spec: v1alpha12.VerrazzanoProjectSpec{
		Placement: v1alpha12.Placement{
			Clusters: []v1alpha12.Cluster{{Name: "test-managed-cluster-name"}},
		},
		Template: v1alpha12.ProjectTemplate{
			Namespaces: []v1alpha12.NamespaceTemplate{
				{
					Metadata: metav1.ObjectMeta{
						Name: "ns1",
					},
				},
			},
			NetworkPolicies: []v1alpha12.NetworkPolicyTemplate{
				{
					Metadata: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "net1",
					},
					Spec: netv1.NetworkPolicySpec{},
				}},
		},
	},
}

// newAdmissionRequest creates a new admissionRequest with the provided operation and object.
// This is a test utility function used by other multi-cluster resource validation tests.
func newAdmissionRequest(op admissionv1.Operation, obj interface{}) admission.Request {
	raw := runtime.RawExtension{}
	bytes, _ := json.Marshal(obj)
	raw.Raw = bytes
	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Operation: op, Object: raw}}
	return req
}

// newScheme creates a new scheme that includes this package's object for use by client
// This is a test utility function used by other multi-cluster resource validation tests.
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = v1alpha12.AddToScheme(scheme)
	scheme.AddKnownTypes(schema.GroupVersion{
		Version: "v1",
	}, &corev1.Secret{}, &corev1.SecretList{})
	_ = v1alpha1.AddToScheme(scheme)
	return scheme
}
