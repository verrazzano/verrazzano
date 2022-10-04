// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const (
	copyrightTemplate = `{{- $createdYear:=.CreatedYear -}}{{- $updatedYear:=.UpdatedYear -}}{{ .Comment }} Copyright (c) {{if ne $createdYear $updatedYear }}{{printf "%s" $createdYear}}, {{end}}{{printf "%s" $updatedYear}}, Oracle and/or its affiliates.
{{ .Comment}} Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
`
)

type pattern []*regexp.Regexp

func (p *pattern) String() string {
	return fmt.Sprint(*p)
}

func (p *pattern) Set(value string) error {
	for _, val := range strings.Split(value, ",") {
		re := regexp.MustCompile(val)
		*p = append(*p, re)
	}
	return nil
}

// This program will accept a list of files and directories and scan all of the files found therin to make sure that
// they have the correct Oracle copyright header and UPL license headers.
//
// Internally, we manage a list of file extensions and relative file/directory names to ignore.  We also load a list
// of ignore paths from the working directory of the program containing a list of paths relative to that working dir
// to explicitly ignore.

var (
	// copyrightRegex is the regular expression for recognizing correctly formatted copyright statements
	// Explanation of the regular expression
	// -------------------------------------
	// ^                           matches start of the line
	// (#|\/\/|<!--|\/\*)          matches either a # character, or two / characters or the literal string "<!--", or "/*"
	// Copyright                   matches the literal string " Copyright "
	// \([cC]\)                    matches "(c)" or "(C)"
	// ([1-2][0-9][0-9][0-9], )    matches a year in the range 1000-2999 followed by a comma and a space
	// ?([1-2][0-9][0-9][0-9], )   matches an OPTIONAL second year in the range 1000-2999 followed by a comma and a space
	// Oracle ... affiliates       matches that literal string
	// (\.|\. -->|\. \*\/|\. --%>) matches "." or ". -->" or ". */"
	// $                           matches the end of the line
	// the correct copyright line looks like this:
	// Copyright (c) 2020, Oracle and/or its affiliates.
	copyrightPattern = `^(#|\/\/|<!--|\/\*|<%--) Copyright \([cC]\) ((?P<CreatedYear>[1-2][0-9][0-9][0-9]), )((?P<UpdatedYear>[1-2][0-9][0-9][0-9]), )?Oracle and\/or its affiliates(\.|\. -->|\. \*\/|\. --%>)$`
	_                = regexp.MustCompile(copyrightPattern)

	// uplRegex is the regular express for recognizing correctly formatted UPL license headers
	// Explanation of the regular expression
	// -------------------------------------
	// ^                           matches start of the line
	// (#|\/\/|<!--|\/\*|<%--)     matches either a # character, or two / characters or the literal string "<!--", "/*" or "<%--"
	// Licensed ... licenses\\/upl matches that literal string
	// (\.|\. -->|\. \*\/|\. --%>) matches "." or ". -->" or ". */" or ". --%>"
	// $                           matches the end of the line
	// the correct copyright line looks like this:
	// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
	uplPattern = `^(#|\/\/|<!--|\/\*|<%--) Licensed under the Universal Permissive License v 1\.0 as shown at https:\/\/oss\.oracle\.com\/licenses\/upl(\.|\. -->|\. \*\/|\. --%>)$`
	_          = regexp.MustCompile(uplPattern)

	copyrightUplPattern = "(?m)" + copyrightPattern + "\n" + uplPattern + "\n"
	copyrightUplRegex   = regexp.MustCompile(copyrightUplPattern)

	verbose = false

	excludePatterns  pattern = []*regexp.Regexp{}
	includePatterns  pattern = []*regexp.Regexp{}
	extensionFlagVal string

	// useExistingUpdateYearFromHeader - use the update date from the existing header
	useExistingUpdateYearFromHeader *bool
)

func shouldFilter(path string) bool {
	if len(includePatterns) > 0 {
		var shouldInclude = false
		for _, re := range includePatterns {
			if re.MatchString(path) {
				shouldInclude = true
				break
			}
		}
		if !shouldInclude {
			log.Printf("Skipping %s as it does not match include patterns %v\n", path, includePatterns)
			return true
		}
	}
	if len(excludePatterns) > 0 {
		var shouldInclude = true
		for _, re := range excludePatterns {
			if re.MatchString(path) {
				shouldInclude = false
				break
			}
		}
		if !shouldInclude {
			log.Printf("Skipping %s as it matches exclude patterns %v\n", path, includePatterns)
			return true
		}
	}
	return false
}

type GitFileStatus int

const (
	Unmodified GitFileStatus = iota
	Modified
	Added
	Deleted
	Copied
	Unmerged
	Untracked
	Ignored
)

