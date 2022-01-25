// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"fmt"
	neturl "net/url"
	"os"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/discovery"
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
	CommitSHA         = "commit_sha"
	KubernetesVersion = "kubernetes_version"
	TestEnv           = "test_env"

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

//NewLogger generates a new logger, and tees ginkgo output to the search db
func NewLogger(pkg string, ind string) (*zap.SugaredLogger, error) {
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

	outputPaths, err := configureOutputs(ind)
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

	suiteUUID := uuid.NewUUID()
	sugaredLogger := log.Sugar().With("suite_uuid", suiteUUID).With("package", pkg)
	return configureLoggerWithJenkinsEnv(sugaredLogger), nil
}

func getKubernetesVersion() (string, error) {

	var kubeVersion string
	kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		logger.Errorf("error getting kubeconfig path:  %v", err)
		return kubeVersion, err
	}
	kubeConfig, err := k8sutil.GetKubeConfigGivenPath(kubeConfigPath)

	if err != nil {
		logger.Errorf("error getting kubeconfig:  %v", err)
		return kubeVersion, err
	}

	discover, err := discovery.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		logger.Errorf("error getting discovery client:  %v", err)
		return kubeVersion, err
	}

	version, err := discover.ServerVersion()
	if err != nil {
		logger.Errorf("error getting ServerVersion info:  %v", err)
		return kubeVersion, err
	}
	kubeVersion = version.Major + "." + version.Minor

	return kubeVersion, nil
}

func configureLoggerWithJenkinsEnv(log *zap.SugaredLogger) *zap.SugaredLogger {

	kubernetesVersion, err := getKubernetesVersion()

	if err == nil {
		log = log.With(KubernetesVersion, kubernetesVersion)
	}

	branchName := os.Getenv("BRANCH_NAME")
	if branchName != "" {
		log = log.With(BranchName, branchName)
	}

	buildURL := os.Getenv("BUILD_URL")

	//Build number is retrieved from the Build url.
	if buildURL != "" {
		buildURL = strings.Replace(buildURL, "%252F", "/", 1)
		buildAPIURL, _ := neturl.Parse(buildURL)
		jenkinsJob := buildAPIURL.Path[5:]
		log = log.With(BuildURL, buildURL).With(JenkinsJob, jenkinsJob)
	}

	gitCommit := os.Getenv("GIT_COMMIT")
	//Tagging commit with the branch.
	if gitCommit != "" {
		gitCommitAndBranch := branchName + "/" + gitCommit
		log = log.With(CommitSHA, gitCommitAndBranch)
	}

	testEnv := os.Getenv("TEST_ENV")
	if testEnv != "" {
		log = log.With(TestEnv, testEnv)
	}

	return log
}

func Millis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

//configureOutputs configures the search output path if it is available
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

	log.With(attempts, spec.NumAttempts).
		With(test, t).
		Info()
}

func DurationMillis() int64 {
	spec := ginkgo.CurrentSpecReport()
	return int64(spec.RunTime) / 1000
}
