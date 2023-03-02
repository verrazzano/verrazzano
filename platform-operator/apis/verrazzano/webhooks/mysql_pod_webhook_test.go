// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

// TestHandleSkipAnnotateMysqlBackupJob tests handling an admission.Request
// GIVEN a MysqlBackupJobWebhook and an admission.Request
// WHEN Handle is called with an admission.Request containing job not created by mysql-operator
// THEN Handle should return an Allowed response with no action required
func TestHandleSkipAnnotateMysqlBackupJob1(t *testing.T) {

	tests := []struct {
		name     string
		pod      *corev1.Pod
		reason   string
		patchLen int
	}{
		{
			name: "Successful patch",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-0",
					Namespace: "keycloak",
					Labels: map[string]string{
						mysqlServerLabelKey: mysqlServerLabelValue,
						mySQLOperatorLabel:  mySQLOperatorLabelValue,
					},
				},
			},
			reason:   "",
			patchLen: 1,
		},
		{
			name: "No patch, not mysql server",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-0",
					Namespace: "keycloak",
					Labels: map[string]string{
						mySQLOperatorLabel: mySQLOperatorLabelValue,
					},
				},
			},
			reason:   "No action required, Pod is not the mysql server",
			patchLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaulter := &MySQLPodWebhook{
				DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
				KubeClient:    fake.NewSimpleClientset(),
			}

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-0",
					Namespace: "keycloak",
					Labels: map[string]string{
						mysqlServerLabelKey: mysqlServerLabelValue,
						mySQLOperatorLabel:  mySQLOperatorLabelValue,
					},
				},
			}

			job, err := defaulter.KubeClient.CoreV1().Pods("keycloak").Create(context.TODO(), pod, metav1.CreateOptions{})
			assert.NoError(t, err, "Unexpected error creating pod")

			decoder := decoder()
			err = defaulter.InjectDecoder(decoder)
			assert.NoError(t, err, "Unexpected error injecting decoder")
			req := admission.Request{}
			req.Namespace = "keycloak"
			marshaledJob, err := json.Marshal(job)
			assert.NoError(t, err, "Unexpected error marshaling pod")
			req.Object = runtime.RawExtension{Raw: marshaledJob}
			res := defaulter.Handle(context.TODO(), req)
			assert.Nil(t, tt.patchLen, len(res.Patches))
			assert.Equal(t, metav1.StatusReason(tt.reason), res.Result.Reason)
			assert.True(t, res.Allowed)
		})
	}
}