func (s GitFileStatus) String() string {
	return [...]string{"unmodified", "modified", "added", "deleted", "renamed", "copied", "unmerged", "untracked", "ignored"}[s]
}

func ParseGitFileStatus(s string) (GitFileStatus, error) {
	switch s {
	default:
		return 0, fmt.Errorf("Unknown git file status %s", s)
	case " ":
		return Unmodified, nil
	case "M":
		return Modified, nil
	case "A":
		return Added, nil
	case "D":
		return Deleted, nil
	case "C":
		return Copied, nil
	case "U":
		return Unmerged, nil
	case "?":
		return Untracked, nil
	case "!":
		return Ignored, nil

	}
}

type GitStatus struct {
	IndexStatus    GitFileStatus
	WorkTreeStatus GitFileStatus
}

func ParseGitStatus(s string) (*GitStatus, error) {
	if strings.TrimSpace(s) == "" {
		return &GitStatus{
			IndexStatus:    Unmodified,
			WorkTreeStatus: Unmodified,
		}, nil
	}
	x, err := ParseGitFileStatus(string(s[0]))
	if err != nil {
		return nil, err
	}
	y, err := ParseGitFileStatus(string(s[1]))
	if err != nil {
		return nil, err
	}

	return &GitStatus{
		IndexStatus:    x,
		WorkTreeStatus: y,
	}, nil
}

type GitFileInfo struct {
	FileName    string
	CreatedYear string
	UpdatedYear string
	GitStatus   *GitStatus
}

type TemplateParams struct {
	Comment     string
	CreatedYear string
	UpdatedYear string
}

func gitFileInfo(path string) (*GitFileInfo, error) {
	currentYear := strconv.Itoa(time.Now().Year())

	out, err := exec.Command("git", "status", "--porcelain", "--", path).Output()
	if err != nil {
		return nil, err
	}
	log.Printf("git status %s: %v", path, string(out))
	gitStatus, err := ParseGitStatus(string(out))
	if err != nil {
		return nil, err
	}

	fi := GitFileInfo{
		FileName:    path,
		CreatedYear: currentYear,
		UpdatedYear: currentYear,
		GitStatus:   gitStatus,
	}

	// if file is untracked or added, use current year only
	if gitStatus.WorkTreeStatus == Untracked || gitStatus.WorkTreeStatus == Added {
		return &fi, nil
	}

	out, err = exec.Command("git", "log", "--format=%at", "--follow", "--", path).Output()
	if err != nil {
		return nil, err
	}
	log.Printf("git log --format=%%at --follow -- %s\n%s", path, string(out))

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	var first, last string
	for scanner.Scan() {
		if first == "" {
			first = scanner.Text()
			last = first
		} else {
			last = scanner.Text()
		}
	}
	log.Printf("git log %s: first date=%s : last date=%s\n", path, first, last)
	ilast, err := strconv.ParseInt(last, 10, 64)
	if err != nil {
		return nil, err
	}
	createdYear := strconv.Itoa(time.Unix(ilast, 0).UTC().Year())

	updatedYear := currentYear
	if gitStatus.WorkTreeStatus != Modified {
		ifirst, err := strconv.ParseInt(first, 10, 64)
		if err != nil {
			return nil, err
		}
		updatedYear = strconv.Itoa(time.Unix(ifirst, 0).UTC().Year())
	}

	log.Printf("CreatedYear %s\n", createdYear)
	log.Printf("UpdatedYear %s\n", updatedYear)
	return &GitFileInfo{
		FileName:    path,
		CreatedYear: createdYear,
		UpdatedYear: updatedYear,
		GitStatus:   gitStatus,
	}, nil
}

func renderTemplate(t *template.Template, params TemplateParams) ([]byte, error) {
	var header bytes.Buffer
	err := t.Execute(&header, params)
	if err != nil {
		return nil, err
	}
	log.Printf("rendered header: %s\n", header.String())
	return header.Bytes(), nil
}

func parseYearsFromHeader(fileContents []byte) ([]byte, string, string) {
	lengthToSearch := 1024
	if len(fileContents) < 1024 {
		lengthToSearch = len(fileContents)
	}
	firstBytes := fileContents[:lengthToSearch]
	log.Printf("firstbytes: %s", string(firstBytes))

	createdYear := ""
	updatedYear := ""
	if copyrightUplRegex.Match(firstBytes) {
		log.Printf("matched copyrightUplRegex")
		match := copyrightUplRegex.FindSubmatch(firstBytes)

		paramsMap := make(map[string]string)
		for i, name := range copyrightUplRegex.SubexpNames() {
			if i > 0 && i <= len(match) {
				paramsMap[name] = string(match[i])
			}
		}
		log.Printf("extracted regex params from parsed header: %q", paramsMap)
		createdYear = paramsMap["CreatedYear"]
		updatedYear = paramsMap["UpdatedYear"]
	}
	return firstBytes, createdYear, updatedYear
}

