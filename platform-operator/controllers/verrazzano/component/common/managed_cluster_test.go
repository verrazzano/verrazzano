// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	k8score "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	testURL                = "url"
	testManagedClusterName = "managed1"
	ComponentNamespace     = vzconst.VerrazzanoSystemNamespace
)

func createTestRegistrationSecret(kvs map[string]string) *k8score.Secret {
	s := &k8score.Secret{
		ObjectMeta: k8smeta.ObjectMeta{
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

// TestManagedClusterRegistrationSecret tests fetching of managed cluster registration secret if it exists
func TestManagedClusterRegistrationSecret(t *testing.T) {
	a := assert.New(t)
	cli := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects().Build()
	a.Nil(GetManagedClusterRegistrationSecret(cli))
	scheme := runtime.NewScheme()
	_ = k8score.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	_ = k8score.AddToScheme(scheme)
	registrationSecret := createTestRegistrationSecret(map[string]string{
		vzconst.OpensearchURLData: testURL,
		vzconst.ClusterNameData:   testManagedClusterName,
	})
	cli = fake.NewClientBuilder().WithScheme(scheme).WithObjects(registrationSecret).Build()
	a.NotNil(GetManagedClusterRegistrationSecret(cli))
}
