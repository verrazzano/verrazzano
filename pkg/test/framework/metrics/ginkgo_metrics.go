// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"k8s.io/apimachinery/pkg/util/uuid"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
	"time"
)

const (
	Started  = "started"
	Status   = "status"
	attempts = "attempts"
	test     = "test"
	pending  = "pending"

	metricsIndex     = "metrics"
	testLogIndex     = "testlogs"
	searchWriterKey  = "searchWriter"
	timeFormatString = "2006.01.02"
	searchURL        = "SEARCH_HTTP_ENDPOINT"
	searchPW         = "SEARCH_PASSWORD"
	searchUser       = "SEARCH_USERNAME"
)

type (
	//SearchWriter writes to a search endpoint, as an io.Writer and zapcore.WriteSyncer
	SearchWriter struct {
		hc    *http.Client
		url   string
		index string
		auth  string
	}

	GinkgoLogFormatter struct {
		s SearchWriter
	}
	GinkgoLogMessage struct {
		Data      string `json:"msg"`
		Timestamp int64  `json:"timestamp"`
		Test      string `json:"test,omitempty"`
		Status    string `json:"status,omitempty"`
	}
)

//NewMetricsLogger generates a new metrics logger, and tees ginkgo output to the search db
func NewMetricsLogger(pkg string) (*zap.SugaredLogger, error) {
	cfg := zap.Config{
		Encoding: "json",
		Level:    zap.NewAtomicLevelAt(zapcore.InfoLevel),
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: zapcore.OmitKey,

			LevelKey: zapcore.OmitKey,

			TimeKey:    "timestamp",
			EncodeTime: zapcore.EpochMillisTimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	outputPaths, err := configureOutputs()
	if err != nil {
		return nil, err
	}
	cfg.OutputPaths = outputPaths
	log, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	TeeToSearchWriter()
	return log.Sugar().With("suite_uuid", uuid.NewUUID()).With("package", pkg), nil
}

func (g GinkgoLogFormatter) Write(data []byte) (int, error) {
	//spec := ginkgo.CurrentSpecReport()
	msg := GinkgoLogMessage{
		Data:      string(data),
		Timestamp: timestamp(),
		//Test:      spec.LeafNodeText,
		//Status:    spec.State.String(),
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}
	return g.s.Write(msgData)
}

func timestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

//SearchWriterFromEnv creates a SearchWriter using environment variables
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
		hc:    &http.Client{},
		url:   uri,
		index: index,
		auth:  auth,
	}, nil
}

//Close implement as needed
func (s SearchWriter) Close() error {
	return nil
}

//Sync implement as needed
func (s SearchWriter) Sync() error {
	return nil
}

//Write out the record to the search data store
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

//postRecord sends the reader record to the search data store via HTTP Post
// Basic Authorization is used, if encoded auth is provided for basicAuth
func postRecord(hc *http.Client, basicAuth, uri string, reader io.Reader) error {
	req, err := http.NewRequest("POST", uri, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if basicAuth != "" { // Basic Auth is used if provided
		req.Header.Set("Authorization", basicAuth)
	}
	_, err = hc.Do(req)
	return err
}

//timeStampIndex formats the current index in %s-YYYY.mm.dd format
func (s SearchWriter) timeStampIndex() string {
	return fmt.Sprintf("%s-%s", s.index, time.Now().Format(timeFormatString))
}

//configureOutputs configures the search output path if it is available
func configureOutputs() ([]string, error) {
	searchWriter, err := SearchWriterFromEnv(metricsIndex)
	if err != nil {
		return []string{"stdout"}, nil
	}

	if err := zap.RegisterSink(searchWriterKey, func(u *neturl.URL) (zap.Sink, error) {
		return searchWriter, nil
	}); err != nil {
		return nil, err
	}

	return []string{searchWriterKey + ":search"}, nil
}

func TeeToSearchWriter() {
	searchWriter, err := SearchWriterFromEnv(testLogIndex)
	if err == nil {
		logFormatter := GinkgoLogFormatter{s: searchWriter}
		ginkgo.GinkgoWriter.TeeTo(logFormatter)
	}
}

func Emit(log *zap.SugaredLogger) {
	spec := ginkgo.CurrentSpecReport()
	if spec.State != types.SpecStateInvalid {
		log = log.With(Status, spec.State.String())
	}
	t := spec.LeafNodeText

	log.With(attempts, spec.NumAttempts).
		With(test, t).
		Info()
}
