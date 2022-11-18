// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	testURL                = "url"
	testCredentials        = "test"
	testManagedClusterName = "managed1"
)

func createTestRegistrationSecret(kvs map[string]string) *corev1.Secret {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vzconst.MCRegistrationSecret,
			Namespace: ComponentNamespace,
		},
	}
	s.Data = map[string][]byte{}
	for k, v := range kvs {
		s.Data[k] = []byte(v)
	}
	return s
}

func TestAppendFluentdLogging(t *testing.T) {
	registrationSecret := createTestRegistrationSecret(map[string]string{
		vzconst.OpensearchURLData: testURL,
		vzconst.ClusterNameData:   testManagedClusterName,
	})
	var tests = []struct {
		name    string
		fluentd *vzapi.FluentdComponent
		client  clipkg.Client
	}{
		{
			"uses fluentd URL and credentials if no registration secret",
			&vzapi.FluentdComponent{
				ElasticsearchSecret: testCredentials,
				ElasticsearchURL:    testURL,
			},
			fake.NewClientBuilder().Build(),
		},
		{
			"uses registration secret for overrides if present",
			nil,
			fake.NewClientBuilder().WithObjects(registrationSecret).Build(),
		},
	}

	regString := func(key string) string {
		return string(registrationSecret.Data[key])
	}

	for _, tt := range tests {
		overrides := &fluentdComponentValues{}
		err := appendFluentdLogging(tt.client, tt.fluentd, overrides)
		assert.NoError(t, err)
		if tt.fluentd != nil {
			assert.Equal(t, tt.fluentd.ElasticsearchURL, overrides.Logging.OpenSearchURL)
			assert.Equal(t, tt.fluentd.ElasticsearchSecret, overrides.Logging.CredentialsSecret)
			assert.Equal(t, vzconst.MCLocalCluster, overrides.Logging.ClusterName)
		} else {
			assert.Equal(t, regString(vzconst.OpensearchURLData), overrides.Logging.OpenSearchURL)
			assert.Equal(t, vzconst.MCRegistrationSecret, overrides.Logging.CredentialsSecret)
			assert.Equal(t, regString(vzconst.ClusterNameData), overrides.Logging.ClusterName)
		}
	}
}
