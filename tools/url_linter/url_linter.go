// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"bufio"
	_ "embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

var (
	urlRE = regexp.MustCompile(`(http|ftp|https):\/\/([\w_-]+(?:(?:\.[\w_-]+)+))([\w.,@?^=%&:\/~+#-]*[\w@?^=%&\/~+#-])`)
	//go:embed ignore_urls.txt
	rawignoredURLs string
	ignoredURLs    = make(map[string]bool)
)

type scanResult struct {
	urlToStatus map[string]int
	err         error
}

func initignoredURLs() {
	scanner := bufio.NewScanner(strings.NewReader(rawignoredURLs))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix("#", line) {
			continue
		}
		s := strings.Fields(line)
		if len(s) >= 1 {
			ignoredURLs[s[0]] = true
		}
	}

}

func gitTopLevelDir() (string, error) {
	stdout, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(stdout)), nil
}

func gitLsFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--exclude-standard", "--cached")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(stdout)

	var files []string
	for scanner.Scan() {
		file := scanner.Text()
		files = append(files, file)
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return files, nil

}

func scanFileForURLs(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	uniqueURLs := make(map[string]bool)
	var urls []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		matches := urlRE.FindAllString(scanner.Text(), -1)
		for _, match := range matches {
			match = strings.TrimSuffix(match, ".")
			if ignoredURLs[match] {
				continue
			}
			if strings.Contains(match, "localhost") || strings.Contains(match, "127.") || strings.Contains(match, "%s") || strings.Contains(match, "%d") {
				continue
			}
			if _, value := uniqueURLs[match]; !value {
				uniqueURLs[match] = true
				urls = append(urls, match)
			}
		}
	}
	return urls, nil
}

func findURLs(gitRoot string, files []string) (map[string]*scanResult, error) {

	fileToScanResults := make(map[string]*scanResult)

	var absoluteFiles []string
	for _, file := range files {
		if strings.HasPrefix(file, "ci/") || strings.HasPrefix(file, "tests/e2e/") || strings.HasPrefix(file, "platform-operator/thirdparty/") || strings.Contains(file, "/testdata/") ||
		strings.Contains(file, "/test/") || strings.HasSuffix(file, "_test.go") {
			continue
		}
		absoluteFile := path.Join(gitRoot, file)
		absoluteFiles = append(absoluteFiles, absoluteFile)
		fileToScanResults[absoluteFile] = &scanResult{
			urlToStatus: make(map[string]int),
		}
	}

	numWorkers := runtime.GOMAXPROCS(-1)

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	filesCh := make(chan string, len(files))
	urlToFiles := make(map[string][]string)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for file := range filesCh {
				lowerFile := strings.ToLower(file)
				if strings.HasSuffix(lowerFile, ".md") || strings.HasSuffix(lowerFile, ".html") {
					urls, err := scanFileForURLs(file)

					scanResult := fileToScanResults[file]
					scanResult.err = err
					for _, url := range urls {
						scanResult.urlToStatus[url] = -1
					}
				}
			}
		}()
	}

	for _, file := range absoluteFiles {
		filesCh <- file
	}
	close(filesCh)
	wg.Wait()

	for f, sr := range fileToScanResults {
		for u := range sr.urlToStatus {
			urlToFiles[u] = append(urlToFiles[u], f)
		}
	}

	var wg2 sync.WaitGroup
	wg2.Add(numWorkers)
	urlsCh := make(chan string, len(urlToFiles))
	type status struct {
		statusCode int
	}
	urlToStatusCode := make(map[string]*status)
	for url := range urlToFiles {
		urlToStatusCode[url] = &status{
			statusCode: -1,
		}
	}

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg2.Done()
			for url := range urlsCh {
				client := &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				}
				res, err := client.Head(url)
				if err != nil {
					fmt.Printf("%v\n", err)
					continue
				}
				res.Body.Close()
				urlToStatusCode[url].statusCode = res.StatusCode
			}
		}()
	}
	for url := range urlToFiles {
		urlsCh <- url
	}
	close(urlsCh)
	wg2.Wait()

	for _, sr := range fileToScanResults {
		for u := range sr.urlToStatus {
			sr.urlToStatus[u] = urlToStatusCode[u].statusCode
		}
	}

	return fileToScanResults, nil
}

func main() {

	var verbose bool
	var help bool
	var concurrency = runtime.NumCPU()

	flag.IntVar(&concurrency, "concurrency", concurrency, "Concurrency - default is the number of CPUs")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&help, "help", false, "Display usage help")
	flag.Parse()

	if help {
		//printUsage()
		os.Exit(0)
	}

	runtime.GOMAXPROCS(concurrency)

	gitTopLevel, err := gitTopLevelDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}

	files, err := gitLsFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
	initignoredURLs()

	fileToScanResults, err := findURLs(gitTopLevel, files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}

	var deadURLs = make(map[string][]string)
	var relocatedURLs = make(map[string][]string)
	for f, sr := range fileToScanResults {
		if len(sr.urlToStatus) == 0 {
			continue
		}
		for u, s := range sr.urlToStatus {
			switch s {
			case -1, 404:
				files := append(deadURLs[u], f)
				deadURLs[u] = files
			case 301, 302, 303, 307, 308:
				files := append(relocatedURLs[u], f)
				relocatedURLs[u] = files
			}
		}
	}
	fmt.Printf("Dead URLs\n")
	for u, files := range deadURLs {
		fmt.Printf("%s\n", u)
		for _, file := range files {
			fmt.Printf("\t%s\n", file)
		}
	}
	fmt.Printf("Relocated URLs\n")
	for u, files := range relocatedURLs {
		fmt.Printf("%s\n", u)
		for _, file := range files {
			fmt.Printf("\t%s\n", file)
		}
	}

	if len(deadURLs) > 0 { 
		os.Exit(1)
	}
	os.Exit(0)
}
