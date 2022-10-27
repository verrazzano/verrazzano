// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appoper

import (
	"context"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	oam "github.com/crossplane/oam-kubernetes-runtime/apis/core"
	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testBomFilePath = "../../testdata/test_bom.json"

// TestAppendAppOperatorOverrides tests the Keycloak override for the theme images
// GIVEN an env override for the app operator image
//
//	WHEN I call AppendApplicationOperatorOverrides
//	THEN the "image" Key is set with the image override.
func TestAppendAppOperatorOverrides(t *testing.T) {
	a := assert.New(t)

	config.SetDefaultBomFilePath(testBomFilePath)

	const expectedFluentdImage = "ghcr.io/verrazzano/fluentd-kubernetes-daemonset:v1.12.3-20210517195222-f345ec2"
	const expectedIstioProxyImage = "ghcr.io/verrazzano/proxyv2:1.7.3"
	const expectedWeblogicMonitoringExporterImage = "ghcr.io/oracle/weblogic-monitoring-exporter:2.0.4"

	kvs, err := AppendApplicationOperatorOverrides(nil, "", "", "", nil)
	a.NoError(err, "AppendApplicationOperatorOverrides returned an error ")
	a.Len(kvs, 3, "AppendApplicationOperatorOverrides returned an unexpected number of Key:Value pairs")
	a.Equalf("fluentdImage", kvs[0].Key, "Did not get expected fluentdImage Key")
	a.Equalf(expectedFluentdImage, kvs[0].Value, "Did not get expected fluentdImage Value")
	a.Equalf("istioProxyImage", kvs[1].Key, "Did not get expected istioProxyImage Key")
	a.Equalf(expectedIstioProxyImage, kvs[1].Value, "Did not get expected istioProxyImage Value")
	a.Equalf("weblogicMonitoringExporterImage", kvs[2].Key, "Did not get expected weblogicMonitoringExporterImage Key")
	a.Equalf(expectedWeblogicMonitoringExporterImage, kvs[2].Value, "Did not get expected weblogicMonitoringExporterImage Value")

	customImage := "myreg.io/myrepo/v8o/verrazzano-application-operator-dev:local-20210707002801-b7449154"
	_ = os.Setenv(constants.VerrazzanoAppOperatorImageEnvVar, customImage)
	defer func() { _ = os.Unsetenv(constants.RegistryOverrideEnvVar) }()

	kvs, err = AppendApplicationOperatorOverrides(nil, "", "", "", nil)
	a.NoError(err, "AppendApplicationOperatorOverrides returned an error ")
	a.Len(kvs, 4, "AppendApplicationOperatorOverrides returned wrong number of Key:Value pairs")
	a.Equalf("image", kvs[0].Key, "Did not get expected image Key")
	a.Equalf(customImage, kvs[0].Value, "Did not get expected image Value")
	a.Equalf("fluentdImage", kvs[1].Key, "Did not get expected fluentdImage Key")
	a.Equalf(expectedFluentdImage, kvs[1].Value, "Did not get expected fluentdImage Value")
	a.Equalf("istioProxyImage", kvs[2].Key, "Did not get expected istioProxyImage Key")
	a.Equalf(expectedIstioProxyImage, kvs[2].Value, "Did not get expected istioProxyImage Value")
	a.Equalf("weblogicMonitoringExporterImage", kvs[3].Key, "Did not get expected weblogicMonitoringExporterImage Key")
	a.Equalf(expectedWeblogicMonitoringExporterImage, kvs[3].Value, "Did not get expected weblogicMonitoringExporterImage Value")
}

// TestIsApplicationOperatorReady tests the isApplicationOperatorReady function
// GIVEN a call to isApplicationOperatorReady
//
//	WHEN the deployment object has enough replicas available
//	THEN true is returned
func TestIsApplicationOperatorReady(t *testing.T) {

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.VerrazzanoSystemNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"app": ComponentName},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": ComponentName},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.VerrazzanoSystemNamespace,
				Name:      ComponentName + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"app":               ComponentName,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   constants.VerrazzanoSystemNamespace,
				Name:        ComponentName + "-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
	).Build()
	assert.True(t, isApplicationOperatorReady(spi.NewFakeContext(fakeClient, nil, nil, false)))
}

// TestIsApplicationOperatorNotReady tests the isApplicationOperatorReady function
// GIVEN a call to isApplicationOperatorReady
//
//	WHEN the deployment object does NOT have enough replicas available
//	THEN false is returned
func TestIsApplicationOperatorNotReady(t *testing.T) {

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      "verrazzano-application-operator",
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   0,
		},
	}).Build()
	assert.False(t, isApplicationOperatorReady(spi.NewFakeContext(fakeClient, nil, nil, false)))
}

