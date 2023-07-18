// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
)

var regexToReplacementList = []string{}

const ipv4Regex = "[[:digit:]]{1,3}\\.[[:digit:]]{1,3}\\.[[:digit:]]{1,3}\\.[[:digit:]]{1,3}"
const userData = "\"user_data\":\\s+\"[A-Za-z0-9=+]+\""
const sshAuthKeys = "ssh-rsa\\s+[A-Za-z0-9=+ \\-\\/@]+"
const ocid = "ocid1\\.[[:lower:]]+\\.[[:alnum:]]+\\.[[:alnum:]]*\\.[[:alnum:]]+"
const hostnames = "([[:alnum:]][a-zA-Z0-9\\-]*)(\\.[a-zA-Z0-9\\-]+)+"

// InitRegexToReplacementMap Initialize the regex string to replacement string map
// Append to this map for any future additions
func InitRegexToReplacementMap() {
	regexToReplacementList = append(regexToReplacementList, ipv4Regex)
	regexToReplacementList = append(regexToReplacementList, userData)
	regexToReplacementList = append(regexToReplacementList, sshAuthKeys)
	regexToReplacementList = append(regexToReplacementList, ocid)
}

// SanitizeString sanitizes each line in a given file,
// Sanitizes based on the regex map initialized above, which is currently filtering for IPv4 addresses and hostnames
func SanitizeString(l string) string {
	if len(regexToReplacementList) == 0 {
		InitRegexToReplacementMap()
	}
	for _, eachRegex := range regexToReplacementList {
		l = regexp.MustCompile(eachRegex).ReplaceAllString(l, getSha256Hash(l))
	}

	return filterHostname(l)
}

// getSha256Hash generates the one way hash for the input string
func getSha256Hash(line string) string {
	data := []byte(line)
	hashedVal := sha256.Sum256(data)
	hexString := hex.EncodeToString(hashedVal[:])
	return hexString
}

func filterHostname(line string) string {
	includeRegex := []string{
		fmt.Sprintf(`("host":|"hostname":)(.*)"%s"`, hostnames),
		fmt.Sprintf(`\S+"%s"$`, hostnames),
	}

	excludeRegex := []string{
		fmt.Sprintf(`%s(.*):`, hostnames),
		fmt.Sprintf(`apiVersion(.*)%s`, hostnames),
		fmt.Sprintf(`f:(.*)%s`, hostnames),
		fmt.Sprintf(`"%s/(.*)`, hostnames),
	}

	if matchesRegexListItem(line, includeRegex) && !matchesRegexListItem(line, excludeRegex) {
		return regexp.MustCompile(hostnames).ReplaceAllString(line, getSha256Hash(line))
	}
	return line
}

func matchesRegexListItem(line string, list []string) bool {
	for _, r := range list {
		if regexp.MustCompile(r).Match([]byte(line)) {
			return true
		}
	}
	return false
}
