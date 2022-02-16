// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appoper

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testBomFilePath = "../../testdata/test_bom.json"

// TestAppendAppOperatorOverrides tests the Keycloak override for the theme images
// GIVEN an env override for the app operator image
//  WHEN I call AppendApplicationOperatorOverrides
//  THEN the "image" Key is set with the image override.
func TestAppendAppOperatorOverrides(t *testing.T) {
	assert := assert.New(t)

	config.SetDefaultBomFilePath(testBomFilePath)

	const expectedFluentdImage = "ghcr.io/verrazzano/fluentd-kubernetes-daemonset:v1.12.3-20210517195222-f345ec2"
	const expectedIstioProxyImage = "ghcr.io/verrazzano/proxyv2:1.7.3"

	kvs, err := AppendApplicationOperatorOverrides(nil, "", "", "", nil)
	assert.NoError(err, "AppendApplicationOperatorOverrides returned an error ")
	assert.Len(kvs, 2, "AppendApplicationOperatorOverrides returned an unexpected number of Key:Value pairs")
	assert.Equalf("fluentdImage", kvs[0].Key, "Did not get expected fluentdImage Key")
	assert.Equalf(expectedFluentdImage, kvs[0].Value, "Did not get expected fluentdImage Value")
	assert.Equalf("istioProxyImage", kvs[1].Key, "Did not get expected istioProxyImage Key")
	assert.Equalf(expectedIstioProxyImage, kvs[1].Value, "Did not get expected istioProxyImage Value")

	customImage := "myreg.io/myrepo/v8o/verrazzano-application-operator-dev:local-20210707002801-b7449154"
	os.Setenv(constants.VerrazzanoAppOperatorImageEnvVar, customImage)
	defer os.Unsetenv(constants.RegistryOverrideEnvVar)

	kvs, err = AppendApplicationOperatorOverrides(nil, "", "", "", nil)
	assert.NoError(err, "AppendApplicationOperatorOverrides returned an error ")
	assert.Len(kvs, 3, "AppendApplicationOperatorOverrides returned wrong number of Key:Value pairs")
	assert.Equalf("image", kvs[0].Key, "Did not get expected image Key")
	assert.Equalf(customImage, kvs[0].Value, "Did not get expected image Value")
	assert.Equalf("fluentdImage", kvs[1].Key, "Did not get expected fluentdImage Key")
	assert.Equalf(expectedFluentdImage, kvs[1].Value, "Did not get expected fluentdImage Value")
	assert.Equalf("istioProxyImage", kvs[2].Key, "Did not get expected istioProxyImage Key")
	assert.Equalf(expectedIstioProxyImage, kvs[2].Value, "Did not get expected istioProxyImage Value")
}

//  TestIsApplyCRDYamlValid tests the ApplyCRDYaml function
//  GIVEN a call to ApplyCRDYaml
//  WHEN the yaml is valid
//  THEN no error is returned
func TestIsApplyCRDYamlValid(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	config.TestHelmConfigDir = "../../../../helm_config"
	assert.Nil(t, ApplyCRDYaml(nil, fakeClient, "", "", ""))
}

//  TestIsApplyCRDYamlInvalidPath tests the ApplyCRDYaml function
//  GIVEN a call to ApplyCRDYaml
//  WHEN the path is invalid
//  THEN an appropriate error is returned
func TestIsApplyCRDYamlInvalidPath(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	config.TestHelmConfigDir = "./testdata"
	assert.Error(t, ApplyCRDYaml(nil, fakeClient, "", "", ""))
}

//  TestIsApplyCRDYamlInvalidChart tests the ApplyCRDYaml function
//  GIVEN a call to ApplyCRDYaml
//  WHEN the yaml is invalid
//  THEN an appropriate error is returned
func TestIsApplyCRDYamlInvalidChart(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	config.TestHelmConfigDir = "invalidPath"
	assert.Error(t, ApplyCRDYaml(nil, fakeClient, "", "", ""))
}