func fixHeaders(args []string) error {

	var err error
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return err
	}
	repoRoot := strings.TrimSpace(string(out))
	for _, arg := range args {
		err = filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("WARNING: failure accessing a path %q: %v\n", path, err)
				return err
			}
			if info.IsDir() {
				return nil
			}
			if shouldFilter(path) {
				return nil
			}
			extension := extensionFlagVal
			if extensionFlagVal == "" {
				extension = strings.ToLower(filepath.Ext(path))
				if extension == "" {
					extension = path
				}
			}
			var comment string
			switch extension {
			default:
				log.Printf("Unknown extension %s\n", extension)
				return nil
			case ".go":
				comment = "//"
			case ".yaml", ".yml":
				comment = "#"
			}
			gfi, err := gitFileInfo(path)
			if err != nil {
				log.Printf("Error getting git file info for path %s: %v", path, err)
				return err
			}
			log.Printf("Git file info: %v\n", gfi)

			t, err := template.New("").Parse(copyrightTemplate)
			if err != nil {
				return err
			}

			params := TemplateParams{
				Comment:     comment,
				CreatedYear: gfi.CreatedYear,
				UpdatedYear: gfi.UpdatedYear,
			}

			fileContents, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			var replacement []byte
			// if file already contains header, use the created year from that copyright header
			firstBytes, createdYearFromHeader, updatedYearFromHeader := parseYearsFromHeader(fileContents)
			modifyExistingHeader := true
			if createdYearFromHeader == "" {
				modifyExistingHeader = false
				// No header matches in file
				if gfi.GitStatus.WorkTreeStatus == Modified || gfi.GitStatus.IndexStatus == Modified {
					log.Printf("No copyright header in file but modified, checking version-controlled file for header for %s", path)
					// Check HEAD revision to see if the header matches there in modified files
					gitPath, err := filepath.Rel(repoRoot, path)
					if err != nil {
						return err
					}
					getGitHead := fmt.Sprintf("HEAD:%s", gitPath)
					cmd := exec.Command("git", "show", getGitHead)
					out, err := cmd.Output()
					if err != nil {
						return err
					}
					_, createdYearFromHeader, updatedYearFromHeader = parseYearsFromHeader(out)
				}
			}

			// Always trust the created year in the file header
			if createdYearFromHeader != "" {
				log.Printf("Using created year in copyright header %s, created year derived from Git is %s\n", createdYearFromHeader, gfi.CreatedYear)
				params.CreatedYear = createdYearFromHeader
			}

			// Determine if updated year from header is to be trusted over the year derived from git log history.
			if *useExistingUpdateYearFromHeader {
				log.Printf("Using updated year from existing header, UpdatedYear = %s", updatedYearFromHeader)
				params.UpdatedYear = createdYearFromHeader
				if updatedYearFromHeader != "" {
					params.UpdatedYear = updatedYearFromHeader
				}
			}

			header, err := renderTemplate(t, params)
			if err != nil {
				return err
			}

			if modifyExistingHeader {
				replacementHeader := copyrightUplRegex.ReplaceAll(firstBytes, header)
				if !bytes.Equal(firstBytes, replacementHeader) {
					replacement = append(replacementHeader, fileContents[len(firstBytes):]...)
				}
			} else {
				replacement = append(header, fileContents...)
			}

			if !bytes.Equal(replacement, []byte{}) {
				st, err := os.Stat(path)
				if err != nil {
					return err
				}
				err = ioutil.WriteFile(path, replacement, st.Mode())
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			log.Printf("error walking the path %q: %v\n", arg, err)
			return err
		}
	}
	return nil
}

// printUsage Prints the help for this program
func printUsage() {
	usageString := `
Usage: %s [options] path1 [path2 path3 ...]
Options:
`
	fmt.Printf(usageString, os.Args[0])
	flag.PrintDefaults()
}

func init() {
	flag.Var(&includePatterns, "include", "comma separated include regexp file filters")
	flag.Var(&excludePatterns, "exclude", "comma separated exclude regexp file filter")
	useExistingUpdateYearFromHeader = flag.Bool("useExistingUpdateYearFromHeader", false, "use years from existing headers in SCM if they exist")
}

func main() {

	help := false
	flag.StringVar(&extensionFlagVal, "extension", "", "Filename extension to force")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&help, "help", false, "Display usage help")
	flag.Usage = printUsage
	flag.Parse()

	if !verbose {
		log.SetOutput(ioutil.Discard)
	}

	if help {
		flag.Usage()
		os.Exit(0)
	}

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	err := fixHeaders(flag.Args())
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
