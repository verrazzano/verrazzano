// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/util/uuid"
)

const (
	status   = "status"
	attempts = "attempts"
	test     = "test"
)

const (
	pending = "pending"
	passed  = "passed"
	failed  = "failed"
)

func NewForPackage(pkg string) (*zap.SugaredLogger, error) {
	cfg := zap.Config{
		Encoding: "json",
		Level:    zap.NewAtomicLevelAt(zapcore.InfoLevel),
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: zapcore.OmitKey,

			LevelKey: zapcore.OmitKey,

			TimeKey:    "timestamp",
			EncodeTime: zapcore.EpochNanosTimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}
	cfg.OutputPaths = []string{"stdout"}
	log, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return log.Sugar().With("id", uuid.NewUUID()).With("package", pkg), nil
}

func Emit(log *zap.SugaredLogger) {
	spec := ginkgo.CurrentSpecReport()
	s := getStatus(spec.State)
	t := spec.LeafNodeText

	log.With(status, s).
		With(attempts, spec.NumAttempts).
		With(test, t).
		Info()
}

func getStatus(state types.SpecState) string {
	if state == types.SpecStateInvalid {
		return pending
	}
	return state.String()
}
