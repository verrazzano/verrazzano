// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"text/template"

	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

const (
	vmoDeployment = "verrazzano-monitoring-operator"
)

var (
	testScheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vmov1.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
	_ = vzclusters.AddToScheme(testScheme)

	_ = istioclinet.AddToScheme(testScheme)
	_ = istioclisec.AddToScheme(testScheme)
	_ = certv1.AddToScheme(testScheme)
	// +kubebuilder:scaffold:testScheme
}

// Test_fixupElasticSearchReplicaCount tests the fixupElasticSearchReplicaCount function.
func Test_fixupElasticSearchReplicaCount(t *testing.T) {
	a := assert.New(t)

	// GIVEN an Elasticsearch pod with a http port
	//  WHEN fixupElasticSearchReplicaCount is called
	//  THEN a command should be executed to get the cluster health information
	//   AND a command should be executed to update the cluster index settings
	//   AND no error should be returned
	ctx, err := createFakeComponentContext()
	a.NoError(err, "Failed to create fake component context.")
	createElasticsearchPod(ctx.Client(), "http")
	execCommand = fakeExecCommand
	fakeExecScenarioNames = []string{"fixupElasticSearchReplicaCount/get", "fixupElasticSearchReplicaCount/put"} // nolint
	fakeExecScenarioIndex = 0                                                                                    // nolint
	err = fixupElasticSearchReplicaCount(ctx, constants.VerrazzanoSystemNamespace)
	a.NoError(err, "Failed to fixup Elasticsearch index template")

	// GIVEN an Elasticsearch pod with no http port
	//  WHEN fixupElasticSearchReplicaCount is called
	//  THEN an error should be returned
	//   AND no commands should be invoked
	fakeExecScenarioNames = []string{} // nolint
	fakeExecScenarioIndex = 0          // nolint
	ctx, err = createFakeComponentContext()
	a.NoError(err, "Failed to create fake component context.")
	createElasticsearchPod(ctx.Client(), "tcp")
	err = fixupElasticSearchReplicaCount(ctx, constants.VerrazzanoSystemNamespace)
	a.Error(err, "Error should be returned if there is no http port for elasticsearch pods")

	// GIVEN an OpenSearch resource with version 1.1.0 in the status
	//  WHEN fixupElasticSearchReplicaCount is called
	//  THEN no error should be returned
	//   AND no commands should be invoked
	fakeExecScenarioNames = []string{} // nolint
	fakeExecScenarioIndex = 0          // nolint
	ctx, err = createFakeComponentContext()
	a.NoError(err, "Unexpected error")
	ctx.ActualCR().Status.Version = "1.1.0"
	err = fixupElasticSearchReplicaCount(ctx, constants.VerrazzanoSystemNamespace)
	a.NoError(err, "No error should be returned if the source version is 1.1.0 or later")

	// GIVEN an OpenSearch resource with Elasticsearch disabled
	//  WHEN fixupElasticSearchReplicaCount is called
	//  THEN no error should be returned
	//   AND no commands should be invoked
	fakeExecScenarioNames = []string{}
	fakeExecScenarioIndex = 0
	falseValue := false
	ctx, err = createFakeComponentContext()
	a.NoError(err, "Unexpected error")
	ctx.EffectiveCR().Spec.Components.Elasticsearch.Enabled = &falseValue
	err = fixupElasticSearchReplicaCount(ctx, constants.VerrazzanoSystemNamespace)
	a.NoError(err, "No error should be returned if the elasticsearch is not enabled")
}

