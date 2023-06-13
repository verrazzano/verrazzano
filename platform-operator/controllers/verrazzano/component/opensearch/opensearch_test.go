// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"
	"testing"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzclusters "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	vmoDeployment = "verrazzano-monitoring-operator"
	masterAppName = "system-es-master"
	vzsys         = "verrazzano-system"
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
				Labels:    map[string]string{"app": masterAppName},
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": masterAppName},
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
					"app":                      masterAppName,
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
				Labels:    map[string]string{"app": masterAppName},
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
					Replicas: common.Int32Ptr(2),
				},
				{
					Name:     "es-data",
					Replicas: common.Int32Ptr(2),
				},
				{
					Name:     "es-ingest",
					Replicas: common.Int32Ptr(2),
				},
			},
		},
	}
	ctx := spi.NewFakeContext(c, vz, nil, false)
	assert.False(t, isOSReady(ctx))
}

// TestFindESReplicas tests the OpenSearch FindESReplicas call
// GIVEN an OpenSearch component, a context and a vmov1.NodeRole
//
//	WHEN I call FindESReplicas
//	THEN the number of replicas of that NodeRole is returned
func TestFindESReplicas(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	val := findESReplicas(ctx, "data")
	assert.Equal(t, val, int32(0))

	trueVal := true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					Enabled: &(trueVal),
					Nodes:   []vzapi.OpenSearchNode{{Name: "node1", Replicas: common.Int32Ptr(4), Roles: []vmov1.NodeRole{"data"}}, {Name: "node2", Replicas: common.Int32Ptr(7), Roles: []vmov1.NodeRole{"master"}}, {Name: "node3", Replicas: common.Int32Ptr(8), Roles: []vmov1.NodeRole{"ingest"}}},
				},
			},
		},
	}
	ctx = spi.NewFakeContext(c, vz, nil, false)
	val = findESReplicas(ctx, "data")
	assert.Equal(t, val, int32(4))

	val = findESReplicas(ctx, "master")
	assert.Equal(t, val, int32(7))

	val = findESReplicas(ctx, "ingest")
	assert.Equal(t, val, int32(8))

	// Check with nil replicas
	vz = &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					Enabled: &(trueVal),
					Nodes: []vzapi.OpenSearchNode{
						{
							Name: "node2", Roles: []vmov1.NodeRole{"master"},
						},
					},
				},
			},
		},
	}
	ctx = spi.NewFakeContext(c, vz, nil, false)
	val = findESReplicas(ctx, "master")
	assert.Equal(t, val, int32(0))
}

// TestNodesToObjectKeys tests the OpenSearch NodesToObjectKeys call
// GIVEN an OpenSearch component and a vzapi
//
//	WHEN I call NodesToObjectKeys
//	THEN objects are returned
func TestNodesToObjectKeys(t *testing.T) {
	trueVal := true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					Enabled: &(trueVal),
				},
			},
		},
	}
	expected := nodesToObjectKeys(vz)
	actual := &ready.AvailabilityObjects{}
	assert.Equal(t, expected, actual)

	vztwo := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					Enabled: &(trueVal),
					Nodes:   []vzapi.OpenSearchNode{{Name: "node1", Replicas: common.Int32Ptr(1), Roles: []vmov1.NodeRole{"data"}}, {Name: "node2", Replicas: common.Int32Ptr(1), Roles: []vmov1.NodeRole{"master"}}, {Name: "node3", Replicas: common.Int32Ptr(1), Roles: []vmov1.NodeRole{"ingest"}}},
				},
			},
		},
	}
	actual = &ready.AvailabilityObjects{StatefulsetNames: []types.NamespacedName{{Namespace: vzsys, Name: "vmi-system-node2"}}, DeploymentNames: []types.NamespacedName{{Namespace: vzsys, Name: "vmi-system-node1-0"}, {Namespace: vzsys, Name: "vmi-system-node3"}}, DeploymentSelectors: []client.ListOption(nil), DaemonsetNames: []types.NamespacedName(nil)}
	expected = nodesToObjectKeys(vztwo)
	assert.Equal(t, expected, actual)

	// nil replicas should have no availability objects
	vzthree := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					Enabled: &(trueVal),
					Nodes: []vzapi.OpenSearchNode{
						{Name: "node1", Roles: []vmov1.NodeRole{"data"}},
						{Name: "node2", Roles: []vmov1.NodeRole{"master"}},
						{Name: "node3", Roles: []vmov1.NodeRole{"ingest"}}},
				},
			},
		},
	}
	actual = &ready.AvailabilityObjects{StatefulsetNames: []types.NamespacedName(nil), DeploymentNames: []types.NamespacedName(nil), DeploymentSelectors: []client.ListOption(nil), DaemonsetNames: []types.NamespacedName(nil)}
	expected = nodesToObjectKeys(vzthree)
	assert.Equal(t, expected, actual)
}

