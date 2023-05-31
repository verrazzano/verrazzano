// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhookoci

import (
	"testing"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Default Acme object
var acme = vzapi.Acme{
	Provider:     "testProvider",
	EmailAddress: "testEmail",
	Environment:  "myenv",
}

// Default Verrazzano object
var vz = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		EnvironmentName: "myenv",
		Components: vzapi.ComponentSpec{
			CertManager: &vzapi.CertManagerComponent{
				Certificate: vzapi.Certificate{},
			},
		},
	},
}

var testScheme = runtime.NewScheme()

func init() {
	_ = k8scheme.AddToScheme(testScheme)
	_ = certv1.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
	_ = apiextv1.AddToScheme(testScheme)
}

// Test_isCertManagerOciDNSReady tests the IsReady function with OCI-DNS enabled
// GIVEN a call to IsReady
// WHEN the deployment object has enough replicas available
// THEN true is returned
func Test_isCertManagerOciDNSReady(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).
		WithObjects(newDeployment(ocidnsDeploymentName, false)).
		Build()

	vz := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						OCIConfigSecret: "oci",
					},
				},
			},
		},
	}
	assert.False(t, isCertManagerOciDNSReady(spi.NewFakeContext(client, &vz, nil, false)))

	client = fake.NewClientBuilder().WithScheme(k8scheme.Scheme).
		WithObjects(
			newDeployment(ocidnsDeploymentName, true),
			newPod(ComponentNamespace, "cert-manager-ocidns-provider"),
			newReplicaSet(ComponentNamespace, "cert-manager-ocidns-provider"),
		).
		Build()
	assert.True(t, isCertManagerOciDNSReady(spi.NewFakeContext(client, &vz, nil, false)))
}

// Test_isCertManagerOciDNSNotReady tests the isCertManagerReady function
// GIVEN a call to isCertManagerReady
// WHEN the deployment object does not have enough replicas available
// THEN false is returned
func Test_isCertManagerOciDNSNotReady(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).
		WithObjects(newDeployment(ocidnsDeploymentName, false)).
		Build()
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						OCIConfigSecret: "oci",
					},
				},
			},
		},
	}
	assert.False(t, isCertManagerOciDNSReady(spi.NewFakeContext(client, vz, nil, false)))
}

// Test_isCertManagerOciDNSReadyDisabled tests the isCertManagerReady function
// GIVEN a call to isCertManagerReady
// WHEN the deployment object does not have enough replicas available
// THEN false is returned
func Test_isCertManagerOciDNSReadyDisabled(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).
		WithObjects(newDeployment(ocidnsDeploymentName, false)).
		Build()
	vz := &vzapi.Verrazzano{}
	assert.True(t, isCertManagerOciDNSReady(spi.NewFakeContext(client, vz, nil, false)))
}

// Create a new deployment object for testing
func newDeployment(name string, ready bool) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			UpdatedReplicas:     1,
			ReadyReplicas:       1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	}

	if !ready {
		deployment.Status = appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       0,
			AvailableReplicas:   0,
			UnavailableReplicas: 1,
		}
	}
	return deployment
}

func newPod(namespace string, name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name + "-95d8c5d96-m6mbr",
			Labels: map[string]string{
				"pod-template-hash": "95d8c5d96",
				"app":               name,
			},
		},
	}
}

func newReplicaSet(namespace string, name string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        name + "-95d8c5d96",
			Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
		},
	}
}
