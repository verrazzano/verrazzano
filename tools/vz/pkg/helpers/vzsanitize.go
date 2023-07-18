// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
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
		fmt.Sprintf(`("host"|"hostname"):(.*)"%s"$`, hostnames),
	}

	excludeRegex := []string{
		fmt.Sprintf(`%s("|):`, hostnames),
		fmt.Sprintf(`apiVersion(.*)%s`, hostnames),
		fmt.Sprintf(`"%s/(.*)`, hostnames),
	}

	var foundHostnames []string
	splitNewlines := strings.Split(line, "\n")
	for i, l := range splitNewlines {
		if matchesRegexListItem(l, includeRegex) && !matchesRegexListItem(l, excludeRegex) {
			splitNewlines[i] = regexp.MustCompile(hostnames).ReplaceAllString(l, getSha256Hash(l))
			foundHostnames = append(foundHostnames, regexp.MustCompile(hostnames).FindString(l))
		}
	}

	fmt.Printf("%v\n", foundHostnames)
	// Now that the hostnames have been collected, go back and filter them out
	for i, l := range splitNewlines {
		for _, host := range foundHostnames {
			splitNewlines[i] = regexp.MustCompile(host).ReplaceAllString(l, getSha256Hash(l))
		}
	}

	return strings.Join(splitNewlines, "\n")
}

func matchesRegexListItem(line string, list []string) bool {
	for _, r := range list {
		if regexp.MustCompile(r).Match([]byte(line)) {
			return true
		}
	}
	return false
}
