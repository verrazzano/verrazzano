// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// TestVzResolveNamespace tests the Verrazzano component name
// GIVEN a Verrazzano component
//  WHEN I call resolveNamespace
//  THEN the Verrazzano namespace name is correctly resolved
func TestVzResolveNamespace(t *testing.T) {
	const defNs = constants.VerrazzanoSystemNamespace
	assert := assert.New(t)
	ns := ResolveVerrazzanoNamespace("")
	assert.Equal(defNs, ns, "Wrong namespace resolved for Verrazzano when using empty namespace")
	ns = ResolveVerrazzanoNamespace("default")
	assert.Equal(defNs, ns, "Wrong namespace resolved for Verrazzano when using default namespace")
	ns = ResolveVerrazzanoNamespace("custom")
	assert.Equal("custom", ns, "Wrong namespace resolved for Verrazzano when using custom namesapce")
}

// TestFixupFluentdDaemonset tests calls to fixupFluentdDaemonset
func TestFixupFluentdDaemonset(t *testing.T) {
	const defNs = constants.VerrazzanoSystemNamespace
	assert := assert.New(t)
	scheme := newFakeRuntimeScheme()
	client := fake.NewFakeClientWithScheme(scheme)
	logger, _ := zap.NewProduction()
	log := logger.Sugar()

	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defNs,
		},
	}
	err := client.Create(context.TODO(), &ns)
	assert.NoError(err)

	// Should return with no error since the fluentd daemonset does not exist.
	// This is valid case when fluentd is not installed.
	err = fixupFluentdDaemonset(log, client, defNs)
	assert.NoError(err)

	// Create a fluentd daemonset for test purposes
	daemonSet := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defNs,
			Name:      "fluentd",
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "wrong-name",
							Env: []corev1.EnvVar{
								{
									Name:  constants.ClusterNameEnvVar,
									Value: "managed1",
								},
								{
									Name:  constants.ElasticsearchURLEnvVar,
									Value: "some-url",
								},
							},
						},
					},
				},
			},
		},
	}
	err = client.Create(context.TODO(), &daemonSet)
	assert.NoError(err)

	// should return error that fluentd container is missing
	err = fixupFluentdDaemonset(log, client, defNs)
	assert.EqualError(err, "fluentd container not found in fluentd daemonset: fluentd")

	daemonSet.Spec.Template.Spec.Containers[0].Name = "fluentd"
	err = client.Update(context.TODO(), &daemonSet)
	assert.NoError(err)

	// should return no error since the env variables don't need fixing up
	err = fixupFluentdDaemonset(log, client, defNs)
	assert.NoError(err)

	// create a secret with needed keys
	data := make(map[string][]byte)
	data[constants.ClusterNameData] = []byte("managed1")
	data[constants.ElasticsearchURLData] = []byte("some-url")
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defNs,
			Name:      constants.MCRegistrationSecret,
		},
		Data: data,
	}
	err = client.Create(context.TODO(), &secret)
	assert.NoError(err)

	// Update env variables to use ValueFrom instead of Value
	clusterNameRef := corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: constants.MCRegistrationSecret,
			},
			Key: constants.ClusterNameData,
		},
	}
	esURLRef := corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: constants.MCRegistrationSecret,
			},
			Key: constants.ElasticsearchURLData,
		},
	}
	daemonSet.Spec.Template.Spec.Containers[0].Env[0].Value = ""
	daemonSet.Spec.Template.Spec.Containers[0].Env[0].ValueFrom = &clusterNameRef
	daemonSet.Spec.Template.Spec.Containers[0].Env[1].Value = ""
	daemonSet.Spec.Template.Spec.Containers[0].Env[1].ValueFrom = &esURLRef
	err = client.Update(context.TODO(), &daemonSet)
	assert.NoError(err)

	// should return no error
	err = fixupFluentdDaemonset(log, client, defNs)
	assert.NoError(err)

	// env variables should be fixed up to use Value instead of ValueFrom
	fluentdNamespacedName := types.NamespacedName{Name: "fluentd", Namespace: defNs}
	updatedDaemonSet := appsv1.DaemonSet{}
	err = client.Get(context.TODO(), fluentdNamespacedName, &updatedDaemonSet)
	assert.NoError(err)
	assert.Equal("managed1", updatedDaemonSet.Spec.Template.Spec.Containers[0].Env[0].Value)
	assert.Nil(updatedDaemonSet.Spec.Template.Spec.Containers[0].Env[0].ValueFrom)
	assert.Equal("some-url", updatedDaemonSet.Spec.Template.Spec.Containers[0].Env[1].Value)
	assert.Nil(updatedDaemonSet.Spec.Template.Spec.Containers[0].Env[1].ValueFrom)
}

