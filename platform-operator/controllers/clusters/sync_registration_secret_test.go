// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"reflect"
	"testing"
)

// Test_combineBundles - test function that combines CA bundles (two base64 encoded strings)
func Test_combineBundles(t *testing.T) {
	type args struct {
		bundle1 []byte
		bundle2 []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Combine two bundles",
			args: args{
				bundle1: []byte("VGhlIGZpcnN0IENBIGJ1bmRsZQo="),
				bundle2: []byte("VGhlIHNlY29uZCBDQSBidW5kbGUK"),
			},
			want:    []byte("VGhlIGZpcnN0IENBIGJ1bmRsZQpUaGUgc2Vjb25kIENBIGJ1bmRsZQo="),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := combineBundles(tt.args.bundle1, tt.args.bundle2)
			if (err != nil) != tt.wantErr {
				t.Errorf("combineBundles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("combineBundles() got = %v, want %v", got, tt.want)
			}
		})
	}
}
