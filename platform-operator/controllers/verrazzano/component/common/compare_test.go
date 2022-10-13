// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"testing"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestCompareInstallArgs(t *testing.T) {
	const exception = `test-exception`
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
			name:    "nil to empty InstallArgs",
			old:     nil,
			new:     []vzapi.InstallArgs{},
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
			if err := CompareInstallArgs(tt.old, tt.new, exception); (err != nil) != tt.wantErr {
				t.Errorf("validateInstallArgsUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestComparePorts(t *testing.T) {
	tests := []struct {
		name    string
		old     []corev1.ServicePort
		new     []corev1.ServicePort
		wantErr bool
	}{
		{
			name:    "nil Ports",
			old:     nil,
			new:     nil,
			wantErr: false,
		},
		{
			name:    "empty to nil Ports",
			old:     []corev1.ServicePort{},
			new:     nil,
			wantErr: false,
		},
		{
			name:    "empty Ports",
			old:     []corev1.ServicePort{},
			new:     []corev1.ServicePort{},
			wantErr: false,
		},
		{
			name: "Different order Ports",
			old: []corev1.ServicePort{
				{
					Name: "a",
					Port: 1,
				},
				{
					Name: "b",
					Port: 2,
				},
			},
			new: []corev1.ServicePort{
				{
					Name: "b",
					Port: 2,
				},
				{
					Name: "a",
					Port: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "Different Ports",
			old: []corev1.ServicePort{
				{
					Name: "a",
					Port: 1,
				},
				{
					Name: "b",
					Port: 2,
				},
			},
			new: []corev1.ServicePort{
				{
					Name: "a",
					Port: 1,
				},
				{
					Name: "c",
					Port: 2,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ComparePorts(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("validateInstallArgsUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
