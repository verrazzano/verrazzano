// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanagerocidns

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
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

// Fake certManagerComponent resource for function calls
var fakeComponent = certManagerOciDNSComponent{}

// TestIsCertManagerOciDNSEnabled tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
// WHEN cert-manager is enabled
// THEN the function returns true
//func TestIsCertManagerOciDNSEnabled(t *testing.T) {
//	localvz := vz.DeepCopy()
//	bt := true
//	localvz.Spec.Components.CertManager.Enabled = &bt
//	localvz.Spec.Components.DNS = &vzapi.DNSComponent{OCI: &vzapi.OCI{}}
//	assert.True(t, fakeComponent.IsEnabled(localvz))
//}

// TestIsCertManagerDisabled tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
// WHEN cert-manager is disabled
// THEN the function returns false
func TestIsCertManagerOciDNSDisabled(t *testing.T) {
	localvz := vz.DeepCopy()
	bf := false
	localvz.Spec.Components.CertManager.Enabled = &bf
	assert.False(t, fakeComponent.IsEnabled(localvz))
}

// TestCertManagerPreInstall tests the PreInstall fn
// GIVEN a call to this fn
// WHEN I call PreInstall with dry-run = true
// THEN no errors are returned
func TestCertManagerOciDNSPreInstallDryRun(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, true))
	assert.NoError(t, err)
}

// TestIsCertManagerReadyOciDNS tests the IsReady function with OCI-DNS enabled
// GIVEN a call to IsReady
// WHEN the deployment object has enough replicas available
// THEN true is returned
func TestIsCertManagerOciDNSReady(t *testing.T) {
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

// TestIsCertManagerNotReady tests the isCertManagerReady function
// GIVEN a call to isCertManagerReady
// WHEN the deployment object does not have enough replicas available
// THEN false is returned
func TestIsCertManagerOciDNSNotReady(t *testing.T) {
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

// TestIsCertManagerNotReady tests the isCertManagerReady function
// GIVEN a call to isCertManagerReady
// WHEN the deployment object does not have enough replicas available
// THEN false is returned
func TestIsCertManagerOciDNSReadyDisabled(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).
		WithObjects(newDeployment(ocidnsDeploymentName, false)).
		Build()
	vz := &vzapi.Verrazzano{}
	assert.True(t, isCertManagerOciDNSReady(spi.NewFakeContext(client, vz, nil, false)))
}

// TestPostInstallAcme tests the PostInstall function
// GIVEN a call to PostInstall
//
//	WHEN the cert type is Acme
//	THEN no error is returned
func TestPostInstallAcme(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	// set OCI DNS secret value and create secret
	localvz.Spec.Components.DNS = &vzapi.DNSComponent{
		OCI: &vzapi.OCI{
			OCIConfigSecret: "ociDNSSecret",
			DNSZoneName:     "example.dns.io",
		},
	}
	client.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ociDNSSecret",
			Namespace: ComponentNamespace,
		},
	})
	err := fakeComponent.PostInstall(spi.NewFakeContext(client, localvz, nil, false))
	assert.NoError(t, err)
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
