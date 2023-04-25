// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	bootstrapOcneProvider     = "bootstrap-ocne"
	controlPlaneOcneProvider  = "control-plane-ocne"
	clusterAPIProvider        = "cluster-api"
	infrastructureOciProvider = "infrastructure-oci"

	deploymentRevisionAnnotation = "deployment.kubernetes.io/revision"
	podTemplateHashLabel         = "pod-template-hash"
	providerLabel                = "cluster.x-k8s.io/provider"
)

func fakeCAPINew(_ string, _ ...client.Option) (client.Client, error) {
	return &FakeCAPIClient{}, nil
}

// TestIsReady tests the IsReady function
// GIVEN a call to IsReady
//
//	WHEN the deployment object has enough replicas available
//	THEN true is returned
func TestIsReady(t *testing.T) {
	fakeClient := getReadyDeployments().Build()
	var comp capiComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	assert.True(t, comp.IsReady(compContext))
}

// TestIsNotReady tests the IsReady function
// GIVEN a call to IsReady
//
//	WHEN the deployment object does not have enough replicas available
//	THEN false is returned
func TestIsNotReady(t *testing.T) {
	fakeClient := getNotReadyDeployments().Build()
	var comp capiComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	assert.False(t, comp.IsReady(compContext))
}

// TestIsAvailable tests the IsAvailable function
// GIVEN a call to IsAvailable
//
//	WHEN deployments are available
//	THEN true is returned
func TestIsAvailable(t *testing.T) {
	fakeClient := getReadyDeployments().Build()
	var comp capiComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	reason, _ := comp.IsAvailable(compContext)
	assert.Equal(t, "", reason)
}

// TestIsNotAvailable tests the IsAvailable function
// GIVEN a call to IsAvailable
//
//	WHEN deployments are not available
//	THEN false is returned
func TestIsNotAvailable(t *testing.T) {
	fakeClient := getNotReadyDeployments().Build()
	var comp capiComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	reason, _ := comp.IsAvailable(compContext)
	assert.Equal(t, "deployment verrazzano-capi/capi-controller-manager not available: 0/1 replicas ready", reason)
}

// TestIsInstalled tests the IsInstalled function
// GIVEN a call to IsInstalled
//
//	WHEN CAPI is installed
//	THEN true is returned
func TestIsInstalled(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: VerrazzanoCAPINamespace,
				Name:      capiCMDeployment,
			},
		}).Build()
	var comp capiComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	installed, err := comp.IsInstalled(compContext)
	assert.NoError(t, err)
	assert.True(t, installed)
}

// TestIsNotInstalled tests the IsInstalled function
// GIVEN a call to IsInstalled
//
//	WHEN CAPI is not installed
//	THEN false is returned
func TestIsNotInstalled(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()
	var comp capiComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	installed, err := comp.IsInstalled(compContext)
	assert.NoError(t, err)
	assert.False(t, installed)
}

// TestInstall tests the Install function
// GIVEN a call to Install
//
//	WHEN CAPI is installed
//	THEN no error is returned
func TestInstall(t *testing.T) {
	SetCAPIInitFunc(fakeCAPINew)
	defer ResetCAPIInitFunc()

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()
	var comp capiComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	err := comp.Install(compContext)
	assert.NoError(t, err)
}

// TestUninstall tests the Uninstall function
// GIVEN a call to Uninstall
//
//	WHEN CAPI is Uninstalled
//	THEN no error is returned
func TestUninstall(t *testing.T) {
	SetCAPIInitFunc(fakeCAPINew)
	defer ResetCAPIInitFunc()

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()
	var comp capiComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	err := comp.Uninstall(compContext)
	assert.NoError(t, err)
}

func getNotReadyDeployments() *fake.ClientBuilder {
	return fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: VerrazzanoCAPINamespace,
				Name:      capiCMDeployment,
				Labels:    map[string]string{providerLabel: clusterAPIProvider},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{providerLabel: clusterAPIProvider},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: VerrazzanoCAPINamespace,
				Name:      capiCMDeployment + "-95d8c5d97-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d97",
					providerLabel:        clusterAPIProvider,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   VerrazzanoCAPINamespace,
				Name:        capiCMDeployment + "-95d8c5d97",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},
	)
}

func getReadyDeployments() *fake.ClientBuilder {
	return fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: VerrazzanoCAPINamespace,
				Name:      capiCMDeployment,
				Labels:    map[string]string{providerLabel: clusterAPIProvider},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{providerLabel: clusterAPIProvider},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				ReadyReplicas:     1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: VerrazzanoCAPINamespace,
				Name:      capiCMDeployment + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d96",
					providerLabel:        clusterAPIProvider,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   VerrazzanoCAPINamespace,
				Name:        capiCMDeployment + "-95d8c5d96",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},

		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: VerrazzanoCAPINamespace,
				Name:      capiOcneBootstrapCMDeployment,
				Labels:    map[string]string{providerLabel: bootstrapOcneProvider},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{providerLabel: bootstrapOcneProvider},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				ReadyReplicas:     1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: VerrazzanoCAPINamespace,
				Name:      capiOcneBootstrapCMDeployment + "-95d8c5d93-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d93",
					providerLabel:        bootstrapOcneProvider,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   VerrazzanoCAPINamespace,
				Name:        capiOcneBootstrapCMDeployment + "-95d8c5d93",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},

		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: VerrazzanoCAPINamespace,
				Name:      capiOcneControlPlaneCMDeployment,
				Labels:    map[string]string{providerLabel: controlPlaneOcneProvider},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{providerLabel: controlPlaneOcneProvider},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				ReadyReplicas:     1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: VerrazzanoCAPINamespace,
				Name:      capiOcneControlPlaneCMDeployment + "-95d8c5d92-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d92",
					providerLabel:        controlPlaneOcneProvider,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   VerrazzanoCAPINamespace,
				Name:        capiOcneControlPlaneCMDeployment + "-95d8c5d92",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},

		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: VerrazzanoCAPINamespace,
				Name:      capiociCMDeployment,
				Labels:    map[string]string{providerLabel: infrastructureOciProvider},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{providerLabel: infrastructureOciProvider},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				ReadyReplicas:     1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: VerrazzanoCAPINamespace,
				Name:      capiociCMDeployment + "-95d8c5d91-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d91",
					providerLabel:        infrastructureOciProvider,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   VerrazzanoCAPINamespace,
				Name:        capiociCMDeployment + "-95d8c5d91",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},
	)
}
