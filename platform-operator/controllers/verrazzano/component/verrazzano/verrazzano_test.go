// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"strings"
	"testing"
	"text/template"
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

	// GIVEN an Elasticsearch pod with a http port
	//  WHEN fixupElasticSearchReplicaCount is called
	//  THEN a command should be executed to get the cluster health information
	//   AND a command should be executed to update the cluster index settings
	//   AND no error should be returned
	context, err := createFakeComponentContext()
	assert.NoError(err, "Failed to create fake component context.")
	createElasticsearchPod(context.Client(), "http")
	execCommand = fakeExecCommand
	fakeExecScenarioNames = []string{"fixupElasticSearchReplicaCount/get", "fixupElasticSearchReplicaCount/put"} //nolint,ineffassign
	fakeExecScenarioIndex = 0                                                                                    //nolint,ineffassign
	err = fixupElasticSearchReplicaCount(context, "verrazzano-system")
	assert.NoError(err, "Failed to fixup Elasticsearch index template")

	// GIVEN an Elasticsearch pod with no http port
	//  WHEN fixupElasticSearchReplicaCount is called
	//  THEN an error should be returned
	//   AND no commands should be invoked
	fakeExecScenarioNames = []string{} //nolint,ineffassign
	fakeExecScenarioIndex = 0          //nolint,ineffassign
	context, err = createFakeComponentContext()
	assert.NoError(err, "Failed to create fake component context.")
	createElasticsearchPod(context.Client(), "tcp")
	err = fixupElasticSearchReplicaCount(context, "verrazzano-system")
	assert.Error(err, "Error should be returned if there is no http port for elasticsearch pods")

	// GIVEN a Verrazzano resource with version 1.1.0 in the status
	//  WHEN fixupElasticSearchReplicaCount is called
	//  THEN no error should be returned
	//   AND no commands should be invoked
	fakeExecScenarioNames = []string{} //nolint,ineffassign
	fakeExecScenarioIndex = 0          //nolint,ineffassign
	context, err = createFakeComponentContext()
	assert.NoError(err, "Unexpected error")
	context.ActualCR().Status.Version = "1.1.0"
	err = fixupElasticSearchReplicaCount(context, "verrazzano-system")
	assert.NoError(err, "No error should be returned if the source version is 1.1.0 or later")

	// GIVEN a Verrazzano resource with Elasticsearch disabled
	//  WHEN fixupElasticSearchReplicaCount is called
	//  THEN no error should be returned
	//   AND no commands should be invoked
	fakeExecScenarioNames = []string{}
	fakeExecScenarioIndex = 0
	falseValue := false
	context, err = createFakeComponentContext()
	assert.NoError(err, "Unexpected error")
	context.EffectiveCR().Spec.Components.Elasticsearch.Enabled = &falseValue
	err = fixupElasticSearchReplicaCount(context, "verrazzano-system")
	assert.NoError(err, "No error should be returned if the elasticsearch is not enabled")
}

// Test_getNamedContainerPortOfContainer tests the getNamedContainerPortOfContainer function.
func Test_getNamedContainerPortOfContainer(t *testing.T) {
	assert := assert.New(t)
	// Create a simple pod
	pod := newPod()

	// GIVEN a pod with a ready container named test-ready-container-name
	//  WHEN getNamedContainerPortOfContainer is invoked for test-ready-container-name
	//  THEN return the port number for the container port named test-ready-port-name
	port, err := getNamedContainerPortOfContainer(*pod, "test-ready-container-name", "test-ready-port-name")
	assert.NoError(err, "Failed to find container port")
	assert.Equal(int32(42), port, "Expected to find valid named container port")

	// GIVEN a pod with a ready and unready container
	//  WHEN getNamedContainerPortOfContainer is invoked for a invalid container name
	//  THEN an error should be returned
	_, err = getNamedContainerPortOfContainer(*pod, "wrong-container-name", "test-port-name")
	assert.Error(err, "Error should be returned when the specified container name does not exist")

	// GIVEN a pod with a ready container named test-ready-container-name
	//  WHEN getNamedContainerPortOfContainer is invoked for the ready container but wrong port name
	//  THEN an error should be returned
	_, err = getNamedContainerPortOfContainer(*pod, "test-ready-container-name", "wrong-port-name")
	assert.Error(err, "Error should be returned when the specified container port name does not exist")
}