// TestLabelAnnotateTraitDefinitions tests the labelAnnotateTraitDefinitions function
// GIVEN a call to labelAnnotateTraitDefinitions
// WHEN trait definitions do not have expected Helm label/annotations
// THEN the trait definitions are updated with the expected Helm label/annotations
func TestLabelAnnotateTraitDefinitions(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = oam.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		testTraitObjects()...,
	).Build()
	assert.NoError(t, labelAnnotateTraitDefinitions(fakeClient))
	trait := oamv1alpha2.TraitDefinition{}
	assert.NoError(t, fakeClient.Get(context.TODO(), types.NamespacedName{Name: "ingresstraits.oam.verrazzano.io"}, &trait))
	checkTraitDefinition(t, &trait)
	trait = oamv1alpha2.TraitDefinition{}
	assert.NoError(t, fakeClient.Get(context.TODO(), types.NamespacedName{Name: "loggingtraits.oam.verrazzano.io"}, &trait))
	checkTraitDefinition(t, &trait)
	trait = oamv1alpha2.TraitDefinition{}
	assert.NoError(t, fakeClient.Get(context.TODO(), types.NamespacedName{Name: "metricstraits.oam.verrazzano.io"}, &trait))
	checkTraitDefinition(t, &trait)
}

func testTraitObjects() []client.Object {
	return []client.Object{
		&oamv1alpha2.TraitDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ingresstraits.oam.verrazzano.io",
			},
		},
		&oamv1alpha2.TraitDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "loggingtraits.oam.verrazzano.io",
			},
		},
		&oamv1alpha2.TraitDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "metricstraits.oam.verrazzano.io",
			},
		},
	}
}

// TestLabelAnnotateWorkloadDefinitions tests the labelAnnotateWorkloadDefinitions function
// GIVEN a call to labelAnnotateWorkloadDefinitions
// WHEN workload definitions do not have expected Helm label/annotations
// THEN the workload definitions are updated with the expected Helm label/annotations
func TestLabelAnnotateWorkloadDefinitions(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = oam.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		testWorkloadDefinitionObjects()...,
	).Build()
	assert.NoError(t, labelAnnotateWorkloadDefinitions(fakeClient))
	workload := oamv1alpha2.WorkloadDefinition{}
	assert.NoError(t, fakeClient.Get(context.TODO(), types.NamespacedName{Name: "coherences.coherence.oracle.com"}, &workload))
	checkWorkloadDefinition(t, &workload)
	workload = oamv1alpha2.WorkloadDefinition{}
	assert.NoError(t, fakeClient.Get(context.TODO(), types.NamespacedName{Name: "deployments.apps"}, &workload))
	checkWorkloadDefinition(t, &workload)
	workload = oamv1alpha2.WorkloadDefinition{}
	assert.NoError(t, fakeClient.Get(context.TODO(), types.NamespacedName{Name: "domains.weblogic.oracle"}, &workload))
	checkWorkloadDefinition(t, &workload)
	workload = oamv1alpha2.WorkloadDefinition{}
	assert.NoError(t, fakeClient.Get(context.TODO(), types.NamespacedName{Name: "verrazzanocoherenceworkloads.oam.verrazzano.io"}, &workload))
	checkWorkloadDefinition(t, &workload)
	workload = oamv1alpha2.WorkloadDefinition{}
	assert.NoError(t, fakeClient.Get(context.TODO(), types.NamespacedName{Name: "verrazzanohelidonworkloads.oam.verrazzano.io"}, &workload))
	checkWorkloadDefinition(t, &workload)
	workload = oamv1alpha2.WorkloadDefinition{}
	assert.NoError(t, fakeClient.Get(context.TODO(), types.NamespacedName{Name: "verrazzanoweblogicworkloads.oam.verrazzano.io"}, &workload))
	checkWorkloadDefinition(t, &workload)
}

func testWorkloadDefinitionObjects() []client.Object {
	return []client.Object{
		&oamv1alpha2.WorkloadDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "coherences.coherence.oracle.com",
			},
		},
		&oamv1alpha2.WorkloadDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "deployments.apps",
			},
		},
		&oamv1alpha2.WorkloadDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "domains.weblogic.oracle",
			},
		},
		&oamv1alpha2.WorkloadDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "verrazzanocoherenceworkloads.oam.verrazzano.io",
			},
		},
		&oamv1alpha2.WorkloadDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "verrazzanohelidonworkloads.oam.verrazzano.io",
			},
		},
		&oamv1alpha2.WorkloadDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "verrazzanoweblogicworkloads.oam.verrazzano.io",
			},
		},
	}
}

func checkTraitDefinition(t *testing.T, trait *oamv1alpha2.TraitDefinition) {
	assert.Contains(t, trait.Labels[helmManagedByLabel], "Helm")
	assert.Contains(t, trait.Annotations[helmReleaseNameAnnotation], ComponentName)
	assert.Contains(t, trait.Annotations[helmReleaseNamespaceAnnotation], ComponentNamespace)
}

func checkWorkloadDefinition(t *testing.T, trait *oamv1alpha2.WorkloadDefinition) {
	assert.Contains(t, trait.Labels[helmManagedByLabel], "Helm")
	assert.Contains(t, trait.Annotations[helmReleaseNameAnnotation], ComponentName)
	assert.Contains(t, trait.Annotations[helmReleaseNamespaceAnnotation], ComponentNamespace)
}
