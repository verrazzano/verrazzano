// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// SearchWriter writes to a search endpoint, as an io.Writer and zapcore.WriteSyncer S
type SearchWriter struct {
	hc    *retryablehttp.Client
	url   string
	index string
	auth  string
}

// SearchWriterFromEnv creates a SearchWriter using environment variables
func SearchWriterFromEnv(index string) (SearchWriter, error) {
	uri := os.Getenv(searchURL)
	if uri == "" {
		return SearchWriter{}, fmt.Errorf("%s is empty", searchURL)
	}
	auth := ""
	user := os.Getenv(searchUser)
	pw := os.Getenv(searchPW)
	if user != "" && pw != "" {
		auth = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, pw)))
	}

	return SearchWriter{
		hc:    retryablehttp.NewClient(),
		url:   uri,
		index: index,
		auth:  auth,
	}, nil
}

// Close implement as needed
func (s SearchWriter) Close() error {
	return nil
}

// Sync implement as needed
func (s SearchWriter) Sync() error {
	return nil
}

// postRecord sends the reader record to the search data store via HTTP Post
// Basic Authorization is used, if encoded auth is provided for basicAuth
func postRecord(hc *retryablehttp.Client, basicAuth, uri string, reader io.Reader) error {
	req, err := retryablehttp.NewRequest("POST", uri, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if basicAuth != "" { // Basic Auth is used if provided
		req.Header.Set("Authorization", fmt.Sprintf("basic %s", basicAuth))
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	if resp != nil && resp.StatusCode != http.StatusCreated {
		logger.Errorf("error when posting record[%d: %s]", resp.StatusCode, resp.Status)
	}
	return nil
}

// Write out the record to the search data store
func (s SearchWriter) Write(data []byte) (int, error) {
	index := s.timeStampIndex()

	if strings.Contains(index, "metrics") {
		v := string(data)
		fmt.Println(v)
	}

	uri := fmt.Sprintf("%s/%s/_doc", s.url, index)
	reader := bytes.NewReader(data)
	if err := postRecord(s.hc, s.auth, uri, reader); err != nil {
		return 0, err
	}

	return len(data), nil
}

// timeStampIndex formats the current index in %s-YYYY.mm.dd format
func (s SearchWriter) timeStampIndex() string {
	return fmt.Sprintf("%s-%s", s.index, time.Now().Format(timeFormatString))
}
