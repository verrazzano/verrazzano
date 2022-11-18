// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restart

import (
	"strings"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vpoFakeClient "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned/fake"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/pkg/k8sclient"
	opensearchpsr "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/opensearch"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sapiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	crtFakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type fakeEnv struct {
	data map[string]string
}

type fakePsrClient struct {
	psrClient *k8sclient.PsrClient
}

// TestGetters tests the worker getters
// GIVEN a worker
//
//	WHEN the getter methods are calls
//	THEN ensure that the correct results are returned
func TestGetters(t *testing.T) {
	origFunc := overridePsrClient()
	defer func() {
		funcNewPsrClient = origFunc
	}()

	envMap := map[string]string{
		openSearchTier: opensearchpsr.MasterTier,
	}
	f := fakeEnv{data: envMap}
	saveEnv := osenv.GetEnvFunc
	osenv.GetEnvFunc = f.GetEnv
	defer func() {
		osenv.GetEnvFunc = saveEnv
	}()

	w, err := NewRestartWorker()
	assert.NoError(t, err)

	wd := w.GetWorkerDesc()
	assert.Equal(t, config.WorkerTypeOpsRestart, wd.WorkerType)
	assert.Equal(t, "Worker to restart pods in the specified OpenSearch tier", wd.Description)
	assert.Equal(t, metricsPrefix, wd.MetricsPrefix)

	logged := w.WantLoopInfoLogged()
	assert.False(t, logged)
}

// TestGetEnvDescList tests the GetEnvDescList method
// GIVEN a worker
//
//	WHEN the GetEnvDescList methods is called
//	THEN ensure that the correct results are returned
func TestGetEnvDescList(t *testing.T) {
	origFunc := overridePsrClient()
	defer func() {
		funcNewPsrClient = origFunc
	}()

	envMap := map[string]string{
		openSearchTier: opensearchpsr.MasterTier,
	}
	f := fakeEnv{data: envMap}
	saveEnv := osenv.GetEnvFunc
	osenv.GetEnvFunc = f.GetEnv
	defer func() {
		osenv.GetEnvFunc = saveEnv
	}()

	tests := []struct {
		name     string
		key      string
		defval   string
		required bool
	}{
		{name: "1",
			key:      openSearchTier,
			defval:   "",
			required: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, err := NewRestartWorker()
			assert.NoError(t, err)
			el := w.GetEnvDescList()
			for _, e := range el {
				if e.Key == test.key {
					assert.Equal(t, test.defval, e.DefaultVal)
					assert.Equal(t, test.required, e.Required)
				}
			}
		})
	}
}

// TestGetMetricDescList tests the GetEnvDescList method
// GIVEN a worker
//
//	WHEN the GetEnvDescList methods is called
//	THEN ensure that the correct results are returned
func TestGetMetricDescList(t *testing.T) {
	origFunc := overridePsrClient()
	defer func() {
		funcNewPsrClient = origFunc
	}()

	envMap := map[string]string{
		openSearchTier: opensearchpsr.MasterTier,
	}
	f := fakeEnv{data: envMap}
	saveEnv := osenv.GetEnvFunc
	osenv.GetEnvFunc = f.GetEnv
	defer func() {
		osenv.GetEnvFunc = saveEnv
	}()

	tests := []struct {
		name   string
		fqName string
		help   string
	}{
		{name: "1", fqName: metricsPrefix + "_pod_restart_count", help: "The total number of OpenSearch pod restarts"},
		{name: "2", fqName: metricsPrefix + "_pod_restart_time_nanoseconds", help: "The number of nanoseconds elapsed to restart the OpenSearch pod"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewRestartWorker()
			w := wi.(worker)
			assert.NoError(t, err)
			dl := w.GetMetricDescList()
			var found int
			for _, d := range dl {
				s := d.String()
				if strings.Contains(s, test.fqName) && strings.Contains(s, test.help) {
					found++
				}
			}
			assert.Equal(t, 1, found)
		})
	}
}