// Test_getNamedContainerPortOfContainer tests the getNamedContainerPortOfContainer function.
func Test_getNamedContainerPortOfContainer(t *testing.T) {
	a := assert.New(t)
	// Create a simple pod
	pod := newPod()

	// GIVEN a pod with a ready container named test-ready-container-name
	//  WHEN getNamedContainerPortOfContainer is invoked for test-ready-container-name
	//  THEN return the port number for the container port named test-ready-port-name
	port, err := getNamedContainerPortOfContainer(*pod, "test-ready-container-name", "test-ready-port-name")
	a.NoError(err, "Failed to find container port")
	a.Equal(int32(42), port, "Expected to find valid named container port")

	// GIVEN a pod with a ready and unready container
	//  WHEN getNamedContainerPortOfContainer is invoked for a invalid container name
	//  THEN an error should be returned
	_, err = getNamedContainerPortOfContainer(*pod, "wrong-container-name", "test-port-name")
	a.Error(err, "Error should be returned when the specified container name does not exist")

	// GIVEN a pod with a ready container named test-ready-container-name
	//  WHEN getNamedContainerPortOfContainer is invoked for the ready container but wrong port name
	//  THEN an error should be returned
	_, err = getNamedContainerPortOfContainer(*pod, "test-ready-container-name", "wrong-port-name")
	a.Error(err, "Error should be returned when the specified container port name does not exist")
}

// Test_getPodsWithReadyContainer tests the getPodsWithReadyContainer function.
func Test_getPodsWithReadyContainer(t *testing.T) {
	a := assert.New(t)
	ctx, err := createFakeComponentContext()
	a.NoError(err, "Failed to create fake component context.")

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
	a.NoError(createResourceFromTemplate(ctx.Client(), &corev1.Pod{}, podTemplate, readyPodParams), "Failed to create test pod.")
	pods, err := getPodsWithReadyContainer(ctx.Client(), "test_container_name", client.InNamespace("test_namespace_name"), client.MatchingLabels{"test_label_name": "test_ready_label_value"})
	a.NoError(err, "Unexpected error")
	a.Len(pods, 1, "Expected to find one pod with a ready container")

	// GIVEN a pod with an unready container
	//  WHEN getPodsWithReadyContainer is invoked
	//  THEN expect not pods to be returned
	//   AND expect no error
	unreadyPodParams := map[string]string{
		"test_pod_name":        "test_unready_pod_name",
		"test_label_value":     "test_unready_label_value",
		"test_container_ready": "false",
	}
	a.NoError(createResourceFromTemplate(ctx.Client(), &corev1.Pod{}, podTemplate, unreadyPodParams), "Failed to create test pod.")
	pods, err = getPodsWithReadyContainer(ctx.Client(), "test_container_name", client.InNamespace("test_namespace_name"), client.MatchingLabels{"test-label-name": "test_unready_label_value"})
	a.NoError(err, "Unexpected error")
	a.Len(pods, 0, "Expected not to find and pods with a ready container")
}

// Test_waitForPodsWithReadyContainer tests the waitForPodsWithReadyContainer function.
func Test_waitForPodsWithReadyContainer(t *testing.T) {
	a := assert.New(t)

	// GIVEN a pod with a ready container
	//  WHEN waitForPodsWithReadyContainer is invoked for the container
	//  THEN expect the ready pod to be returned
	ctx, err := createFakeComponentContext()
	createPod(ctx.Client())
	a.NoError(err, "Failed to create fake component context.")
	pods, err := waitForPodsWithReadyContainer(ctx.Client(), "test-ready-container-name", client.InNamespace("test-namespace-name"), client.MatchingLabels{"test-label-name": "test-label-value"})
	a.NoError(err, "Unexpected error finding pods with ready container")
	a.Len(pods, 1, "Expected to find one pod with a ready container")

	// GIVEN a pod with a ready container
	//  WHEN waitForPodsWithReadyContainer is invoked for a container that will never be ready
	//  THEN expect no pods to eventually be returned
	ctx, err = createFakeComponentContext()
	a.NoError(err, "Failed to create fake component context.")
	pods, err = waitForPodsWithReadyContainer(ctx.Client(), "test-unready-container-name", client.InNamespace("test-namespace-name"), client.MatchingLabels{"test-label-name": "test-label-value"})
	a.NoError(err, "Unexpected error finding pods with ready container")
	a.Len(pods, 0, "Expected to find no pods with a ready container")
}

