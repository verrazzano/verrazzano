// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package password

import (
	"crypto/rand"
	b64 "encoding/base64"
)

// GeneratePassword will generate a password of length
func GeneratePassword(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	// The slice should now contain random bytes instead of only zeroes.
	pw := b64.StdEncoding.EncodeToString(b)
	return pw, nil
}