// Test_getPodsWithReadyContainer tests the getPodsWithReadyContainer function.
func Test_getPodsWithReadyContainer(t *testing.T) {
	assert := assert.New(t)
	ctx, err := createFakeComponentContext()
	assert.NoError(err, "Failed to create fake component context.")

	podTemplate := `---
apiVersion: v1
kind: Pod
metadata:
 labels:
   test_label_name: {{.test_label_value}}
 name: {{.test_pod_name}}
 namespace: test_namespace_name
spec:
 containers:
   - name: test_container_name
     ports:
       - name: http
         containerPort: 9200
         protocol: TCP
status:
 containerStatuses:
   - name: test_container_name
     ready: {{.test_container_ready}}`

	// GIVEN a pod with a ready container
	//  WHEN getPodsWithReadyContainer is invoked
	//  THEN expect the pod to be returned
	//   AND expect no error
	readyPodParams := map[string]string{
		"test_pod_name":        "test_ready_pod_name",
		"test_label_value":     "test_ready_label_value",
		"test_container_ready": "true",
	}
	assert.NoError(createResourceFromTemplate(ctx.Client(), &corev1.Pod{}, podTemplate, readyPodParams), "Failed to create test pod.")
	pods, err := getPodsWithReadyContainer(ctx.Client(), "test_container_name", client.InNamespace("test_namespace_name"), client.MatchingLabels{"test_label_name": "test_ready_label_value"})
	assert.NoError(err, "Unexpected error")
	assert.Len(pods, 1, "Expected to find one pod with a ready container")

	// GIVEN a pod with an unready container
	//  WHEN getPodsWithReadyContainer is invoked
	//  THEN expect not pods to be returned
	//   AND expect no error
	unreadyPodParams := map[string]string{
		"test_pod_name":        "test_unready_pod_name",
		"test_label_value":     "test_unready_label_value",
		"test_container_ready": "false",
	}
	assert.NoError(createResourceFromTemplate(ctx.Client(), &corev1.Pod{}, podTemplate, unreadyPodParams), "Failed to create test pod.")
	pods, err = getPodsWithReadyContainer(ctx.Client(), "test_container_name", client.InNamespace("test_namespace_name"), client.MatchingLabels{"test-label-name": "test_unready_label_value"})
	assert.NoError(err, "Unexpected error")
	assert.Len(pods, 0, "Expected not to find and pods with a ready container")
}

// Test_waitForPodsWithReadyContainer tests the waitForPodsWithReadyContainer function.
func Test_waitForPodsWithReadyContainer(t *testing.T) {
	assert := assert.New(t)

	// GIVEN a pod with a ready container
	//  WHEN waitForPodsWithReadyContainer is invoked for the container
	//  THEN expect the ready pod to be returned
	context, err := createFakeComponentContext()
	createPod(context.Client())
	assert.NoError(err, "Failed to create fake component context.")
	pods, err := waitForPodsWithReadyContainer(context.Client(), 1*time.Nanosecond, 5*time.Nanosecond, "test-ready-container-name", client.InNamespace("test-namespace-name"), client.MatchingLabels{"test-label-name": "test-label-value"})
	assert.NoError(err, "Unexpected error finding pods with ready container")
	assert.Len(pods, 1, "Expected to find one pod with a ready container")

	// GIVEN a pod with a ready container
	//  WHEN waitForPodsWithReadyContainer is invoked for a container that will never be ready
	//  THEN expect no pods to eventually be returned
	context, err = createFakeComponentContext()
	assert.NoError(err, "Failed to create fake component context.")
	pods, err = waitForPodsWithReadyContainer(context.Client(), 1*time.Nanosecond, 2*time.Nanosecond, "test-unready-container-name", client.InNamespace("test-namespace-name"), client.MatchingLabels{"test-label-name": "test-label-value"})
	assert.NoError(err, "Unexpected error finding pods with ready container")
	assert.Len(pods, 0, "Expected to find no pods with a ready container")
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
	client := fake.NewFakeClientWithScheme(newFakeRuntimeScheme())

	vzTemplate := `---
apiVersion: install.verrazzano.io/v1alpha1
kind: Verrazzano
metadata:
  name: test-verrazzano
  namespace: default
spec:
  version: 1.1.0
  profile: dev
  components:
    elasticsearch:
      enabled: true
status:
  version: 1.0.0
`
	vzObject := vzapi.Verrazzano{}
	if err := createObjectFromTemplate(&vzObject, vzTemplate, nil); err != nil {
		return nil, err
	}

	return spi.NewFakeContext(client, &vzObject, false), nil
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
					Name: "test-ready-container-name",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 42,
							Name:          "test-ready-port-name",
						},
					},
				},
				{
					Name: "test-not-ready-container-name",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 777,
							Name:          "test-not-ready-port-name",
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "test-ready-container-name",
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

var fakeExecScenarioNames = []string{}
var fakeExecScenarioIndex = 0

// fakeExecCommand is used to fake command execution.
// The TestFakeExecHandler test is executed as a test.
// The test scenario is communicated using the TEST_FAKE_EXEC_SCENARIO environment variable.
// The value of that variable is derrived from fakeExecScenarioNames at fakeExecScenarioIndex
// The fakeExecScenarioIndex is incremented after every invocation.
func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestFakeExecHandler", "--", command}
	cs = append(cs, args...)
	firstArg := os.Args[0]
	cmd := exec.Command(firstArg, cs...)
	cmd.Env = []string{
		fmt.Sprintf("TEST_FAKE_EXEC_SCENARIO=%s", fakeExecScenarioNames[fakeExecScenarioIndex]),
	}
	fakeExecScenarioIndex++
	return cmd
}

