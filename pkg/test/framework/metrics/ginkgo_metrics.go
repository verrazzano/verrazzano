// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"encoding/json"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/util/uuid"
	neturl "net/url"
	"time"
)

const (
	Duration = "duration"
	Started  = "started"
	Status   = "status"
	attempts = "attempts"
	test     = "test"

	metricsIndex     = "metrics"
	testLogIndex     = "testlogs"
	searchWriterKey  = "searchWriter"
	timeFormatString = "2006.01.02"
	searchURL        = "SEARCH_HTTP_ENDPOINT"
	searchPW         = "SEARCH_PASSWORD"
	searchUser       = "SEARCH_USERNAME"
)

type (
	MetricWriter interface {
		zapcore.WriteSyncer
		FromEnv() zapcore.WriteSyncer
	}

	GinkgoLogFormatter struct {
		writers []zapcore.WriteSyncer
	}
	GinkgoLogMessage struct {
		Data      string `json:"msg"`
		Timestamp int64  `json:"timestamp"`
		Test      string `json:"test,omitempty"`
		Status    string `json:"status,omitempty"`
	}
)

var logger = internalLogger()

func internalLogger() *zap.SugaredLogger {
	cfg := zap.Config{
		Encoding: "json",
		Level:    zap.NewAtomicLevelAt(zapcore.InfoLevel),
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "msg",
			LevelKey:     "level",
			TimeKey:      "timestamp",
			EncodeTime:   zapcore.EpochMillisTimeEncoder,
			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
		OutputPaths: []string{"stdout"},
	}

	log, err := cfg.Build()
	if err != nil {
		panic("failed to create internal logger")
	}
	return log.Sugar()
}

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
		logger.Errorf("failed to configure outputs: %v", err)
		return nil, err
	}
	cfg.OutputPaths = outputPaths
	log, err := cfg.Build()
	if err != nil {
		logger.Errorf("error creating %s logger %v", pkg, err)
		return nil, err
	}

	TeeWriters()
	return log.Sugar().With("suite_uuid", uuid.NewUUID()).With("package", pkg), nil
}

func (g GinkgoLogFormatter) Write(data []byte) (int, error) {
	//spec := ginkgo.CurrentSpecReport()
	msg := GinkgoLogMessage{
		Data:      string(data),
		Timestamp: Millis(),
		//Test:      spec.LeafNodeText,
		//Status:    spec.State.String(),
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}
	for _, writer := range g.writers {
		_, err := writer.Write(msgData)
		if err != nil {
			logger.Errorf("error when writing data for writer: %v", err)
		}
	}
	return len(msgData), nil
}

func Millis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

//configureOutputs configures the search output path if it is available
func configureOutputs() ([]string, error) {
	outputs := []string{"stdout"}
	searchWriter, err := SearchWriterFromEnv(metricsIndex)

	// Register SearchWriter
	if err == nil {
		if err := zap.RegisterSink(searchWriterKey, func(u *neturl.URL) (zap.Sink, error) {
			return searchWriter, nil
		}); err != nil {
			return nil, err
		}
		outputs = append(outputs, searchWriterKey+":search")
	}

	return outputs, nil
}

//TeeWriters adds any WriteSyncer implementations to the Ginkgo output tee
func TeeWriters() {
	var writers []zapcore.WriteSyncer
	searchWriter, err := SearchWriterFromEnv(testLogIndex)
	if err != nil {
		logger.Debug("configured new SearchWriter")
		writers = append(writers, searchWriter)
	}

	if len(writers) > 0 {
		logFormatter := GinkgoLogFormatter{writers: writers}
		ginkgo.GinkgoWriter.TeeTo(logFormatter)
	}
}

func Emit(log *zap.SugaredLogger) {
	spec := ginkgo.CurrentSpecReport()
	if spec.State != types.SpecStateInvalid {
		log = log.With(Status, spec.State)
	}
	t := spec.LeafNodeText

	log.With(attempts, spec.NumAttempts).
		With(test, t).
		Info()
}

func DurationMillis() int64 {
	spec := ginkgo.CurrentSpecReport()
	return int64(spec.RunTime) / 1000
}
