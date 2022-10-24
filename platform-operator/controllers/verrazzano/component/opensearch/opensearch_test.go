// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
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