// TestFakeExecHandler is a test intended to be use to handle fake command execution
// See the fakeExecCommand function.
// When this test is invoked normally no TEST_FAKE_EXEC_SCENARIO is present
// so no assertions are made and therefore passes.
func TestFakeExecHandler(t *testing.T) {
	assert := assert.New(t)
	scenario, found := os.LookupEnv("TEST_FAKE_EXEC_SCENARIO")
	if found {
		switch scenario {
		case "fixupElasticSearchReplicaCount/get":
			assert.Equal(`curl -v -XGET -s -k --fail http://localhost:42/_cluster/health`,
				os.Args[13], "Expected curl command to be correct.")
			fmt.Print(`"number_of_data_nodes":1,`)
		case "fixupElasticSearchReplicaCount/put":
			fmt.Println(scenario)
			fmt.Println(strings.Join(os.Args, " "))
			assert.Equal(`curl -v -XPUT -d '{"index":{"auto_expand_replicas":"0-1"}}' --header 'Content-Type: application/json' -s -k --fail http://localhost:42/verrazzano-*/_settings`,
				os.Args[13], "Expected curl command to be correct.")
		default:
			assert.Fail("Unknown test scenario provided in environment variable TEST_FAKE_EXEC_SCENARIO: %s", scenario)
		}
	}
}

// populateTemplate reads a template from a file and replaces values in the template from param maps
// template - The template text
// params - a vararg of param maps
func populateTemplate(templateStr string, data interface{}) (string, error) {
	hasher := sha256.New()
	hasher.Write([]byte(templateStr))
	name := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	t, err := template.New(name).Option("missingkey=error").Parse(templateStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = t.ExecuteTemplate(&buf, name, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// updateUnstructuredFromYAMLTemplate updates an unstructured from a populated YAML template file.
// uns - The unstructured to update
// template - The template text
// params - The param maps to merge into the template
func updateUnstructuredFromYAMLTemplate(uns *unstructured.Unstructured, template string, data interface{}) error {
	str, err := populateTemplate(template, data)
	if err != nil {
		return err
	}
	bytes, err := yaml.YAMLToJSON([]byte(str))
	if err != nil {
		return err
	}
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(bytes, nil, uns)
	if err != nil {
		return err
	}
	return nil
}

// createResourceFromTemplate builds a resource by merging the data with the template and then
// stores the resource using the provided client.
func createResourceFromTemplate(cli client.Client, obj runtime.Object, template string, data interface{}) error {
	if err := createObjectFromTemplate(obj, template, data); err != nil {
		return err
	}
	if err := cli.Create(context.TODO(), obj); err != nil {
		return err
	}
	return nil
}

// createObjectFromTemplate builds an object by merging the data with the template
func createObjectFromTemplate(obj runtime.Object, template string, data interface{}) error {
	uns := unstructured.Unstructured{}
	if err := updateUnstructuredFromYAMLTemplate(&uns, template, data); err != nil {
		return err
	}
	return runtime.DefaultUnstructuredConverter.FromUnstructured(uns.Object, obj)
}
