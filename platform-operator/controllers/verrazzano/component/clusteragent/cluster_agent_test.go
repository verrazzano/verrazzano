// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusteragent

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// TestAppendAppOperatorOverrides tests the app operator image override
// GIVEN an env override for the app operator image
//
//	WHEN I call AppendClusterAgentOverrides
//	THEN the "image" Key is set with the image override.
func TestAppendClusterAgentOverrides(t *testing.T) {
	a := assert.New(t)

	kvs, err := AppendClusterAgentOverrides(nil, "", "", "", nil)
	a.NoError(err, "AppendClusterAgentOverrides returned an error ")
	a.Len(kvs, 0, "AppendClusterAgentOverrides returned an unexpected number of Key:Value pairs")

	customImage := "myreg.io/myrepo/v8o/verrazzano-application-operator-dev:local-20210707002801-b7449154"
	_ = os.Setenv(constants.VerrazzanoAppOperatorImageEnvVar, customImage)
	defer func() { _ = os.Unsetenv(constants.RegistryOverrideEnvVar) }()

	kvs, err = AppendClusterAgentOverrides(nil, "", "", "", nil)
	a.NoError(err, "AppendClusterAgentOverrides returned an error ")
	a.Len(kvs, 1, "AppendClusterAgentOverrides returned wrong number of Key:Value pairs")
	a.Equalf("image", kvs[0].Key, "Did not get expected image Key")
	a.Equalf(customImage, kvs[0].Value, "Did not get expected image Value")
}

// TestGetOverrides tests the cluster agent overrides
// GIVEN an override in the cluster agent component
//
//	WHEN I call GetOverrides
//	THEN the component overrides get populated
func TestGetOverrides(t *testing.T) {
	a := assert.New(t)

	vzEmpty := &v1alpha1.Verrazzano{}
	overInterface := GetOverrides(vzEmpty)
	overrides := overInterface.([]v1alpha1.Overrides)
	a.Len(overrides, 0, "Overrides returned the wrong amount of values")

	vzOverride := vzEmpty.DeepCopy()
	vzOverride.Spec.Components.ClusterAgent = &v1alpha1.ClusterAgentComponent{
		InstallOverrides: v1alpha1.InstallOverrides{
			ValueOverrides: []v1alpha1.Overrides{
				{
					Values: &apiextensionsv1.JSON{Raw: []byte(`{"test": "json"}`)},
				},
			},
		},
	}
	overInterface = GetOverrides(vzOverride)
	overrides = overInterface.([]v1alpha1.Overrides)
	a.Len(overrides, 1, "Overrides returned the wrong amount of values")
}
