// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package password

import (
	"crypto/rand"
	b64 "encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

const mask = "******"

//GeneratePassword will generate a password of length
func GeneratePassword(length int) (string, error) {
	if length < 1 {
		return "", fmt.Errorf("cannot create password of length %d", length)
	}
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	pw := b64.URLEncoding.EncodeToString(b)
	return pw[:length], nil
}

//MaskFunction creates a function intended to mask passwords which are substrings in other strings
// f := MaskFunction("pw=") creates a function that masks strings like so:
// f("pw=xyz") = "pw=******"
func MaskFunction(prefix string) func(string) string {
	re := regexp.MustCompile(fmt.Sprintf(`%s.*?(?:\s|$)`, prefix))
	return func(s string) string {
		replace := fmt.Sprintf("%s%s", prefix, mask)
		matches := re.FindAllString(s, -1)
		for _, match := range matches {
			ch := match[len(match)-1]
			var m string
			if ch == ' ' || ch == '\n' || ch == '\t' || ch == '\r' {
				m = match[:len(match)-1] // preserve last non-masked character if there is one
			} else {
				m = match
			}
			s = strings.ReplaceAll(s, m, replace)
		}
		return s
	}
}