// TestIsSingleDataNodeCluster tests the OpenSearch IsSingleDataNodeCluster call
// GIVEN an OpenSearch component and a context
//
//	WHEN I call IsSingleDataNodeCluster
//	THEN a bool value is returned
func TestIsSingleDataNodeCluster(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	val := IsSingleDataNodeCluster(ctx)
	assert.True(t, val)

	trueVal := true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					Enabled: &(trueVal),
					Nodes:   []vzapi.OpenSearchNode{{Name: "node1", Replicas: common.Int32Ptr(4), Roles: []vmov1.NodeRole{"data"}}, {Name: "node2", Replicas: common.Int32Ptr(7), Roles: []vmov1.NodeRole{"master"}}, {Name: "node3", Replicas: common.Int32Ptr(8), Roles: []vmov1.NodeRole{"ingest"}}},
				},
			},
		},
	}
	ctx = spi.NewFakeContext(c, vz, nil, false)
	val = IsSingleDataNodeCluster(ctx)
	assert.False(t, val)

	vz = &vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{Elasticsearch: &vzapi.ElasticsearchComponent{Nodes: []vzapi.OpenSearchNode{{Name: "node1", Roles: []vmov1.NodeRole{"data"}}}}}}}
	ctx = spi.NewFakeContext(c, vz, nil, false)
	val = IsSingleDataNodeCluster(ctx)
	assert.True(t, val)
}

// TestIsReadyDeploymentVMIDisabled tests the OpenSearch isOpenSearchReady call
// GIVEN an OpenSearch component with all VMI components disabled
//
//	WHEN I call isOpenSearchReady
//	THEN true is returned
func TestIsReadyDeploymentVMIDisabled(t *testing.T) {
	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(func(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
		return helm.CreateActionConfig(true, ComponentName, release.StatusDeployed, vzlog.DefaultLogger(), func(name string, releaseStatus release.Status) *release.Release {
			now := time.Now()
			return &release.Release{
				Name:      ComponentName,
				Namespace: ComponentNamespace,
				Info: &release.Info{
					FirstDeployed: now,
					LastDeployed:  now,
					Status:        releaseStatus,
					Description:   "Named Release Stub",
				},
				Version: 1,
			}
		})
	})
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
				Labels:    map[string]string{"app": masterAppName},
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
		Replicas: common.Int32Ptr(1),
		Roles: []vmov1.NodeRole{
			vmov1.MasterRole,
		},
	}

	dataNode := vzapi.OpenSearchNode{
		Name:     "es-data",
		Replicas: common.Int32Ptr(2),
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

	nodeWithNilReplicas := vzapi.OpenSearchNode{Name: "es-master", Roles: []vmov1.NodeRole{"master"}}
	nilReplicasClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
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
		{
			"nil replicas is ready",
			spi.NewFakeContext(nilReplicasClient, nil, nil, false),
			nodeWithNilReplicas,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.ready, isOSNodeReady(tt.ctx, tt.node, tt.name))
		})
	}
}
