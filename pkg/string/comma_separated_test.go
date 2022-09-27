// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package string

import "testing"

// TestCommaSeparatedContainsString tests the CommaSeparatedStringContains function
func TestCommaSeparatedContainsString(t *testing.T) {
	type args struct {
		commaSeparated string
		s              string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test empty string",
			args: args{
				commaSeparated: "",
				s:              "test",
			},
			want: false,
		},
		{
			name: "Test match without any commas",
			args: args{
				commaSeparated: "test",
				s:              "test",
			},
			want: true,
		},
		{
			name: "Test match with commas",
			args: args{
				commaSeparated: "test1,test2,test3",
				s:              "test2",
			},
			want: true,
		},
		{
			name: "Test no match with commas",
			args: args{
				commaSeparated: "test1,test2,test3",
				s:              "test",
			},
			want: false,
		},
	}

	// Process each test case
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CommaSeparatedStringContains(tt.args.commaSeparated, tt.args.s); got != tt.want {
				t.Errorf("CommaSeparatedStringContains() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCommaSeparatedAppendString tests the AppendToCommaSeparatedString function
func TestCommaSeparatedAppendString(t *testing.T) {
	type args struct {
		commaSeparated string
		s              string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test append to empty string",
			args: args{
				commaSeparated: "",
				s:              "test",
			},
			want: "test",
		},
		{
			name: "Test append to string without commas",
			args: args{
				commaSeparated: "test1",
				s:              "test2",
			},
			want: "test1,test2",
		},
		{
			name: "Test append to string with commas",
			args: args{
				commaSeparated: "test1,test2,test3",
				s:              "test4",
			},
			want: "test1,test2,test3,test4",
		},
	}

	// Process each test case
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AppendToCommaSeparatedString(tt.args.commaSeparated, tt.args.s); got != tt.want {
				t.Errorf("AppendToCommaSeparatedString() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCommaSeparatedRemoveString tests the RemoveFromCommaSeparatedString function
func TestCommaSeparatedRemoveString(t *testing.T) {
	type args struct {
		commaSeparated string
		s              string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test remove from empty string",
			args: args{
				commaSeparated: "",
				s:              "test",
			},
			want: "",
		},
		{
			name: "Test remove from string that does not contain the value",
			args: args{
				commaSeparated: "test1,test2",
				s:              "test",
			},
			want: "test1,test2",
		},
		{
			name: "Test remove from string that does contains the value but no commas",
			args: args{
				commaSeparated: "test1",
				s:              "test1",
			},
			want: "",
		},
		{
			name: "Test remove from string that does contains the value with commas",
			args: args{
				commaSeparated: "test1,test2,test3",
				s:              "test2",
			},
			want: "test1,test3",
		},
	}

	// Process each test case
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveFromCommaSeparatedString(tt.args.commaSeparated, tt.args.s); got != tt.want {
				t.Errorf("RemoveFromCommaSeparatedString() = %v, want %v", got, tt.want)
			}
		})
	}
}
