// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"testing"
)

func TestCompareInstallArgs(t *testing.T) {
	const exception = `test-execption`
	tests := []struct {
		name    string
		old     []vzapi.InstallArgs
		new     []vzapi.InstallArgs
		wantErr bool
	}{
		{
			name:    "nil InstallArgs",
			old:     nil,
			new:     nil,
			wantErr: false,
		},
		{
			name:    "empty InstallArgs",
			old:     []vzapi.InstallArgs{},
			new:     []vzapi.InstallArgs{},
			wantErr: false,
		},
		{
			name: "Different order InstallArgs",
			old: []vzapi.InstallArgs{
				{
					Name:  "a",
					Value: "1",
				},
				{
					Name:  "b",
					Value: "2",
				},
			},
			new: []vzapi.InstallArgs{
				{
					Name:  "b",
					Value: "2",
				},
				{
					Name:  "a",
					Value: "1",
				},
			},
			wantErr: false,
		},
		{
			name: "Different InstallArgs",
			old: []vzapi.InstallArgs{
				{
					Name:  "a",
					Value: "1",
				},
				{
					Name:  "b",
					Value: "2",
				},
			},
			new: []vzapi.InstallArgs{
				{
					Name:  "a",
					Value: "1",
				},
				{
					Name:  "b",
					Value: "3",
				},
			},
			wantErr: true,
		},
		{
			name: "Different exception InstallArgs",
			old: []vzapi.InstallArgs{
				{
					Name:  "a",
					Value: "1",
				},
				{
					Name:  exception,
					Value: "e1",
				},
			},
			new: []vzapi.InstallArgs{
				{
					Name:  "a",
					Value: "1",
				},
				{
					Name:  exception,
					Value: "e2",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CompareInstallArgs(tt.old, tt.new, []string{exception}); (err != nil) != tt.wantErr {
				t.Errorf("validateInstallArgsUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
