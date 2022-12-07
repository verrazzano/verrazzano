// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scale

import (
	"github.com/verrazzano/verrazzano/tools/psr/backend/pkg/k8sclient"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
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
	envMap := map[string]string{
		DomainUID:       "test-domain",
		DomainNamespace: "test-namespace",
		MinReplicaCount: "1",
		MaxReplicaCount: "2",
	}
	f := fakeEnv{data: envMap}
	saveEnv := osenv.GetEnvFunc
	osenv.GetEnvFunc = f.GetEnv
	defer func() {
		osenv.GetEnvFunc = saveEnv
	}()

	w, err := NewScaleWorker()
	assert.NoError(t, err)

	wd := w.GetWorkerDesc()
	assert.Equal(t, config.WorkerTypeWlsScale, wd.WorkerType)
	assert.Equal(t, "The scale domain worker scales up and scales down the domain", wd.Description)
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
	envMap := map[string]string{
		DomainUID:       "test-domain",
		DomainNamespace: "test-namespace",
		MinReplicaCount: "1",
		MaxReplicaCount: "2",
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
			key:      DomainUID,
			defval:   "",
			required: true,
		},
		{name: "2",
			key:      DomainNamespace,
			defval:   "",
			required: true,
		},
		{name: "3",
			key:      MinReplicaCount,
			defval:   "2",
			required: true,
		},
		{name: "4",
			key:      MaxReplicaCount,
			defval:   "4",
			required: true,
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
	envMap := map[string]string{
		DomainUID:       "test-domain",
		DomainNamespace: "test-namespace",
		MinReplicaCount: "2",
		MaxReplicaCount: "4",
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
		{name: "1", fqName: "scale_up_domain_count_total", help: "The total number of successful scale up domain requests"},
		{name: "2", fqName: "scale_down_domain_count_total", help: "The total number of failed scale down domain requests"},
		{name: "3", fqName: "scale_up_seconds", help: "The total number of seconds elapsed to scale up the domain"},
		{name: "4", fqName: "scale_down_seconds", help: "The total number of seconds elapsed to scale down the domain"},
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

	envMap := map[string]string{
		DomainUID:       "test-domain",
		DomainNamespace: "test-namespace",
		MinReplicaCount: "2",
		MaxReplicaCount: "4",
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
		{name: "1", fqName: "scale_up_domain_count_total", help: "The total number of successful scale up domain requests"},
		{name: "2", fqName: "scale_down_domain_count_total", help: "The total number of failed scale down domain requests"},
		{name: "3", fqName: "scale_up_seconds", help: "The total number of seconds elapsed to scale up the domain"},
		{name: "4", fqName: "scale_down_seconds", help: "The total number of seconds elapsed to scale down the domain"},
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
/* func TestDoWork(t *testing.T) {
	envMap := map[string]string{
		DomainUID:       "test-domain",
		DomainNamespace: "test-namespace",
		MinReplicaCount: "2",
		MaxReplicaCount: "4",
	}
	f := fakeEnv{data: envMap}
	saveEnv := osenv.GetEnvFunc
	osenv.GetEnvFunc = f.GetEnv
	defer func() {
		osenv.GetEnvFunc = saveEnv
	}()

	tests := []struct {
		name            string
		currentReplicas string
		readyReplicas   string
		minReplicas     string
		maxReplicas     string
		expectError     bool
	}{
		{
			name:            "scaleup",
			currentReplicas: "2",
			readyReplicas:   "4",
			minReplicas:     "2",
			maxReplicas:     "4",
			expectError:     false,
		},
		{
			name:            "scaledown",
			currentReplicas: "4",
			readyReplicas:   "2",
			minReplicas:     "2",
			maxReplicas:     "4",
			expectError:     false,
		},
		{
			name:            "scaleupfailed",
			currentReplicas: "2",
			readyReplicas:   "2",
			minReplicas:     "2",
			maxReplicas:     "4",
			expectError:     true,
		},
		{
			name:            "scaledownfailed",
			currentReplicas: "4",
			readyReplicas:   "4",
			minReplicas:     "2",
			maxReplicas:     "4",
			expectError:     true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			envMap := map[string]string{
				DomainUID:       "test-domain",
				DomainNamespace: "test-namespace",
				MinReplicaCount: "2",
				MaxReplicaCount: "4",
			}
			f := fakeEnv{data: envMap}
			saveEnv := osenv.GetEnvFunc
			osenv.GetEnvFunc = f.GetEnv
			defer func() {
				osenv.GetEnvFunc = saveEnv
			}()

			// Setup fake VZ client
			cr := initFakeVzCr(v1alpha1.VzStateReady)
			vzclient := vpoFakeClient.NewSimpleClientset(cr)

			// Setup fake K8s client
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			_ = k8sapiext.AddToScheme(scheme)
			_ = v1alpha1.AddToScheme(scheme)
			builder := crtFakeClient.NewClientBuilder().WithScheme(scheme)

			crtClient := builder.Build()
			// Load the PsrClient with fake clients
			psrClient := fakePsrClient{
				psrClient: &k8sclient.PsrClient{
					CrtlRuntime: crtClient,
					VzInstall:   vzclient,
					DynClient:   dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
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
} */

func (f *fakeEnv) GetEnv(key string) string {
	return f.data[key]
}

func (f *fakePsrClient) NewPsrClient() (k8sclient.PsrClient, error) {
	return *f.psrClient, nil
}

func overridePsrClient() func() (k8sclient.PsrClient, error) {
	f := fakePsrClient{
		psrClient: &k8sclient.PsrClient{},
	}
	origFc := funcNewPsrClient
	funcNewPsrClient = f.NewPsrClient
	return origFc
}
