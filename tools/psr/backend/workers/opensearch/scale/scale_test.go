// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scale

import (
	"strings"
	"testing"
	"time"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoFakeClient "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned/fake"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/pkg/k8sclient"
	opensearchpsr "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/opensearch"
	"github.com/verrazzano/verrazzano/tools/psr/backend/pkg/verrazzano"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	k8sapiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

	w, err := NewScaleWorker()
	assert.NoError(t, err)

	wd := w.GetWorkerDesc()
	assert.Equal(t, config.WorkerTypeScale, wd.WorkerType)
	assert.Equal(t, "The OpenSearch scale worker scales an OpenSearch tier in and out continuously", wd.Description)
	assert.Equal(t, config.WorkerTypeScale, wd.MetricsName)

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
		{name: "2",
			key:      minReplicaCount,
			defval:   "3",
			required: false,
		},
		{name: "3",
			key:      maxReplicaCount,
			defval:   "5",
			required: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, err := NewScaleWorker()
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

	tests := []struct {
		name   string
		fqName string
		help   string
	}{
		{name: "1", fqName: "opensearch_scale_out_count_total", help: "The total number of times OpenSearch scaled out"},
		{name: "2", fqName: "opensearch_scale_in_count_total", help: "The total number of times OpenSearch scaled in"},
		{name: "3", fqName: "opensearch_scale_out_seconds", help: "The number of seconds elapsed to scale out OpenSearch"},
		{name: "4", fqName: "opensearch_scale_in_seconds", help: "The number of seconds elapsed to scale in OpenSearch"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewScaleWorker()
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

	tests := []struct {
		name   string
		fqName string
		help   string
	}{
		{name: "1", fqName: "opensearch_scale_out_count_total", help: "The total number of times OpenSearch scaled out"},
		{name: "2", fqName: "opensearch_scale_in_count_total", help: "The total number of times OpenSearch scaled in"},
		{name: "3", fqName: "opensearch_scale_out_seconds", help: "The number of seconds elapsed to scale out OpenSearch"},
		{name: "4", fqName: "opensearch_scale_in_seconds", help: "The number of seconds elapsed to scale in OpenSearch"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewScaleWorker()
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
	tests := []struct {
		name          string
		tier          string
		expectError   bool
		minReplicas   string
		maxReplicas   string
		skipUpdate    bool
		skipPodCreate bool
		firstState    v1alpha1.VzStateType
		secondState   v1alpha1.VzStateType
		thirtState    v1alpha1.VzStateType
	}{
		{
			name:        "master",
			tier:        opensearchpsr.MasterTier,
			expectError: false,
			minReplicas: "3",
			maxReplicas: "4",
			firstState:  v1alpha1.VzStateReady,
			secondState: v1alpha1.VzStateReconciling,
		},
		{
			name:        "data",
			tier:        opensearchpsr.DataTier,
			expectError: false,
			minReplicas: "3",
			maxReplicas: "4",
			firstState:  v1alpha1.VzStateReady,
			secondState: v1alpha1.VzStateReconciling,
		},
		{
			name:        "ingest",
			tier:        opensearchpsr.IngestTier,
			expectError: false,
			minReplicas: "3",
			maxReplicas: "4",
			firstState:  v1alpha1.VzStateReady,
			secondState: v1alpha1.VzStateReconciling,
		},
		{
			name:        "replicaErr",
			tier:        opensearchpsr.MasterTier,
			skipUpdate:  true,
			expectError: true,
			minReplicas: "1",
			maxReplicas: "4",
			firstState:  v1alpha1.VzStateReady,
		},
		// Test missing pods
		{
			name:          "noPods",
			tier:          opensearchpsr.MasterTier,
			skipUpdate:    true,
			skipPodCreate: true,
			expectError:   true,
			minReplicas:   "1",
			maxReplicas:   "4",
			firstState:    v1alpha1.VzStateReady,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			envMap := map[string]string{
				openSearchTier:  test.tier,
				minReplicaCount: test.minReplicas,
				maxReplicaCount: test.maxReplicas,
			}
			f := fakeEnv{data: envMap}
			saveEnv := osenv.GetEnvFunc
			osenv.GetEnvFunc = f.GetEnv
			defer func() {
				osenv.GetEnvFunc = saveEnv
			}()

			// Setup fake VZ client
			cr := initFakeVzCr(test.firstState)
			vzclient := vpoFakeClient.NewSimpleClientset(cr)

			// Setup fake K8s client
			podLabels := getTierLabels(test.tier)
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			_ = k8sapiext.AddToScheme(scheme)
			_ = v1alpha1.AddToScheme(scheme)
			builder := crtFakeClient.NewClientBuilder().WithScheme(scheme)
			if !test.skipPodCreate {
				builder = builder.WithObjects(initFakePodWithLabels(podLabels))
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
			wi, err := NewScaleWorker()
			assert.NoError(t, err)
			w := wi.(worker)
			err = config.PsrEnv.LoadFromEnv(w.GetEnvDescList())
			assert.NoError(t, err)

			// DoWork expects the Verrazzano CR state to change.
			// Worker waits for CR to be ready, modifies it, then waits for it to be not ready (e.g. reconciling)
			// Run a background thread to change the state after the work starts
			if !test.skipUpdate {
				go func() {
					time.Sleep(1 * time.Second)
					cr := initFakeVzCr(test.secondState)
					verrazzano.UpdateVerrazzano(vzclient, cr)
					if len(test.thirtState) > 0 {
						time.Sleep(1 * time.Second)
						cr := initFakeVzCr(test.thirtState)
						verrazzano.UpdateVerrazzano(vzclient, cr)
					}
				}()
			}

			err = w.DoWork(config.CommonConfig{
				WorkerType: "scale",
			}, vzlog.DefaultLogger())
			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
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
		},
	}
}

// initFakeVzCr inits a fake Verrazzano CR
func initFakeVzCr(state v1alpha1.VzStateType) *v1alpha1.Verrazzano {
	return &v1alpha1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testPod",
			Namespace: "verrazzano-system",
		},
		Status: v1alpha1.VerrazzanoStatus{
			State: state,
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
