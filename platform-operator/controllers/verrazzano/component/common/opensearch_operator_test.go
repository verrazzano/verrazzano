// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testConfigData       = "config:\n      dynamic:\n        kibana:\n          multitenancy_enabled: false\n        http:\n          anonymous_auth_enabled: false\n          xff:\n            enabled: true\n            internalProxies: '.*'\n            remoteIpHeader: 'x-forwarded-for'\n        authc:\n          vz_proxy_auth_domain:\n            description: \"Authenticate via Verrazzano proxy\"\n            http_enabled: true\n            transport_enabled: true\n            order: 0\n            http_authenticator:\n              type: proxy\n              challenge: false\n              config:\n                user_header: \"X-WEBAUTH-USER\"\n                roles_header: \"x-proxy-roles\"\n            authentication_backend:\n              type: noop\n          vz_basic_internal_auth_domain:\n            description: \"Authenticate via HTTP Basic against internal users database\"\n            http_enabled: true\n            transport_enabled: true\n            order: 1\n            http_authenticator:\n              type: basic\n              challenge: false\n            authentication_backend:\n              type: intern\n          vz_clientcert_auth_domain:\n             description: \"Authenticate via SSL client certificates\"\n             http_enabled: true\n             transport_enabled: true\n             order: 2\n             http_authenticator:\n               type: clientcert\n               config:\n                 enforce_hostname_verification: false\n                 username_attribute: cn\n               challenge: false\n             authentication_backend:\n                 type: noop"
	testUsersData        = "admin:\n    hash: \n    reserved: true\n    backend_roles:\n    - \"admin\"\n    description: \"Admin user\""
	profilesRelativePath = "../../../../manifests/profiles"
	clusterLabel         = "opster.io/opensearch-cluster"
	esData               = "es-data"
)

// TestMergeSecurityConfigs tests the MergeSecretData function
// GIVEN a call to MergeSecretData
// WHEN OpenSearchOperator is pre-installed
// THEN no error is returned
func TestMergeSecurityConfigs(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	fakeCtx := spi.NewFakeContext(mock, nil, nil, false)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: securityNamespace, Name: securitySecretName}, gomock.Not(gomock.Nil()), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
		secret.Name = securitySecretName
		secret.Namespace = securityNamespace
		secret.Data = map[string][]byte{configYaml: []byte(testConfigData), usersYaml: []byte(testUsersData)}
		return nil
	})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: securityNamespace, Name: hashSecName}, gomock.Not(gomock.Nil()), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
		secret.Name = hashSecName
		secret.Namespace = securityNamespace
		secret.Data = map[string][]byte{"hash": []byte("abcdef")}
		return nil
	})
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	config.TestThirdPartyManifestDir = "../../../../thirdparty/manifests"
	err := MergeSecretData(fakeCtx, config.GetThirdPartyManifestsDir())
	asserts.NoError(err)
}

// TestMergeSecurityConfigsGetConfigError tests the MergeSecretData function
// GIVEN a call to MergeSecretData
// WHEN get security config secret fails
// THEN error is returned
func TestMergeSecurityConfigsGetConfigError(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	fakeCtx := spi.NewFakeContext(mock, nil, nil, false)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: securityNamespace, Name: securitySecretName}, gomock.Not(gomock.Nil()), gomock.Any()).Return(fmt.Errorf("test-error"))
	err := MergeSecretData(fakeCtx, config.GetThirdPartyManifestsDir())
	asserts.Error(err)
}

// TestMergeSecurityConfigsGetAdminError tests the MergeSecretData function
// GIVEN a call to MergeSecretData
// WHEN get admin secret fails
// THEN error is returned
func TestMergeSecurityConfigsGetAdminError(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	fakeCtx := spi.NewFakeContext(mock, nil, nil, false)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: securityNamespace, Name: securitySecretName}, gomock.Not(gomock.Nil()), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
		secret.Name = securitySecretName
		secret.Namespace = securityNamespace
		secret.Data = map[string][]byte{configYaml: []byte(testConfigData), usersYaml: []byte(testUsersData)}
		return nil
	})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: securityNamespace, Name: hashSecName}, gomock.Not(gomock.Nil()), gomock.Any()).Return(fmt.Errorf("test-error"))
	err := MergeSecretData(fakeCtx, config.GetThirdPartyManifestsDir())
	asserts.Error(err)
}

