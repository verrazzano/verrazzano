// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"math/big"
	"strings"
)

// Encodes byte stream to base64-url string
// Returns the encoded string
func encode(msg []byte) string {
	encoded := base64.StdEncoding.EncodeToString(msg)
	encoded = strings.Replace(encoded, "+", "-", -1)
	encoded = strings.Replace(encoded, "/", "_", -1)
	encoded = strings.Replace(encoded, "=", "", -1)
	return encoded
}

// Generates a random code verifier and then produces a code challenge using it.
// Returns the produced code_verifier and code_challenge pair
func GenerateRandomCodePair() (string, string, error) {
	length := 32
	r, err := rand.Int(rand.Reader, big.NewInt(256))
	if err != nil {
		return "", "", err
	}
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = byte(r.Uint64())
	}
	codeVerifier := encode(b)
	h := sha256.New()
	h.Write([]byte(codeVerifier))
	codeChallenge := encode(h.Sum(nil))
	return codeVerifier, codeChallenge, nil
}

// Generates a random string which is used as the state
func GenerateRandomState() (string, error) {
	length := 14
	r, err := rand.Int(rand.Reader, big.NewInt(256))
	if err != nil {
		return "", err
	}
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = byte(r.Uint64())
	}
	return encode(b), nil
}