// newFakeRuntimeScheme creates a new fake scheme
func newFakeRuntimeScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	return scheme
}

// createFakeComponentContext creates a fake component context
func createFakeComponentContext() (spi.ComponentContext, error) {
	c := fake.NewClientBuilder().WithScheme(newFakeRuntimeScheme()).Build()

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

	return spi.NewFakeContext(c, &vzObject, nil, false), nil
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
			Namespace: constants.VerrazzanoSystemNamespace,
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
	a := assert.New(t)
	scenario, found := os.LookupEnv("TEST_FAKE_EXEC_SCENARIO")
	if found {
		switch scenario {
		case "fixupElasticSearchReplicaCount/get":
			a.Equal(`curl -v -XGET -s -k --fail http://localhost:42/_cluster/health`,
				os.Args[13], "Expected curl command to be correct.")
			fmt.Print(`"number_of_data_nodes":1,`)
		case "fixupElasticSearchReplicaCount/put":
			fmt.Println(scenario)
			fmt.Println(strings.Join(os.Args, " "))
			a.Equal(`curl -v -XPUT -d '{"index":{"auto_expand_replicas":"0-1"}}' --header 'Content-Type: application/json' -s -k --fail http://localhost:42/verrazzano-*/_settings`,
				os.Args[13], "Expected curl command to be correct.")
		default:
			a.Fail("Unknown test scenario provided in environment variable TEST_FAKE_EXEC_SCENARIO: %s", scenario)
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
	ybytes, err := yaml.YAMLToJSON([]byte(str))
	if err != nil {
		return err
	}
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(ybytes, nil, uns)
	if err != nil {
		return err
	}
	return nil
}

// createResourceFromTemplate builds a resource by merging the data with the template and then
// stores the resource using the provided client.
func createResourceFromTemplate(cli client.Client, obj client.Object, template string, data interface{}) error {
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

// TestIsReadySecretNotReady tests the OpenSearch isOpenSearchReady call
// GIVEN an OpenSearch component
//
//	WHEN I call isOpenSearchReady when it is installed and the deployment availability criteria are met, but the secret is not found
//	THEN false is returned
func TestIsReadySecretNotReady(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	falseValue := false
	vz.Spec.Components = vzapi.ComponentSpec{
		Console:       &vzapi.ConsoleComponent{Enabled: &falseValue},
		Fluentd:       &vzapi.FluentdComponent{Enabled: &falseValue},
		Kibana:        &vzapi.KibanaComponent{Enabled: &falseValue},
		Elasticsearch: &vzapi.ElasticsearchComponent{Enabled: &falseValue},
		Prometheus:    &vzapi.PrometheusComponent{Enabled: &falseValue},
		Grafana:       &vzapi.GrafanaComponent{Enabled: &falseValue},
	}
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      vmoDeployment,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	}).Build()
	ctx := spi.NewFakeContext(c, vz, nil, false)
	assert.False(t, isOSReady(ctx))
}

// TestIsReadyNotInstalled tests the OpenSearch isOpenSearchReady call
// GIVEN an OpenSearch component
//
//	WHEN I call isOpenSearchReady when it is not installed
//	THEN false is returned
func TestIsReadyNotInstalled(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	assert.False(t, isOSReady(ctx))
}

// TestIsReady tests the isOpenSearchReady call
// GIVEN OpenSearch components that are all enabled by default
//
//	WHEN I call isOpenSearchReady when all requirements are met
//	THEN true is returned
func TestIsReady(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      fmt.Sprintf("%s-0", esDataDeployment),
				Labels: map[string]string{
					"app":   "system-es-data",
					"index": "0",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app":   "system-es-data",
						"index": "0",
					},
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
				Namespace: ComponentNamespace,
				Name:      fmt.Sprintf("%s-0-95d8c5d96-m6mbr", esDataDeployment),
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"app":               "system-es-data",
					"index":             "0",
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        fmt.Sprintf("%s-0-95d8c5d96", esDataDeployment),
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      fmt.Sprintf("%s-1", esDataDeployment),
				Labels: map[string]string{
					"app":   "system-es-data",
					"index": "1",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app":   "system-es-data",
						"index": "1",
					},
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
				Namespace: ComponentNamespace,
				Name:      fmt.Sprintf("%s-1-95d8c5d96-m6mbr", esDataDeployment),
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"app":               "system-es-data",
					"index":             "1",
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        fmt.Sprintf("%s-1-95d8c5d96", esDataDeployment),
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esIngestDeployment,
				Labels:    map[string]string{"app": "system-es-ingest"},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "system-es-ingest"},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 2,
				Replicas:          2,
				UpdatedReplicas:   2,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esIngestDeployment + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"app":               "system-es-ingest",
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esIngestDeployment + "-95d8c5d96-x1v76",
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"app":               "system-es-ingest",
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        esIngestDeployment + "-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esMasterStatefulset,
				Labels:    map[string]string{"app": "system-es-master"},
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "system-es-master"},
				},
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas:   1,
				UpdatedReplicas: 1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esMasterStatefulset + "-0",
				Labels: map[string]string{
					"app":                      "system-es-master",
					"controller-revision-hash": "test-95d8c5d96",
				},
			},
		},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano",
			Namespace: ComponentNamespace}},
		&appsv1.ControllerRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-95d8c5d96",
				Namespace: ComponentNamespace,
			},
			Revision: 1,
		}).Build()

	vz := &vzapi.Verrazzano{}
	vz.Spec.Components = vzapi.ComponentSpec{
		Elasticsearch: &vzapi.ElasticsearchComponent{
			ESInstallArgs: []vzapi.InstallArgs{
				{
					Name:  "nodes.master.replicas",
					Value: "1",
				},
				{
					Name:  "nodes.data.replicas",
					Value: "2",
				},
				{
					Name:  "nodes.ingest.replicas",
					Value: "2",
				},
			},
		},
	}
	ctx := spi.NewFakeContext(c, vz, nil, false)
	assert.True(t, isOSReady(ctx))
}

