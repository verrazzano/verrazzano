// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
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
	Started           = "started"
	Status            = "status"
	attempts          = "attempts"
	test              = "test"
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

	kubernetesVersion := os.Getenv("K8S_VERSION_LABEL")
	if kubernetesVersion != "" {
		log = log.With(KubernetesVersion, kubernetesVersion)
	}

	branchName := os.Getenv("BRANCH_NAME")
	if branchName != "" {
		log = log.With(BranchName, branchName)
	}

	buildURL := os.Getenv("BUILD_URL")
	if buildURL != "" {
		buildURL = strings.Replace(buildURL, "%252F", "/", 1)
		log = log.With(BuildURL, buildURL)
	}

	jobName := os.Getenv("JOB_NAME")
	if jobName != "" {
		jobName = strings.Replace(jobName, "%252F", "/", 1)
		jobNameSplit := strings.Split(jobName, "/")
		jobPipeline := jobNameSplit[0]
		log = log.With(JenkinsJob, jobPipeline)
	}

	gitCommit := os.Getenv("GIT_COMMIT")
	//Tagging commit with the branch.
	if gitCommit != "" {
		gitCommitAndBranch := branchName + "/" + gitCommit
		log = log.With(CommitHash, gitCommitAndBranch)
	}

	testEnv := os.Getenv("TEST_ENV")
	if testEnv != "" {
		log = log.With(TestEnv, testEnv)
	}

	return log
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

func Emit(log *zap.SugaredLogger) {
	spec := ginkgo.CurrentSpecReport()
	if spec.State != types.SpecStateInvalid {
		log = log.With(Status, spec.State)
	}
	t := spec.FullText()
	l := spec.Labels()

	log.With(attempts, spec.NumAttempts).
		With(test, t).
		With(Label, l).
		Info()
}

func DurationMillis() int64 {
	// this value is in nanoseconds, so we need to divide by one million
	// to convert to milliseconds
	spec := ginkgo.CurrentSpecReport()
	return int64(spec.RunTime) / 1_000_000
}
