// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	istiofake "istio.io/client-go/pkg/clientset/versioned/fake"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

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
		IstioClient:   istiofake.NewSimpleClientset(),
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
		IstioClient:   istiofake.NewSimpleClientset(),
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
		IstioClient:   istiofake.NewSimpleClientset(),
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

// TestHandleAnnotateMysqlBackupCronJob tests handling an admission.Request
// GIVEN a MysqlBackupJobWebhook and an admission.Request
// WHEN Handle is called with an admission.Request containing job created by mysql-operator
// THEN Handle should return an Allowed response with no action required
//func TestHandleAnnotateMysqlBackupCronJob(t *testing.T) {
//
//	defaulter := &MySQLBackupJobWebhook{
//		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
//		KubeClient:    fake.NewSimpleClientset(),
//		IstioClient:   istiofake.NewSimpleClientset(),
//	}
//	// Create a job with Cronjob owner refernce
//	job := &batchv1.Job{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "backup-job",
//			Namespace: "default",
//			//Labels: map[string]string{
//			//	"app.kubernetes.io/created-by": "mysql-operator",
//			//},
//			OwnerReferences: []metav1.OwnerReference{
//				metav1.OwnerReference{
//					Kind: "CronJob",
//					Name: "BackupFoo",
//				},
//			},
//		},
//	}
//
//	cjob := &batchv1.CronJob{
//		ObjectMeta: metav1.ObjectMeta{
//			Labels: map[string]string{
//				"app.kubernetes.io/created-by": "mysql-operator",
//			},
//			OwnerReferences: []metav1.OwnerReference{
//				metav1.OwnerReference{
//					Kind: "InnoDBCluster",
//					Name: "mysql",
//				},
//			},
//			Name: "BackupFoo",
//		},
//		//Spec: batchv1.CronJobSpec{
//		//	JobTemplate: batchv1.JobTemplateSpec{
//		//		Spec: job,
//		//	}
//		//},
//	}
//
//	job, err := defaulter.KubeClient.BatchV1().Jobs("default").Create(context.TODO(), job, metav1.CreateOptions{})
//	_, err = defaulter.KubeClient.BatchV1().CronJobs("default").Create(context.TODO(), cjob, metav1.CreateOptions{})
//	assert.NoError(t, err, "Unexpected error creating job")
//
//	decoder := decoder()
//	err = defaulter.InjectDecoder(decoder)
//	assert.NoError(t, err, "Unexpected error injecting decoder")
//	req := admission.Request{}
//	req.Namespace = "default"
//	marshaledJob, err := json.Marshal(job)
//	assert.NoError(t, err, "Unexpected error marshaling job")
//	req.Object = runtime.RawExtension{Raw: marshaledJob}
//	res := defaulter.Handle(context.TODO(), req)
//	assert.True(t, res.Allowed)
//	assert.NoError(t, err, "Unexpected error marshaling job")
//}

// TestIstioMysqlBackupHandleFailed tests to make sure the failure metric is being exposed
func TestIstioMysqlBackupHandleFailed(t *testing.T) {

	assert := assert.New(t)
	// Create a request and decode(Handle) it
	decoder := decoder()
	defaulter := &IstioWebhook{}
	_ = defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	defaulter.Handle(context.TODO(), req)
	reconcileerrorCounterObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.MysqlHaHandleError)
	assert.NoError(err)
	// Expect a call to fetch the error
	reconcileFailedCounterBefore := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	reconcileerrorCounterObject.Get().Inc()
	reconcileFailedCounterAfter := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	assert.Equal(reconcileFailedCounterBefore, reconcileFailedCounterAfter-1)
}
