package mysql_webhook

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

func decoder() *admission.Decoder {
	scheme := runtime.NewScheme()
	_ = core.AddToScheme(scheme)
	decoder, err := admission.NewDecoder(scheme)
	if err != nil {
		zap.S().Errorf("Failed creating new decoder: %v", err)
	}
	return decoder
}

// TestHandleBadRequest tests handling an invalid admission.Request
// GIVEN an MysqlBackupJobWebhook and an admission.Request
// WHEN Handle is called with an invalid admission.Request containing no content
// THEN Handle should return an error with http.StatusBadRequest
func TestHandleBadRequestMySQL(t *testing.T) {

	decoder := decoder()
	defaulter := &MySQLBackupJobWebhook{}
	err := defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
	req := admission.Request{}
	res := defaulter.Handle(context.TODO(), req)
	assert.False(t, res.Allowed)
	assert.Equal(t, int32(http.StatusBadRequest), res.Result.Code)
}

// TestHandleIstioDisabledMysqlBackupJob tests handling an admission.Request
// GIVEN a MysqlBackupJobWebhook and an admission.Request
// WHEN Handle is called with an admission.Request containing a job resource with MysqlHA disabled
// THEN Handle should return an Allowed response with no action required
func TestHandleIstioDisabledMysqlBackupJob(t *testing.T) {

	defaulter := &MySQLBackupJobWebhook{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}
	// Create a job with Istio injection disabled
	p := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-disabled",
			Namespace: "default",
			Annotations: map[string]string{
				"sidecar.istio.io/inject": "false",
			},
		},
	}

	job, err := defaulter.KubeClient.BatchV1().Jobs("default").Create(context.TODO(), p, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating job")

	decoder := decoder()
	err = defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
	req := admission.Request{}
	req.Namespace = "default"
	marshaledJob, err := json.Marshal(job)
	assert.NoError(t, err, "Unexpected error marshaling job")
	req.Object = runtime.RawExtension{Raw: marshaledJob}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Equal(t, metav1.StatusReason("No action required, job labeled with sidecar.istio.io/inject: false"), res.Result.Reason)
}

// TestHandleSkipAnnotateMysqlBackupJob tests handling an admission.Request
// GIVEN a MysqlBackupJobWebhook and an admission.Request
// WHEN Handle is called with an admission.Request containing job not created by mysql-operator
// THEN Handle should return an Allowed response with no action required
func TestHandleSkipAnnotateMysqlBackupJob(t *testing.T) {

	defaulter := &MySQLBackupJobWebhook{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}
	// Create a job with Istio injection disabled
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "random-job",
			Namespace: "default",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
	}

	job, err := defaulter.KubeClient.BatchV1().Jobs("default").Create(context.TODO(), job, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating job")

	decoder := decoder()
	err = defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
	req := admission.Request{}
	req.Namespace = "default"
	marshaledJob, err := json.Marshal(job)
	assert.NoError(t, err, "Unexpected error marshaling job")
	req.Object = runtime.RawExtension{Raw: marshaledJob}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Equal(t, metav1.StatusReason("No action required, job not labelled with app.kubernetes.io/created-by: mysql-operator"), res.Result.Reason)
}

// TestHandleAnnotateMysqlBackupJob tests handling an admission.Request
// GIVEN a MysqlBackupJobWebhook and an admission.Request
// WHEN Handle is called with an admission.Request containing job created by mysql-operator
// THEN Handle should return an Allowed response with no action required
func TestHandleAnnotateMysqlBackupJob(t *testing.T) {

	defaulter := &MySQLBackupJobWebhook{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}
	// Create a job with Istio injection disabled
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backup-job",
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/created-by": "mysql-operator",
			},
		},
	}

	job, err := defaulter.KubeClient.BatchV1().Jobs("default").Create(context.TODO(), job, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating job")

	decoder := decoder()
	err = defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
	req := admission.Request{}
	req.Namespace = "default"
	marshaledJob, err := json.Marshal(job)
	assert.NoError(t, err, "Unexpected error marshaling job")
	req.Object = runtime.RawExtension{Raw: marshaledJob}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.NoError(t, err, "Unexpected error marshaling job")
}