// TestGetMetricList tests the GetMetricList method
// GIVEN a worker
//
//	WHEN the GetMetricList methods is called
//	THEN ensure that the correct results are returned
func TestGetMetricList(t *testing.T) {
	origFunc := overridePsrClient()
	defer func() {
		funcNewPsrClient = origFunc
	}()

	envMap := map[string]string{
		openSearchTier: opensearchpsr.MasterTier,
	}
	f := fakeEnv{data: envMap}
	saveEnv := osenv.GetEnvFunc
	osenv.GetEnvFunc = f.GetEnv
	defer func() {
		osenv.GetEnvFunc = saveEnv
	}()

	tests := []struct {
		name   string
		fqName string
		help   string
	}{
		{name: "1", fqName: metricsPrefix + "_pod_restart_count", help: "The total number of OpenSearch pod restarts"},
		{name: "2", fqName: metricsPrefix + "_pod_restart_time_nanoseconds", help: "The number of nanoseconds elapsed to restart the OpenSearch pod"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewRestartWorker()
			w := wi.(worker)
			assert.NoError(t, err)
			ml := w.GetMetricList()
			var found int
			for _, m := range ml {
				s := m.Desc().String()
				if strings.Contains(s, test.fqName) && strings.Contains(s, test.help) {
					found++
				}
			}
			assert.Equal(t, 1, found)
		})
	}
}

// TestDoWork tests the DoWork method
// GIVEN a worker
//
//	WHEN the DoWork methods is called
//	THEN ensure that the correct results are returned
func TestDoWork(t *testing.T) {
	readyState := "ready"
	notReadyState := "notready"
	podExistsState := "podexists"
	podUID := "poduid"

	tests := []struct {
		name  string
		tier  string
		state string
	}{
		{
			name:  "master-ready",
			tier:  opensearchpsr.MasterTier,
			state: readyState,
		},
		{
			name:  "data-ready",
			tier:  opensearchpsr.DataTier,
			state: readyState,
		},
		{
			name:  "ingest-ready",
			tier:  opensearchpsr.IngestTier,
			state: readyState,
		},
		{
			name:  "master-not-ready",
			tier:  opensearchpsr.MasterTier,
			state: notReadyState,
		},
		{
			name:  "data-not-ready",
			tier:  opensearchpsr.DataTier,
			state: notReadyState,
		},
		{
			name:  "ingest-not-ready",
			tier:  opensearchpsr.IngestTier,
			state: notReadyState,
		},
		{
			name:  "master-pod-exists",
			tier:  opensearchpsr.MasterTier,
			state: podExistsState,
		},
		{
			name:  "data-pod-exists",
			tier:  opensearchpsr.DataTier,
			state: podExistsState,
		},
		{
			name:  "ingest-pod-exists",
			tier:  opensearchpsr.IngestTier,
			state: podExistsState,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			envMap := map[string]string{
				openSearchTier: test.tier,
			}
			f := fakeEnv{data: envMap}
			saveEnv := osenv.GetEnvFunc
			osenv.GetEnvFunc = f.GetEnv
			defer func() {
				osenv.GetEnvFunc = saveEnv
			}()

			// Setup fake VZ client
			cr := &v1beta1.Verrazzano{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testVZ",
				},
			}
			vzclient := vpoFakeClient.NewSimpleClientset(cr)

			// Setup fake K8s client
			podLabels := getTierLabels(test.tier)
			masterSTSLabel := map[string]string{"verrazzano-component": "opensearch"}
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			_ = k8sapiext.AddToScheme(scheme)
			_ = v1alpha1.AddToScheme(scheme)
			_ = appsv1.AddToScheme(scheme)
			builder := crtFakeClient.NewClientBuilder().WithScheme(scheme)
			if test.state == readyState {
				builder = builder.WithObjects(initFakePodWithLabels(podLabels)).WithLists(initReadyDeployments(podLabels), initReadyStatefulSets(masterSTSLabel))
			} else if test.state == notReadyState {
				builder = builder.WithLists(initNotReadyDeployments(podLabels), initNotReadyStatefulSets(masterSTSLabel))
			} else {
				builder = builder.WithObjects(initFakePodWithLabels(podLabels)).WithLists(initReadyDeployments(podLabels), initReadyStatefulSets(masterSTSLabel))
			}
			crtClient := builder.Build()

			// Load the PsrClient with both fake clients
			psrClient := fakePsrClient{
				psrClient: &k8sclient.PsrClient{
					CrtlRuntime: crtClient,
					VzInstall:   vzclient,
				},
			}
			origFc := funcNewPsrClient
			defer func() {
				funcNewPsrClient = origFc
			}()
			funcNewPsrClient = psrClient.NewPsrClient

			// Create worker and call dowork
			wi, err := NewRestartWorker()
			assert.NoError(t, err)
			w := wi.(worker)
			err = config.PsrEnv.LoadFromEnv(w.GetEnvDescList())
			assert.NoError(t, err)
			if test.state == podExistsState {
				w.restartedPodUID = types.UID(podUID)
			}
			err = w.DoWork(config.CommonConfig{
				WorkerType: "restart",
			}, vzlog.DefaultLogger())
			if test.state == readyState {
				assert.NoError(t, err)
				assert.True(t, w.restartStartTime > 0)
				assert.Equal(t, w.restartedPodUID, types.UID(podUID))
			} else if test.state == notReadyState {
				assert.Error(t, err)
				assert.False(t, w.restartStartTime > 0)
				assert.NotEqual(t, w.restartedPodUID, types.UID(podUID))
			} else {
				assert.Error(t, err)
				assert.False(t, w.restartStartTime > 0)
				assert.Equal(t, w.restartedPodUID, types.UID(podUID))
			}
		})
	}
}