// TestIsReadyDeploymentNotAvailable tests the OpenSearch isOpenSearchReady call
// GIVEN an OpenSearch component
//
//	WHEN I call isOpenSearchReady when the Kibana deployment is not available
//	THEN false is returned
func TestIsReadyDeploymentNotAvailable(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.Deployment{},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      fmt.Sprintf("%s-0", esDataDeployment),
				Labels:    map[string]string{"app": "system-es-data", "index": "0"},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   0,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      fmt.Sprintf("%s-1", esDataDeployment),
				Labels:    map[string]string{"app": "system-es-data", "index": "1"},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esIngestDeployment,
				Labels:    map[string]string{"app": "system-es-ingest"},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esMasterStatefulset,
				Labels:    map[string]string{"app": "system-es-master"},
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas:   1,
				UpdatedReplicas: 1,
			},
		},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano",
			Namespace: ComponentNamespace}},
	).Build()

	vz := &vzapi.Verrazzano{}
	vz.Spec.Components = vzapi.ComponentSpec{
		Elasticsearch: &vzapi.ElasticsearchComponent{
			Nodes: []vzapi.OpenSearchNode{
				{
					Name:     "es-master",
					Replicas: 2,
				},
				{
					Name:     "es-data",
					Replicas: 2,
				},
				{
					Name:     "es-ingest",
					Replicas: 2,
				},
			},
		},
	}
	ctx := spi.NewFakeContext(c, vz, nil, false)
	assert.False(t, isOSReady(ctx))
}