// TestMergeSecurityConfigsUpdateError tests the MergeSecretData function
// GIVEN a call to MergeSecretData
// WHEN secret update fails
// THEN error is returned
func TestMergeSecurityConfigsUpdateError(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	fakeCtx := spi.NewFakeContext(mock, nil, nil, false)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: securityNamespace, Name: securitySecretName}, gomock.Not(gomock.Nil()), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
		secret.Name = securitySecretName
		secret.Namespace = securityNamespace
		secret.Data = map[string][]byte{configYaml: []byte(testConfigData), usersYaml: []byte(testUsersData)}
		return nil
	})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: securityNamespace, Name: hashSecName}, gomock.Not(gomock.Nil()), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
		secret.Name = hashSecName
		secret.Namespace = securityNamespace
		secret.Data = map[string][]byte{"hash": []byte("test")}
		return nil
	})
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(fmt.Errorf("test-error"))

	err := MergeSecretData(fakeCtx, config.GetThirdPartyManifestsDir())
	asserts.Error(err)
}

// TestMergeSecurityConfigsHashError tests the MergeSecretData function
// GIVEN a call to MergeSecretData
// WHEN hash key is not found
// THEN error is returned
func TestMergeSecurityConfigsHashError(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	fakeCtx := spi.NewFakeContext(mock, nil, nil, false)
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: securityNamespace, Name: securitySecretName}, gomock.Not(gomock.Nil()), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
		secret.Name = securitySecretName
		secret.Namespace = securityNamespace
		secret.Data = map[string][]byte{configYaml: []byte(testConfigData), usersYaml: []byte(testUsersData)}
		return nil
	})
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: securityNamespace, Name: hashSecName}, gomock.Not(gomock.Nil()), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
		secret.Name = hashSecName
		secret.Namespace = securityNamespace
		secret.Data = map[string][]byte{"hash1": []byte("test")}
		return nil
	})

	err := MergeSecretData(fakeCtx, config.GetThirdPartyManifestsDir())
	asserts.Error(err)
}

// TestIsUpgrade tests the IsUpgrade function
// GIVEN a call to IsUpgrade
// WHEN there are older PVs is called
// THEN expected boolean is returned
func TestIsUpgrade(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()

	fakeCtx := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false, profilesRelativePath)
	assert.False(t, IsUpgrade(fakeCtx))

	fakeClient = fake.NewClientBuilder().WithScheme(testScheme).WithLists(
		&corev1.PersistentVolumeList{Items: []corev1.PersistentVolume{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-1",
					Labels: map[string]string{
						opensearchNodeLabel: esData,
						clusterLabel:        clusterName,
					},
				},
				Spec: corev1.PersistentVolumeSpec{
					ClaimRef: &corev1.ObjectReference{
						Namespace: constants.VerrazzanoSystemNamespace,
						Name:      "vmi-system-es-data-1",
					},
					PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
				},
			},
		}}).Build()
	fakeCtx = spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false, profilesRelativePath)
	assert.True(t, IsUpgrade(fakeCtx))
}

// TestIsSingleMasterNodeCluster tests the IsSingleMasterNodeCluster function
// GIVEN a VZ CR
// WHEN IsSingleMasterNodeCluster is called
// THEN expected boolean is returned
func TestIsSingleMasterNodeCluster(t *testing.T) {
	fakeCtx := spi.NewFakeContext(nil, &v1alpha1.Verrazzano{}, nil, false, profilesRelativePath)
	assert.False(t, IsSingleMasterNodeCluster(fakeCtx))
	fakeCtx = spi.NewFakeContext(nil, &v1alpha1.Verrazzano{Spec: v1alpha1.VerrazzanoSpec{Profile: "dev"}}, nil, false, profilesRelativePath)
	assert.True(t, IsSingleMasterNodeCluster(fakeCtx))
}

func TestGetImagesOverrides(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	kvs, err := GetVMOImagesOverrides()
	assert.NoError(t, err)

	// Assert that OS, OSD and init images are found
	osImageFound := false
	osdImageFound := false
	initImageFound := false

	for _, kv := range kvs {
		if kv.Key == "monitoringOperator.osImage" {
			t.Log(kv.Value)
			osImageFound = true
		} else if kv.Key == "monitoringOperator.osdImage" {
			t.Log(kv.Value)
			osdImageFound = true
		} else if kv.Key == "monitoringOperator.osInitImage" {
			t.Log(kv.Value)
			initImageFound = true
		}
	}
	assert.True(t, osImageFound)
	assert.True(t, osdImageFound)
	assert.True(t, initImageFound)
}
