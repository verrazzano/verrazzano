// Copyright (c) 2022, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"sync"
)

type regexPlan struct {
	preprocess  func(string) string
	regex       string
	postprocess func(string) string
}

var regexToReplacementList = []regexPlan{}
var KnownHostNames = make(map[string]bool)
var knownHostNamesMutex = &sync.Mutex{}

var ipv4Regex = regexPlan{regex: "[[:digit:]]{1,3}\\.[[:digit:]]{1,3}\\.[[:digit:]]{1,3}\\.[[:digit:]]{1,3}"}
var userData = regexPlan{regex: "\"user_data\":\\s+\"[A-Za-z0-9=+]+\""}
var sshAuthKeys = regexPlan{regex: "ssh-rsa\\s+[A-Za-z0-9=+ \\-\\/@]+"}
var ocid = regexPlan{regex: "ocid1\\.[[:lower:]]+\\.[[:alnum:]]+\\.[[:alnum:]]*\\.[[:alnum:]]+"}
var opcid = regexPlan{
	preprocess: func(s string) string {
		return strings.Trim(strings.TrimPrefix(s, "Opc request id:"), " ")
	},
	regex: "(?:Opc request id:) *[A-Z,a-z,/,0-9]+",
	postprocess: func(s string) string {
		return "Opc request id: " + s
	},
}

// InitRegexToReplacementMap Initialize the regex string to replacement string map
// Append to this map for any future additions
func InitRegexToReplacementMap() {
	regexToReplacementList = append(regexToReplacementList, ipv4Regex)
	regexToReplacementList = append(regexToReplacementList, userData)
	regexToReplacementList = append(regexToReplacementList, sshAuthKeys)
	regexToReplacementList = append(regexToReplacementList, ocid)
	regexToReplacementList = append(regexToReplacementList, opcid)
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
		l = regexp.MustCompile(eachRegex.regex).ReplaceAllStringFunc(l, eachRegex.compilePlan())
	}
	return l
}

func (rp regexPlan) compilePlan() func(string) string {
	return func(s string) string {
		var i string
		if rp.preprocess != nil {
			i = rp.preprocess(s)
		}
		i = redact(i)
		if rp.postprocess != nil {
			return rp.postprocess(i)
		}
		return i
	}
}

// redact outputs a string, representing a piece of redacted text
func redact(s string) string {
	return "REDACTED-" + getSha256Hash(s)
}

// getSha256Hash generates the one way hash for the input string
func getSha256Hash(line string) string {
	data := []byte(line)
	hashedVal := sha256.Sum256(data)
	hexString := hex.EncodeToString(hashedVal[:])
	return hexString
}