const testBomFilePath = "../../testdata/test_bom.json"

// Test_appendOverrides tests the AppendOverrides function
// GIVEN a call to AppendOverrides
//  WHEN I call with no extra kvs
//  THEN the correct KeyValue objects are returned and no error occurs
func Test_appendOverrides(t *testing.T) {
	assert := assert.New(t)
	config.SetDefaultBomFilePath(testBomFilePath)

	kvs, err := AppendOverrides(spi.NewFakeContext(nil, nil, false), "", "", "", []bom.KeyValue{})

	assert.NoError(err)
	assert.Len(kvs, 2)
	assert.Contains(kvs, bom.KeyValue{
		Key:   "monitoringOperator.prometheusInitImage",
		Value: "ghcr.io/oracle/oraclelinux:7-slim",
	})
	assert.Contains(kvs, bom.KeyValue{
		Key:   "monitoringOperator.esInitImage",
		Value: "ghcr.io/oracle/oraclelinux:7.8",
	})
}

// Test_fixupElasticSearchReplicaCount tests the fixupElasticSearchReplicaCount function.
// GIVEN
// WHEN
// THEN
//func Test_fixupElasticSearchReplicaCount(t *testing.T) {
//	assert := assert.New(t)
//	context, err := createFakeComponentContext()
//	assert.NoError(err, "Failed to create fake component context.")
//	err = fixupElasticSearchReplicaCount(context, "test-namespace-name")
//	assert.NoError(err, "Failed to fixup Elasticsearch index template")
//}

// Test_getNamedContainerPortOfContainer tests the getNamedContainerPortOfContainer function.
// GIVEN
// WHEN
// THEN
//func Test_getNamedContainerPortOfContainer(t *testing.T) {
//	assert := assert.New(t)
//	pod := corev1.Pod{}
//	port, err := getNamedContainerPortOfContainer(pod, "test-container-name", "test-port-name")
//	assert.NoError(err, "Failed to find fake container port")
//	assert.Equal(42, port, "Expected to find valid named container port")
//}

// Test_getPodsWithReadyContainer tests the getPodsWithReadyContainer function.
// GIVEN
// WHEN
// THEN
//func Test_getPodsWithReadyContainer(t *testing.T) {
//	assert := assert.New(t)
//	context, err := createFakeComponentContext()
//	assert.NoError(err, "Failed to create fake component context.")
//	pods, err := getPodsWithReadyContainer(context.Client(), "test-container-name", client.InNamespace("test-namespace-name"), client.MatchingLabels{"test-label-name": "test-label-value"})
//	assert.NoError(err, "Failed to find fake pod with ready container")
//	assert.Len(pods, 1, "Expected to find one pod with a ready container")
//}

// Test_waitForPodsWithReadyContainer tests the waitForPodsWithReadyContainer function.
// GIVEN
// WHEN
// THEN
//func Test_waitForPodsWithReadyContainer(t *testing.T) {
//	assert := assert.New(t)
//	context, err := createFakeComponentContext()
//	assert.NoError(err, "Failed to create fake component context.")
//	pods, err := waitForPodsWithReadyContainer(context.Client(), 1*time.Second, 5*time.Second, "test-container-name", client.InNamespace("test-namespace-name"), client.MatchingLabels{"test-label-name": "test-label-value"})
//	assert.NoError(err, "Failed to find fake pod with ready container")
//	assert.Len(pods, 1, "Expected to find one pod with a ready container")
//}

// newFakeRuntimeScheme creates a new fake scheme
func newFakeRuntimeScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	appsv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	return scheme
}

// createFakeComponentContext creates a fake component context
//func createFakeComponentContext() (spi.ComponentContext, error) {
//	client := fake.NewFakeClientWithScheme(newFakeRuntimeScheme())
//	logger, _ := zap.NewProduction()
//	return spi.NewContext(logger.Sugar(), client, nil, false)
//}

