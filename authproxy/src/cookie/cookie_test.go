// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cookie

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStateCookie tests the state cookie encryption and decryption functionality
func TestStateCookie(t *testing.T) {
	// create a temporary file with a generated cookie encryption key
	filename, err := writeEncryptionKeyFile()
	assert.NoError(t, err)
	defer os.Remove(filename)
	prevEncryptionKeyFile := GetEncryptionKeyFile()
	defer SetEncryptionKeyFile(prevEncryptionKeyFile)
	SetEncryptionKeyFile(filename)

	// GIVEN a VZState struct
	// WHEN the struct is encrypted and stored in a cookie and then read back and decrypted into a new struct
	// THEN the two structs have exactly the same data

	// populate a VZState, encrypt it, and store it in a cookie
	vzState := &VZState{
		State:        "test-state",
		Nonce:        "test-nonce",
		CodeVerifier: "test-verifier",
		RedirectURI:  "https://example.com/redirect",
	}

	rw := httptest.NewRecorder()
	SetStateCookie(rw, vzState)

	req := httptest.NewRequest("", "https://example.com", nil)
	for _, c := range rw.Result().Cookies() {
		req.AddCookie(c)
	}

	// read the cookie back (decrypted) and validate that it matches the input state
	newVZState, err := GetStateCookie(req)
	assert.NoError(t, err)
	assert.Equal(t, vzState, newVZState)
}

// writeEncryptionKeyFile creates a temporary file and writes an encryption key. The function returns the file name.
func writeEncryptionKeyFile() (string, error) {
	f, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}
	f.Write([]byte("abcdefghijklmnopqrstuvwxyz1234567890"))
	f.Close()
	return f.Name(), nil
}
