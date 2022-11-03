// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
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

const (
	modeAlphaNum   = 0
	modeAlphaLower = 1
)

// GeneratePassword will generate a password of length
func GeneratePassword(length int) (string, error) {
	return GeneratePasswordUsingMode(length, modeAlphaNum)
}

// GenerateRandomAlphaLower will generate a lower-case alpha string of length
// Should be used only for generating semi-unique, non-cryptographic, non-secret strings -- NOT passwords!
func GenerateRandomAlphaLower(length int) (string, error) {
	return GeneratePasswordUsingMode(length, modeAlphaLower)
}

// GeneratePasswordUsingMode will generate a password of length with mode
func GeneratePasswordUsingMode(length int, mode int) (string, error) {
	if length < 1 {
		return "", fmt.Errorf("cannot create password of length %d", length)
	}
	// Enlarge buffer so plenty of room is left when special characters are stripped out
	b := make([]byte, length*4)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	pw := b64.StdEncoding.EncodeToString(b)
	if mode == modeAlphaNum {
		pw, err = makeAlphaNumeric(pw)
	} else {
		pw, err = makeAlphaLower(pw)
	}
	if err != nil {
		return "", err
	}
	return pw[:length], nil
}

// makeAlphaNumeric removes all special characters from a password string
func makeAlphaNumeric(input string) (string, error) {
	// Make a Regex to say we only want letters and numbers
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return "", err
	}
	return reg.ReplaceAllString(input, ""), nil
}

// makeAlphaLower removes all special characters and numbers from a string, and lowercases the result
func makeAlphaLower(input string) (string, error) {
	// Make a Regex to say we only want letters
	reg, err := regexp.Compile("[^a-zA-Z]+")
	if err != nil {
		return "", err
	}
	return strings.ToLower(reg.ReplaceAllString(input, "")), nil
}

// MaskFunction creates a function intended to mask passwords which are substrings in other strings
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
