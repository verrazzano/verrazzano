package files

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
)

var regex_confluence = regexp.MustCompile("confluence\\.oraclecorp\\.com\\/confluence")
var regex_gitlab = regexp.MustCompile("gitlab-odx\\.oracledx\\.com")
var regex_verrazzano = regexp.MustCompile("verrazzano.*\\.oracledx\\.com")
var regex_oracle = regexp.MustCompile("[a-zA-Z0-9]*\\.us\\.oracle\\.com")
var regex_ip = regexp.MustCompile("[[:digit:]]{1,3}\\.[[:digit:]]{1,3}\\.[[:digit:]]{1,3}\\.[[:digit:]]{1,3}")
var regex_oraclecorp = regexp.MustCompile("[a-zA-Z0-9]*\\.oraclecorp\\.com")

func SanitizeDirectory(rootDirectory string) {
	fileMatches, _ := GetMatchingFiles(nil, rootDirectory, regexp.MustCompile(".*"))
	for _, eachFile := range fileMatches {
		SanitizeFile(eachFile)
	}
}

func SanitizeFile(path string) error {
	input, err := os.Open(path)
	check(err)
	defer input.Close()

	outFile, err := os.Create(path + "_tmpfoo")
	check(err)
	defer outFile.Close()

	br := bufio.NewReader(input)
	for {
		l, _, err := br.ReadLine()

		if err == io.EOF {
			break
		}
		check(err)

		sanitizedLine := sanitizeEachLine(string(l))
		outFile.WriteString(sanitizedLine + "\n")
	}

	return nil
}

func sanitizeEachLine(l string) string {
	fmt.Println("current line is, ", l)
	l = regex_confluence.ReplaceAllString(l, "REDACTED-HOSTNAME")
	l = regex_gitlab.ReplaceAllString(l, "REDACTED-HOSTNAME")
	l = regex_verrazzano.ReplaceAllString(l, "REDACTED-HOSTNAME")
	l = regex_oracle.ReplaceAllString(l, "REDACTED-HOSTNAME")
	l = regex_ip.ReplaceAllString(l, "REDACTED-IP4-ADDRESS")
	l = regex_oraclecorp.ReplaceAllString(l, "REDACTED-HOSTNAME")

	return l
}

func check(e error) error {
	if e != nil {
		return e
	}
	return nil
}
