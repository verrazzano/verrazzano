// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"context"
	_ "embed"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"testing"
)

func TestCreateVZMeta(t *testing.T) {
	var tests = []struct {
		name   string
		client corev1.CoreV1Interface
	}{
		{
			"create configmap if not exists",
			fake.NewSimpleClientset().CoreV1(),
		},
		{
			"update configmap if exists",
			fake.NewSimpleClientset(&v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configmapName,
					Namespace: vpoconst.VerrazzanoInstallNamespace,
				},
			}).CoreV1(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			k8sutil.GetCoreV1Func = func(_ ...vzlog.VerrazzanoLogger) (corev1.CoreV1Interface, error) {
				return tt.client, nil
			}
			defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()

			err := CreateVZMeta(ctx)
			assert.NoError(t, err)

			cm, err := tt.client.ConfigMaps(vpoconst.VerrazzanoInstallNamespace).Get(ctx, configmapName, metav1.GetOptions{})
			assert.NoError(t, err)
			assert.NotNil(t, cm.Data)
			assert.Equal(t, cm.Data[verrazzanoVersionsKey], verrazzanoVersions)
		})
	}
}
