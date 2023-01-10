// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restart

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"testing"
)

const (
	testBomFilePath = "../../testdata/test_bom.json"
)

func TestBomTool(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	b, bomErr := newBomTool()
	assert.NoError(t, bomErr)

	var tests = []struct {
		name         string
		subcomponent string
		imageName    string
		hasError     bool
	}{
		{
			"has proxyv2 image for istiod subcomponent",
			istioSubcomponent,
			proxyv2ImageName,
			false,
		},
		{
			"has fluentd image for verrazzano subcomponent",
			verrazzanoSubcomponent,
			fluentdImageName,
			false,
		},
		{
			"has wko exporter image for wko subcomponent",
			wkoSubcomponent,
			wkoExporterImageName,
			false,
		},
		{
			"fails for invalid subcomponent",
			"foobar",
			proxyv2ImageName,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image, err := b.getImage(tt.subcomponent, tt.imageName)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, image)
			}
		})
	}
}
