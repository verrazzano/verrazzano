// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"encoding/json"
	"fmt"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/util/uuid"
	neturl "net/url"
	"os"
	"strings"
)

const (
	Duration          = "duration"
	Status            = "status"
	attempts          = "attempts"
	test              = "test"
	codeLocation      = "code_location"
	fullSpecJSON      = "spec_report"
	stageName         = "stage_name"
	BuildURL          = "build_url"
	JenkinsJob        = "jenkins_job"
	BranchName        = "branch_name"
	CommitHash        = "commit_hash"
	KubernetesVersion = "kubernetes_version"
	TestEnv           = "test_env"
	Label             = "label"

	MetricsIndex     = "metrics"
	TestLogIndex     = "testlogs"
	searchWriterKey  = "searchWriter"
	timeFormatString = "2006.01.02"
	searchURL        = "SEARCH_HTTP_ENDPOINT"
	searchPW         = "SEARCH_PASSWORD"
	searchUser       = "SEARCH_USERNAME"
	stageNameEnv     = "STAGE_NAME"
)

var logger = internalLogger()

func internalLogger() *zap.SugaredLogger {
	cfg := zap.Config{
		Encoding: "json",
		Level:    zap.NewAtomicLevelAt(zapcore.InfoLevel),
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "msg",
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

// NewLogger generates a new logger, and tees ginkgo output to the search db
func NewLogger(pkg string, ind string, paths ...string) (*zap.SugaredLogger, error) {
	var messageKey = zapcore.OmitKey
	if ind == TestLogIndex {
		messageKey = "msg"
	}
	cfg := zap.Config{
		Encoding: "json",
		Level:    zap.NewAtomicLevelAt(zapcore.InfoLevel),
		EncoderConfig: zapcore.EncoderConfig{

			MessageKey: messageKey,

			LevelKey: zapcore.OmitKey,

			TimeKey:    "timestamp",
			EncodeTime: zapcore.EpochMillisTimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	outputPaths, err := configureOutputs(ind)
	if err != nil {
		logger.Errorf("failed to configure outputs: %v", err)
		return nil, err
	}
	cfg.OutputPaths = append(outputPaths, paths...)
	log, err := cfg.Build()
	if err != nil {
		logger.Errorf("error creating %s logger %v", pkg, err)
		return nil, err
	}

	suiteUUID := uuid.NewUUID()
	sugaredLogger := log.Sugar().With("suite_uuid", suiteUUID).With("package", pkg)
	return configureLoggerWithJenkinsEnv(sugaredLogger), nil
}

func configureLoggerWithJenkinsEnv(log *zap.SugaredLogger) *zap.SugaredLogger {
	const separator = "/"
	log, _ = withEnvVar(log, KubernetesVersion, "K8S_VERSION_LABEL")
	log, branchName := withEnvVar(log, BranchName, "BRANCH_NAME")
	log, _ = withEnvVarMutate(log, BuildURL, "BUILD_URL", func(buildURL string) string {
		buildURL, _ = neturl.QueryUnescape(buildURL)
		return buildURL
	})
	log, _ = withEnvVarMutate(log, JenkinsJob, "JOB_NAME", func(jobName string) string {
		jobName, _ = neturl.QueryUnescape(jobName)
		jobNameSplit := strings.Split(jobName, separator)
		return jobNameSplit[0]
	})
	log, _ = withEnvVarMutate(log, CommitHash, "GIT_COMMIT", func(gitCommit string) string {
		return branchName + separator + gitCommit
	})
	log, _ = withEnvVar(log, TestEnv, "TEST_ENV")
	return log
}

// withEnvVarMutate enriches the logger with the mutated value of envVar, if it exists
func withEnvVarMutate(log *zap.SugaredLogger, withKey, envVar string, mutateFunc func(string) string) (*zap.SugaredLogger, string) {
	val := os.Getenv(envVar)
	if len(val) > 0 {
		val = mutateFunc(val)
		log = log.With(withKey, val)
	}
	return log, val
}

// withEnvVar enriches the logger with the value of envVar, if it exists in the environment
func withEnvVar(log *zap.SugaredLogger, withKey, envVar string) (*zap.SugaredLogger, string) {
	return withEnvVarMutate(log, withKey, envVar, func(val string) string {
		return val
	})
}

// configureOutputs configures the search output path if it is available
func configureOutputs(ind string) ([]string, error) {
	var outputs []string
	searchWriter, err := SearchWriterFromEnv(ind)
	sinkKey := fmt.Sprintf("%s%s", searchWriterKey, ind)
	// Register SearchWriter
	if err == nil {
		if err := zap.RegisterSink(sinkKey, func(u *neturl.URL) (zap.Sink, error) {
			return searchWriter, nil
		}); err != nil {
			return nil, err
		}
		outputs = append(outputs, sinkKey+":search")
	}

	return outputs, nil
}

func EmitFail(log *zap.SugaredLogger) {
	spec := ginkgo.CurrentSpecReport()
	log = log.With(Status, types.SpecStateFailed)
	emitInternal(log, spec)
}

func Emit(log *zap.SugaredLogger) {
	spec := ginkgo.CurrentSpecReport()
	if spec.State == types.SpecStatePassed {
		log = log.With(Status, spec.State)
	}
	emitInternal(log, spec)
}

func emitInternal(log *zap.SugaredLogger, spec ginkgo.SpecReport) {
	t := spec.FullText()
	l := spec.Labels()
	log = withCodeLocation(log, spec)
	log, _ = withEnvVar(log, stageName, stageNameEnv)
	log = withSpecJSON(log, spec)
	log.With(attempts, spec.NumAttempts,
		test, t,
		Label, l).
		Info()
}

func withSpecJSON(log *zap.SugaredLogger, spec ginkgo.SpecReport) *zap.SugaredLogger {
	specJSON, err := spec.MarshalJSON()
	raw := json.RawMessage(specJSON)
	if err == nil {
		log = log.With(fullSpecJSON, raw)
	}
	return log
}

func withCodeLocation(log *zap.SugaredLogger, spec ginkgo.SpecReport) *zap.SugaredLogger {
	filePath := spec.LeafNodeLocation.FileName
	if len(filePath) < 1 {
		return log
	}
	lineNumber := spec.LeafNodeLocation.LineNumber
	split := strings.Split(filePath, "/")
	fileName := split[len(split)-1]
	return log.With(codeLocation, fmt.Sprintf("%s:%d", fileName, lineNumber))
}

func DurationMillis() int64 {
	// this value is in nanoseconds, so we need to divide by one million
	// to convert to milliseconds
	spec := ginkgo.CurrentSpecReport()
	return int64(spec.RunTime) / 1_000_000
}