// TestIsReadyDeploymentVMIDisabled tests the OpenSearch isOpenSearchReady call
// GIVEN an OpenSearch component with all VMI components disabled
//
//	WHEN I call isOpenSearchReady
//	THEN true is returned
func TestIsReadyDeploymentVMIDisabled(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano",
		Namespace: ComponentNamespace}},
	).Build()
	vz := &vzapi.Verrazzano{}
	falseValue := false
	vz.Spec.Components = vzapi.ComponentSpec{
		Console:       &vzapi.ConsoleComponent{Enabled: &falseValue},
		Fluentd:       &vzapi.FluentdComponent{Enabled: &falseValue},
		Kibana:        &vzapi.KibanaComponent{Enabled: &falseValue},
		Elasticsearch: &vzapi.ElasticsearchComponent{Enabled: &falseValue},
		Prometheus:    &vzapi.PrometheusComponent{Enabled: &falseValue},
		Grafana:       &vzapi.GrafanaComponent{Enabled: &falseValue},
	}
	ctx := spi.NewFakeContext(c, vz, nil, false)
	assert.True(t, isOSReady(ctx))
}

// TestIsinstalled tests the OpenSearch doesOSExist call
// GIVEN a verrazzano
//
//	WHEN I call doesOSExist
//	THEN true is returned
func TestIsinstalled(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esMasterStatefulset,
				Labels:    map[string]string{"app": "system-es-master"},
			},
		},
	).Build()

	vz := &vzapi.Verrazzano{}
	ctx := spi.NewFakeContext(c, vz, nil, false)
	assert.True(t, doesOSExist(ctx))
}

func TestIsOSNodeReady(t *testing.T) {
	readyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo-95d8c5d96-m6mbr",
			Namespace: ComponentNamespace,
			Labels: map[string]string{
				"controller-revision-hash": "foo-95d8c5d96",
				"pod-template-hash":        "95d8c5d96",
				"app":                      "foo",
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Ready: true,
				},
			},
		},
	}
	controllerRevision := &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo-95d8c5d96",
			Namespace: ComponentNamespace,
		},
		Revision: 1,
	}
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": "foo",
		},
	}
	singleMasterClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      esMasterStatefulset,
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: selector,
			},
			Status: appsv1.StatefulSetStatus{
				UpdatedReplicas: 1,
				ReadyReplicas:   1,
			},
		},
		readyPod,
		controllerRevision,
	).Build()
	masterNode := vzapi.OpenSearchNode{
		Name:     "es-master",
		Replicas: 1,
		Roles: []vmov1.NodeRole{
			vmov1.MasterRole,
		},
	}

	dataNode := vzapi.OpenSearchNode{
		Name:     "es-data",
		Replicas: 2,
		Roles: []vmov1.NodeRole{
			vmov1.DataRole,
		},
	}
	dataDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      fmt.Sprintf("%s-%d", esDataDeployment, 0),
			Labels:    map[string]string{"app": "foo"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: selector,
		},
		Status: appsv1.DeploymentStatus{
			UpdatedReplicas:   1,
			ReadyReplicas:     1,
			AvailableReplicas: 1,
		},
	}
	dataDeployment2 := dataDeployment.DeepCopy()
	dataDeployment2.Name = fmt.Sprintf("%s-%d", esDataDeployment, 1)
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   ComponentNamespace,
			Name:        "vmi-system-es-data-0-95d8c5d96",
			Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
		},
	}
	rs2 := rs.DeepCopy()
	rs2.Name = "vmi-system-es-data-1-95d8c5d96"
	dataNodeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		dataDeployment,
		dataDeployment2,
		readyPod,
		controllerRevision,
		rs,
		rs2,
	).Build()

	var tests = []struct {
		name  string
		ctx   spi.ComponentContext
		node  vzapi.OpenSearchNode
		ready bool
	}{
		{
			"ready when master node is ready",
			spi.NewFakeContext(singleMasterClient, nil, nil, false),
			masterNode,
			true,
		},
		{
			"ready when data node is ready",
			spi.NewFakeContext(dataNodeClient, nil, nil, false),
			dataNode,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.ready, isOSNodeReady(tt.ctx, tt.node, tt.name))
		})
	}
}
