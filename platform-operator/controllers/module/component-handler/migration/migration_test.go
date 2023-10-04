// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package migration

import (
	"github.com/stretchr/testify/assert"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"testing"
)

// TestIsUpgradeRequired tests that isUpgradeRequired method tells us when an upgrade is required before we can apply
// any module updates
// GIVEN a call to isUpgradeRequired
// WHEN the Verrazzano spec version is out of sync with the BOM version
// THEN true is returned
func TestIsUpgradeRequired(t *testing.T) {
	defaultTestBomFile := "./testdata/test_bom.json"

	tests := []struct {
		name        string
		actualCR    *vzv1alpha1.Verrazzano
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
			actualCR: &vzv1alpha1.Verrazzano{
				Spec: vzv1alpha1.VerrazzanoSpec{},
			},
		},
		{
			name: "UpdateInstall",
			actualCR: &vzv1alpha1.Verrazzano{
				Spec: vzv1alpha1.VerrazzanoSpec{},
				Status: vzv1alpha1.VerrazzanoStatus{
					Version: "2.0.2",
				},
			},
		},
		{
			name: "UpgradeIsPending",
			actualCR: &vzv1alpha1.Verrazzano{
				Spec: vzv1alpha1.VerrazzanoSpec{
					Version: "2.0.1",
				},
				Status: vzv1alpha1.VerrazzanoStatus{
					Version: "2.0.1",
				},
			},
			want: true,
		},
		{
			name: "UpgradeToBranchBuildPending",
			actualCR: &vzv1alpha1.Verrazzano{
				Spec: vzv1alpha1.VerrazzanoSpec{
					Version: "2.0.1",
				},
				Status: vzv1alpha1.VerrazzanoStatus{
					Version: "2.0.1",
				},
			},
			testBOMPath: "./testdata/test_bom_buildver.json",
			want:        true,
		},
		{
			name: "UpgradeToBranchBuildSameMMP",
			actualCR: &vzv1alpha1.Verrazzano{
				Spec: vzv1alpha1.VerrazzanoSpec{
					Version: "2.0.2",
				},
				Status: vzv1alpha1.VerrazzanoStatus{
					Version: "2.0.2",
				},
			},
			testBOMPath: "./testdata/test_bom_buildver.json",
			want:        true,
		},
		{
			name: "UpgradeToNewBranchBuild",
			actualCR: &vzv1alpha1.Verrazzano{
				Spec: vzv1alpha1.VerrazzanoSpec{
					Version: "2.0.2-1+bbadkkad",
				},
				Status: vzv1alpha1.VerrazzanoStatus{
					Version: "2.0.2-1+bbadkkad",
				},
			},
			testBOMPath: "./testdata/test_bom_buildver.json",
			want:        true,
		},
		{
			name: "UpgradeToNewBranchBuild2",
			actualCR: &vzv1alpha1.Verrazzano{
				Spec: vzv1alpha1.VerrazzanoSpec{
					Version: "2.0.2-2+caafdaad",
				},
				Status: vzv1alpha1.VerrazzanoStatus{
					Version: "2.0.2-2+caafdaad",
				},
			},
			testBOMPath: "./testdata/test_bom_buildver.json",
			want:        true,
		},
		{
			name: "UpgradeToNewBranchBuildFromInstall",
			actualCR: &vzv1alpha1.Verrazzano{
				Status: vzv1alpha1.VerrazzanoStatus{
					Version: "2.0.2-1+bbadkkad",
				},
			},
			testBOMPath: "./testdata/test_bom_buildver.json",
			want:        true,
		},
		{
			name: "UpgradeToNewBranchBuildFromInstall2",
			actualCR: &vzv1alpha1.Verrazzano{
				Status: vzv1alpha1.VerrazzanoStatus{
					Version: "2.0.2-2+caafdaad",
				},
			},
			testBOMPath: "./testdata/test_bom_buildver.json",
			want:        true,
		},
		{
			name: "UpgradeToBranchBuildCompleted",
			actualCR: &vzv1alpha1.Verrazzano{
				Spec: vzv1alpha1.VerrazzanoSpec{
					Version: "2.0.2-1+asdfdf",
				},
				Status: vzv1alpha1.VerrazzanoStatus{
					Version: "2.0.2-1+asdfdf",
				},
			},
			testBOMPath: "./testdata/test_bom_buildver.json",
		},
		{
			name: "UpgradeInProgress",
			actualCR: &vzv1alpha1.Verrazzano{
				Spec: vzv1alpha1.VerrazzanoSpec{
					Version: "2.0.2",
				},
				Status: vzv1alpha1.VerrazzanoStatus{
					Version: "2.0.1",
				},
			},
		},
		{
			name: "UpgradeComplete",
			actualCR: &vzv1alpha1.Verrazzano{
				Spec: vzv1alpha1.VerrazzanoSpec{
					Version: "2.0.2",
				},
				Status: vzv1alpha1.VerrazzanoStatus{
					Version: "2.0.2",
				},
			},
		},
		{
			name: "BOM Error",
			actualCR: &vzv1alpha1.Verrazzano{
				Spec: vzv1alpha1.VerrazzanoSpec{},
				Status: vzv1alpha1.VerrazzanoStatus{
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

			got, err := isUpgradeRequired(tt.actualCR)
			if !wantErr(t, err, "Did not get expected error result") {
				return
			}
			assert.Equal(t, got, tt.want)
		})
	}
}
