// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"testing"
)

func TestIsUpgradePending(t *testing.T) {
	defaultTestBomFile := "./testdata/test_bom.json"

	tests := []struct {
		name        string
		actualCR    *vzapi.Verrazzano
		testBOMPath string
		want        bool
		wantErr     assert.ErrorAssertionFunc
	}{
		{
			name:    "VerrazzanoCRIsNil",
			wantErr: assert.Error,
		},
		{
			name: "BothEmpty",
			actualCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{},
			},
		},
		{
			name: "UpdateInstall",
			actualCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{},
				Status: vzapi.VerrazzanoStatus{
					Version: "2.0.2",
				},
			},
		},
		{
			name: "UpgradeIsPending",
			actualCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Version: "2.0.1",
				},
				Status: vzapi.VerrazzanoStatus{
					Version: "2.0.1",
				},
			},
			want: true,
		},
		{
			name: "UpgradeInProgress",
			actualCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Version: "2.0.2",
				},
				Status: vzapi.VerrazzanoStatus{
					Version: "2.0.1",
				},
			},
		},
		{
			name: "UpgradeComplete",
			actualCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Version: "2.0.2",
				},
				Status: vzapi.VerrazzanoStatus{
					Version: "2.0.2",
				},
			},
		},
		{
			name: "BOM Error",
			actualCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{},
				Status: vzapi.VerrazzanoStatus{
					Version: "2.0.2",
				},
			},
			testBOMPath: "badpath",
			wantErr:     assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the test BOM path
			testBOM := tt.testBOMPath
			if len(testBOM) == 0 {
				testBOM = defaultTestBomFile
			}
			config.SetDefaultBomFilePath(testBOM)
			defer func() { config.SetDefaultBomFilePath("") }()

			// Set up the err check assertion
			wantErr := tt.wantErr
			if wantErr == nil {
				wantErr = assert.NoError
			}

			r := Reconciler{}
			got, err := r.isUpgradePending(tt.actualCR)
			if !wantErr(t, err, "Did not get expected error result") {
				return
			}
			assert.Equal(t, got, tt.want)
		})
	}
}
