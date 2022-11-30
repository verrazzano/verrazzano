// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package alerts

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoFakeClient "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned/fake"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/pkg/k8sclient"
	psrprom "github.com/verrazzano/verrazzano/tools/psr/backend/pkg/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	crtFakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"
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
	w, err := NewReceiveAlertsWorker()
	assert.NoError(t, err)

	wd := w.GetWorkerDesc()
	assert.Equal(t, config.WorkerTypeHTTPGet, wd.WorkerType)
	assert.Equal(t, "The alerts receiver worker configures alertmanger and receives alerts and writes them to events", wd.Description)
	assert.Equal(t, metricsPrefix, wd.MetricsPrefix)

	logged := w.WantLoopInfoLogged()
	assert.False(t, logged)
}

func TestGetMetricDescList(t *testing.T) {
	tests := []struct {
		name   string
		fqName string
		help   string
	}{
		{name: "1", fqName: "alerts_firing_received_count", help: "The total number of alerts received from alertmanager"},
		{name: "2", fqName: "alerts_resolved_received_count", help: "The total number of alerts received from alertmanager"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			wi, err := NewReceiveAlertsWorker()
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

func TestGetMetricList(t *testing.T) {
	tests := []struct {
		name   string
		fqName string
		help   string
	}{
		{name: "1", fqName: "alerts_firing_received_count", help: "The total number of alerts received from alertmanager"},
		{name: "2", fqName: "alerts_resolved_received_count", help: "The total number of alerts received from alertmanager"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewReceiveAlertsWorker()
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

//
//// TestDoWork tests the DoWork method
//// GIVEN a worker
////
////	WHEN the DoWork methods is called
////	THEN ensure that the correct results are returned
//func TestDoWork(t *testing.T) {
//	tests := []struct {
//		name         string
//		bodyData     string
//		getError     error
//		doworkError  error
//		statusCode   int
//		nilResp      bool
//		reqCount     int
//		successCount int
//		failureCount int
//	}{
//		{
//			name:         "1",
//			bodyData:     "testsuccess",
//			statusCode:   200,
//			reqCount:     1,
//			successCount: 1,
//			failureCount: 0,
//		},
//		{
//			name:         "2",
//			bodyData:     "testerror",
//			getError:     errors.New("error"),
//			reqCount:     1,
//			successCount: 0,
//			failureCount: 1,
//		},
//		{
//			name:         "3",
//			bodyData:     "testRespError",
//			statusCode:   500,
//			reqCount:     1,
//			successCount: 0,
//			failureCount: 1,
//		},
//		{
//			name:         "4",
//			bodyData:     "testNilResp",
//			doworkError:  errors.New("GET request to endpoint received a nil response"),
//			nilResp:      true,
//			reqCount:     1,
//			successCount: 0,
//			failureCount: 1,
//		},
//	}
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			f := httpGetFunc
//			defer func() {
//				httpGetFunc = f
//			}()
//			var resp *http.Response
//			if !test.nilResp {
//				resp = &http.Response{
//					StatusCode:    test.statusCode,
//					Body:          &fakeBody{data: test.bodyData},
//					ContentLength: int64(len(test.bodyData)),
//				}
//			}
//			httpGetFunc = fakeHTTP{
//				bodyData: test.bodyData,
//				error:    test.getError,
//				resp:     resp,
//			}.Get
//
//			wi, err := NewHTTPGetWorker()
//			assert.NoError(t, err)
//			w := wi.(worker)
//			err = w.DoWork(config.CommonConfig{
//				WorkerType: "Fake",
//			}, vzlog.DefaultLogger())
//			if test.doworkError == nil && test.getError == nil {
//				assert.NoError(t, err)
//			} else {
//				assert.Error(t, err)
//			}
//
//			assert.Equal(t, int64(test.reqCount), w.getRequestsCountTotal.Val)
//			assert.Equal(t, int64(test.successCount), w.getRequestsSucceededCountTotal.Val)
//			assert.Equal(t, int64(test.failureCount), w.getRequestsFailedCountTotal.Val)
//		})
//	}
//}

func Test_updateVZForAlertmanager_updateVZCR(t *testing.T) {
	envMap := map[string]string{
		config.PsrWorkerType:        config.WorkerTypeReceiveAlerts,
		config.PsrWorkerReleaseName: "test-alerts",
		config.PsrWorkerNamespace:   "test-psr",
	}
	f := fakeEnv{data: envMap}
	saveEnv := osenv.GetEnvFunc
	osenv.GetEnvFunc = f.GetEnv
	defer func() {
		osenv.GetEnvFunc = saveEnv
	}()

	// Setup fake VZ client
	cr := &v1alpha1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vz",
			Namespace: "test-ns",
		},
	}
	vzclient := vpoFakeClient.NewSimpleClientset(cr)

	// Setup fake K8s client
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	builder := crtFakeClient.NewClientBuilder().WithScheme(scheme).WithObjects(cr)
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

	log := vzlog.DefaultLogger()
	conf, err := config.GetCommonConfig(log)
	err = updateVZForAlertmanager(log, conf)
	assert.NoError(t, err)

	var cm corev1.ConfigMap
	err = psrClient.psrClient.CrtlRuntime.Get(context.TODO(), types.NamespacedName{
		Name:      psrprom.AlertmanagerCMName,
		Namespace: cr.Namespace,
	}, &cm)
	assert.NoError(t, err)
	assert.Equal(t, `alertmanager:
  alertmanagerSpec:
    podMetadata:
      annotations:
        sidecar.istio.io/inject: "false"
  config:
    receivers:
    - webhook_configs:
      - url: http://test-alerts-http-alerts.test-psr:9090/alerts
      name: webhook
    route:
      group_by:
      - alertname
      receiver: webhook
      routes:
      - match:
          alertname: Watchdog
        receiver: webhook
  enabled: true
`, cm.Data[psrprom.AlertmanagerCMKey])

	cr, err = psrClient.psrClient.VzInstall.VerrazzanoV1alpha1().Verrazzanos(cr.Namespace).Get(context.TODO(), cr.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: psrprom.AlertmanagerCMName,
		},
		Key: psrprom.AlertmanagerCMKey,
	}, cr.Spec.Components.PrometheusOperator.ValueOverrides[0].ConfigMapRef)
}

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
