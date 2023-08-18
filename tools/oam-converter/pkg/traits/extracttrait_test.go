// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package traits

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	reader "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/testdata"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

// Test cases for ExtractTrait function
func TestExtractTrait(t *testing.T) {

	// Test data: sample appMaps
	appMaps := []map[string]interface{}{}
	input := types.ConversionInput{}

	appConf, err := reader.ReadFromYAMLTemplate("testdata/template/app_conf.yaml")
	if err != nil {
		return
	}
	appMaps = append(appMaps, appConf)

	// Call the function to test
	result, err := ExtractTrait(appMaps, input)

	// Assertions
	assert.NoError(t, err)

	expectedResult := []*types.ConversionComponents{
		{
			AppNamespace:  "test-namespace",
			AppName:       "test-appconf",
			ComponentName: "test-component",
			IngressTrait: &vzapi.IngressTrait{
				TypeMeta: metav1.TypeMeta{
					Kind:       "IngressTrait",
					APIVersion: "oam.verrazzano.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ingress-trait",
				},
				Spec: vzapi.IngressTraitSpec{
					Rules: []vzapi.IngressRule{
						{
							Destination: vzapi.IngressDestination{},
							Paths: []vzapi.IngressPath{
								{
									Path:     "/test-ingress-path",
									PathType: "Prefix",
								},
							},
						},
					},
				},
			},
		},
	}
	assert.True(t, assert.Equal(t, expectedResult, result))
}
