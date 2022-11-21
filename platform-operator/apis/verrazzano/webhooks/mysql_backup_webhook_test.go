// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

var mySQLJobAnnotation = map[string]string{
	"app.kubernetes.io/created-by": "mysql-wls",
}

const (
	jobAPIVerison = "batch/v1"
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
// WHEN Handle is called with an admission.Request containing job not created by mysql-wls
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
	assert.Equal(t, metav1.StatusReason("No action required, job not labelled with app.kubernetes.io/created-by: mysql-wls"), res.Result.Reason)
}

// TestHandleAnnotateMysqlBackupJob tests handling an admission.Request
// GIVEN a MysqlBackupJobWebhook and an admission.Request
// WHEN Handle is called with an admission.Request containing job created by mysql-wls
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
			Labels:    mySQLJobAnnotation,
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

// TestConvertAPIVersionToGroupAndVersion validates whether  given apiVersion
// will be successfully segregated into group and version

func TestConvertAPIVersionToGroupAndVersion(t *testing.T) {
	apiVersion1 := "networking.k8s.io/v1"
	apiVersion2 := "v1alpha1"
	var api1part1, api1part2 string
	api1part1, api1part2 = ConvertAPIVersionToGroupAndVersion(apiVersion1)
	assert.Equal(t, "networking.k8s.io", api1part1, "part1 is networking.k8s.io")
	assert.Equal(t, "v1", api1part2, "part2 is v1")
	api1part1, api1part2 = ConvertAPIVersionToGroupAndVersion(apiVersion2)
	assert.Equal(t, "", api1part1, "part1  is empty and correct")
	assert.Equal(t, "v1alpha1", api1part2, "part2 is v1alpha1")
}

// TestIsCronJobCreatedByMysqlOperator tests handling an admission.Request
// GIVEN a MysqlBackupJobWebhook and an admission.Request
// WHEN Handle is called with an admission.Request containing cronjob created by mysql-wls
// THEN Handle should return an Allowed response with no action required
func TestIsCronJobCreatedByMysqlOperator(t *testing.T) {
	var err error
	defaulter := &MySQLBackupJobWebhook{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}
	u := newUnstructured(jobAPIVerison, "CronJob", "MySqlCronJob")
	resource := schema.GroupVersionResource{
		Group:    "batch",
		Version:  "v1",
		Resource: "cronjobs",
	}
	_, err = defaulter.DynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating cronjob resource ")
	cronjob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MySqlCronJob",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "CronJob",
					APIVersion: jobAPIVerison,
					Name:       "MySqlCronJob",
				},
			},
			Labels: mySQLJobAnnotation,
		},
	}
	job, err := defaulter.KubeClient.BatchV1().CronJobs("default").Create(context.TODO(), cronjob, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating cronjob")
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

// TestIsCronJobCreatedByMysqlOperator2 tests handling an admission.Request
// GIVEN a MysqlBackupJobWebhook and an admission.Request
// WHEN Handle is called with an admission.Request containing cronjob created by mysql-wls
// and cronjob didn't exist
// THEN Handle should not allow response with no action required

func TestIsCronJobCreatedByMysqlOperator2(t *testing.T) {
	var err error
	defaulter := &MySQLBackupJobWebhook{
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		KubeClient:    fake.NewSimpleClientset(),
	}
	u := newUnstructured(jobAPIVerison, "CronJob", "MySqlCronJob1")
	resource := schema.GroupVersionResource{
		Group:    "batch",
		Version:  "v1",
		Resource: "cronjobs",
	}
	_, err = defaulter.DynamicClient.Resource(resource).Namespace("abc").Create(context.TODO(), u, metav1.CreateOptions{})
	assert.EqualError(t, err, "request namespace does not match object namespace, request: \"abc\" object: \"default\"")
	cronjob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MySqlCronJob1",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "CronJob",
					APIVersion: jobAPIVerison,
					Name:       "MySqlCronJob1",
				},
			},
			Labels: mySQLJobAnnotation,
		},
	}
	job, err := defaulter.KubeClient.BatchV1().CronJobs("default").Create(context.TODO(), cronjob, metav1.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating cronjob")
	decoder := decoder()
	err = defaulter.InjectDecoder(decoder)
	assert.NoError(t, err, "Unexpected error injecting decoder")
	req := admission.Request{}
	req.Namespace = "default"
	marshaledJob, err := json.Marshal(job)
	assert.NoError(t, err, "Unexpected error marshaling job")
	req.Object = runtime.RawExtension{Raw: marshaledJob}
	res := defaulter.Handle(context.TODO(), req)
	assert.False(t, res.Allowed)
	assert.NoError(t, err, "Unexpected error marshaling job")
}

// Helper function which return unstructured object
func newUnstructured(apiVersion string, kind string, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": "default",
				"name":      name,
			},
		},
	}
}
