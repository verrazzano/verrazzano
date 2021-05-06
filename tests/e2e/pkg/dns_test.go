// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	asserts "github.com/stretchr/testify/assert"
	"testing"
)

// TestDNSWildcards tests if a string uses a DNS wildcard
// Given a set of strings
// WHEN a string has a valid DNS wildcard syntax
// THEN return true
func TestValidWildcard(t *testing.T) {
	assert := asserts.New(t)
	tests := []struct {
		name   string
		data   string
		domain string
	}{
		{
			name:   "test1",
			data:   "nip.io",
			domain: "nip.io",
		},
		{
			name:   "test2",
			data:   "a.nip.io",
			domain: "nip.io",
		},
		{
			name:   "test3",
			data:   "a.nip.io:33",
			domain: "nip.io",
		},
		{
			name:   "test4",
			data:   "a.nip.io:33",
			domain: "nip.io",
		},
		{
			name:   "test5",
			data:   "123.23.344.343.nip.io/foo:33",
			domain: "nip.io",
		},
		{
			name:   "test6",
			data:   "a.nip.io",
			domain: "nip.io",
		},
		{
			name:   "test7",
			data:   "a.sslip.io",
			domain: "sslip.io",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.True(HasWildcardDNS(test.data), test.name+" failed")
			assert.Equal(GetWildcardDNS(test.data), test.domain, test.name+" failed")
		})
	}
}

// TestNotWildcard tests if a string does not use a DNS wildcard
// Given a set of strings
// WHEN a string does not have a valid DNS wildcard syntax
// THEN return false
func TestNotWildcard(t *testing.T) {
	assert := asserts.New(t)
	tests := []struct {
		name string
		data string
	}{
		{
			name: "test1",
			data: "xipp.io",
		},
		{
			name: "test2",
			data: "xipio.a",
		},
		{
			name: "test5",
			data: "123.23.344.343.xip:33",
		},
		{
			name: "test6",
			data: "nip.b",
		},
		{
			name: "test7",
			data: "sslip.b",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.False(HasWildcardDNS(test.data), test.name+" failed")
		})
	}
}
