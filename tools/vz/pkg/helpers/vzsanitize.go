// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sync"
)

var regexToReplacementList = []string{}
var KnownHostNames = make(map[string]bool)
var knownHostNamesMutex = &sync.Mutex{}

const ipv4Regex = "[[:digit:]]{1,3}\\.[[:digit:]]{1,3}\\.[[:digit:]]{1,3}\\.[[:digit:]]{1,3}"
const userData = "\"user_data\":\\s+\"[A-Za-z0-9=+]+\""
const sshAuthKeys = "ssh-rsa\\s+[A-Za-z0-9=+ \\-\\/@]+"
const ocid = "ocid1\\.[[:lower:]]+\\.[[:alnum:]]+\\.[[:alnum:]]*\\.[[:alnum:]]+"

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
	knownHostNamesMutex.Lock()
	for knownHost := range KnownHostNames {
		wholeOccurrenceHostPattern := "\"" + knownHost + "\""
		l = regexp.MustCompile(wholeOccurrenceHostPattern).ReplaceAllString(l, "\""+getSha256Hash(knownHost)+"\"")
	}
	knownHostNamesMutex.Unlock()
	for _, eachRegex := range regexToReplacementList {
		l = regexp.MustCompile(eachRegex).ReplaceAllString(l, getSha256Hash(l))
	}
	return l
}

// getSha256Hash generates the one way hash for the input string
func getSha256Hash(line string) string {
	data := []byte(line)
	hashedVal := sha256.Sum256(data)
	hexString := hex.EncodeToString(hashedVal[:])
	return hexString
}
