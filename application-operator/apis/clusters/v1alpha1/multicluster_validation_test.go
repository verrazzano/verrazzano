// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"encoding/json"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
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
		PrometheusSecret:             "test-prometheus-secret",
		ManagedClusterManifestSecret: "test-cluster-manifest-secret",
		ServiceAccount:               "test-service-account",
	},
}

var testProject = VerrazzanoProject{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: constants.VerrazzanoMultiClusterNamespace,
	},
	Spec: VerrazzanoProjectSpec{
		Placement: Placement{
			Clusters: []Cluster{{Name: "test-managed-cluster-name"}},
		},
		Template: ProjectTemplate{
			Namespaces: []NamespaceTemplate{
				{
					Metadata: metav1.ObjectMeta{
						Name: "newNS1",
					},
				},
			},
		},
	},
}

var testNetworkPolicy = VerrazzanoProject{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: constants.VerrazzanoMultiClusterNamespace,
	},
	Spec: VerrazzanoProjectSpec{
		Placement: Placement{
			Clusters: []Cluster{{Name: "test-managed-cluster-name"}},
		},
		Template: ProjectTemplate{
			Namespaces: []NamespaceTemplate{
				{
					Metadata: metav1.ObjectMeta{
						Name: "ns1",
					},
				},
			},
			NetworkPolicies: []NetworkPolicyTemplate{
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
func newAdmissionRequest(op admissionv1beta1.Operation, obj interface{}) admission.Request {
	raw := runtime.RawExtension{}
	bytes, _ := json.Marshal(obj)
	raw.Raw = bytes
	req := admission.Request{
		AdmissionRequest: admissionv1beta1.AdmissionRequest{
			Operation: op, Object: raw}}
	return req
}

// newScheme creates a new scheme that includes this package's object for use by client
// This is a test utility function used by other multi-cluster resource validation tests.
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	AddToScheme(scheme)
	scheme.AddKnownTypes(schema.GroupVersion{
		Version: "v1",
	}, &corev1.Secret{})
	v1alpha1.AddToScheme(scheme)
	return scheme
}