// initFakePodWithLabels inits a fake Pod with specified image and labels
func initFakePodWithLabels(labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testPod",
			Namespace: "verrazzano-system",
			Labels:    labels,
			UID:       "poduid",
		},
	}
}

func initReadyDeployments(labels map[string]string) *appsv1.DeploymentList {
	return &appsv1.DeploymentList{
		Items: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vmi-system-es-ingest",
					Namespace: constants.VerrazzanoSystemNamespace,
					Labels:    labels,
				},
				Status: appsv1.DeploymentStatus{
					Replicas:      3,
					ReadyReplicas: 3,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vmi-system-es-data-0",
					Namespace: constants.VerrazzanoSystemNamespace,
					Labels:    labels,
				},
				Status: appsv1.DeploymentStatus{
					Replicas:      1,
					ReadyReplicas: 1,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vmi-system-es-data-1",
					Namespace: constants.VerrazzanoSystemNamespace,
					Labels:    labels,
				},
				Status: appsv1.DeploymentStatus{
					Replicas:      1,
					ReadyReplicas: 1,
				},
			},
		},
	}
}

func initNotReadyDeployments(labels map[string]string) *appsv1.DeploymentList {
	return &appsv1.DeploymentList{
		Items: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vmi-system-es-ingest",
					Namespace: constants.VerrazzanoSystemNamespace,
					Labels:    labels,
				},
				Status: appsv1.DeploymentStatus{
					Replicas:      2,
					ReadyReplicas: 3,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vmi-system-es-data-0",
					Namespace: constants.VerrazzanoSystemNamespace,
					Labels:    labels,
				},
				Status: appsv1.DeploymentStatus{
					Replicas:      1,
					ReadyReplicas: 1,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vmi-system-es-data-1",
					Namespace: constants.VerrazzanoSystemNamespace,
					Labels:    labels,
				},
				Status: appsv1.DeploymentStatus{
					Replicas:      0,
					ReadyReplicas: 1,
				},
			},
		},
	}
}

func initReadyStatefulSets(labels map[string]string) *appsv1.StatefulSetList {
	return &appsv1.StatefulSetList{
		Items: []appsv1.StatefulSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vmi-system-es-master",
					Namespace: constants.VerrazzanoSystemNamespace,
					Labels:    labels,
				},
				Status: appsv1.StatefulSetStatus{
					Replicas:      3,
					ReadyReplicas: 3,
				},
			},
		},
	}
}

func initNotReadyStatefulSets(labels map[string]string) *appsv1.StatefulSetList {
	return &appsv1.StatefulSetList{
		Items: []appsv1.StatefulSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vmi-system-es-master",
					Namespace: constants.VerrazzanoSystemNamespace,
					Labels:    labels,
				},
				Status: appsv1.StatefulSetStatus{
					Replicas:      2,
					ReadyReplicas: 3,
				},
			},
		},
	}
}

func (f *fakeEnv) GetEnv(key string) string {
	return f.data[key]
}

func (f *fakePsrClient) NewPsrClient() (k8sclient.PsrClient, error) {
	return *f.psrClient, nil
}

func getTierLabels(tier string) map[string]string {
	switch tier {
	case opensearchpsr.MasterTier:
		return map[string]string{"opensearch.verrazzano.io/role-master": "true"}
	case opensearchpsr.DataTier:
		return map[string]string{"opensearch.verrazzano.io/role-data": "true"}
	case opensearchpsr.IngestTier:
		return map[string]string{"opensearch.verrazzano.io/role-ingest": "true"}
	default:
		return nil
	}
}

func overridePsrClient() func() (k8sclient.PsrClient, error) {
	f := fakePsrClient{
		psrClient: &k8sclient.PsrClient{},
	}
	origFc := funcNewPsrClient
	funcNewPsrClient = f.NewPsrClient
	return origFc
}
