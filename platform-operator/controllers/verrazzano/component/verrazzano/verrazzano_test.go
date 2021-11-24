// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
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
func Test_fixupElasticSearchReplicaCount(t *testing.T) {
	assert := assert.New(t)
	context, err := createFakeComponentContext()
	assert.NoError(err, "Failed to create fake component context.")
	createElasticsearchPod(context.Client(), "http")
	execCommand = fakeExecCommand
	err = fixupElasticSearchReplicaCount(context, "verrazzano-system")
	assert.NoError(err, "Failed to fixup Elasticsearch index template")
	// Error should be returned if there is no http port for the elasticsearch pod
	context1, err := createFakeComponentContext()
	assert.NoError(err, "Failed to create fake component context.")
	createElasticsearchPod(context1.Client(), "tcp")
	err = fixupElasticSearchReplicaCount(context1, "verrazzano-system")
	assert.Error(err, "Error should be returned if there is no http port for elasticsearch pods")
	// Change source version to 1.1.0
	context.ActualCR().Spec.Version = "1.1.0"
	err = fixupElasticSearchReplicaCount(context, "verrazzano-system")
	assert.NoError(err, "No error should be returned if the source version is 1.1.0 or later")
	// Disable Elasticsearch and no error should be returned
	falseValue := false
	context.EffectiveCR().Spec.Components.Elasticsearch.Enabled = &falseValue
	err = fixupElasticSearchReplicaCount(context, "verrazzano-system")
	assert.NoError(err, "No error should be returned if the elasticsearch is not enabled")

}

// Test_getNamedContainerPortOfContainer tests the getNamedContainerPortOfContainer function.
func Test_getNamedContainerPortOfContainer(t *testing.T) {
	assert := assert.New(t)
	// Create a simple pod
	pod := newPod()
	port, err := getNamedContainerPortOfContainer(*pod, "test-container-name", "test-port-name")
	assert.NoError(err, "Failed to find container port")
	assert.Equal(int32(42), port, "Expected to find valid named container port")
	_, err = getNamedContainerPortOfContainer(*pod, "wrong-container-name", "test-port-name")
	assert.Error(err, "Error should be returned when the specified container name does not exist")
	_, err = getNamedContainerPortOfContainer(*pod, "test-container-name", "wrong-port-name")
	assert.Error(err, "Error should be returned when the specified container port name does not exist")
}

// Test_getPodsWithReadyContainer tests the getPodsWithReadyContainer function.
func Test_getPodsWithReadyContainer(t *testing.T) {
	assert := assert.New(t)
	context, err := createFakeComponentContext()
	assert.NoError(err, "Failed to create fake component context.")
	pods, err := getPodsWithReadyContainer(context.Client(), "test-container-name", client.InNamespace("test-namespace-name"), client.MatchingLabels{"test-label-name": "test-label-value"})
	assert.NoError(err, "Failed to find pods with ready container")
	assert.Len(pods, 1, "Expected to find one pod with a ready container")
	// No pod returned when the container is not yet ready
	pods, err = getPodsWithReadyContainer(context.Client(), "test-not-ready-container-name", client.InNamespace("test-namespace-name"), client.MatchingLabels{"test-label-name": "test-label-value"})
	assert.NoError(err, "Failed to find pods")
	assert.Len(pods, 0, "Expected none of the pods to be returned when the container is not ready")
}

// Test_waitForPodsWithReadyContainer tests the waitForPodsWithReadyContainer function.
func Test_waitForPodsWithReadyContainer(t *testing.T) {
	assert := assert.New(t)
	context, err := createFakeComponentContext()
	assert.NoError(err, "Failed to create fake component context.")
	pods, err := waitForPodsWithReadyContainer(context.Client(), 1*time.Second, 5*time.Second, "test-container-name", client.InNamespace("test-namespace-name"), client.MatchingLabels{"test-label-name": "test-label-value"})
	assert.NoError(err, "Failed to find fake pod with ready container")
	assert.Len(pods, 1, "Expected to find one pod with a ready container")
}

// newFakeRuntimeScheme creates a new fake scheme
func newFakeRuntimeScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	appsv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	return scheme
}

// createFakeComponentContext creates a fake component context
func createFakeComponentContext() (spi.ComponentContext, error) {
	trueValue := true
	client2 := fake.NewFakeClientWithScheme(newFakeRuntimeScheme())
	createPod(client2)
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Version: "1.1.0",
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					MonitoringComponent: vzapi.MonitoringComponent{Enabled: &trueValue},
				},
			},
		},
		Status: vzapi.VerrazzanoStatus{
			Version: "1.0.0",
		},
	}

	return spi.NewFakeContext(client2, vz, false), nil
}

// createPod creates a k8s pod
func createPod(cli client.Client) {
	_ = cli.Create(context.TODO(), newPod())
}

func newPod() *corev1.Pod {
	labels := map[string]string{
		"test-label-name": "test-label-value",
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple-pod",
			Namespace: "test-namespace-name",
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "test-container-name",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 42,
							Name:          "test-port-name",
						},
					},
				},
				{
					Name: "test-not-ready-container-name",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 42,
							Name:          "test-port-name",
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "test-container-name",
					Ready: true,
				},
				{
					Name:  "test-not-ready-container-name",
					Ready: false,
				},
			},
		},
	}
}

func createElasticsearchPod(cli client.Client, portName string) {
	labels := map[string]string{
		"app": "system-es-master",
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "es-pod",
			Namespace: "verrazzano-system",
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "es-master",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 42,
							Name:          portName,
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "es-master",
					Ready: true,
				},
			},
		},
	}
	_ = cli.Create(context.TODO(), pod)
}

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	firstArg := os.Args[0]
	cmd := exec.Command(firstArg, cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}
