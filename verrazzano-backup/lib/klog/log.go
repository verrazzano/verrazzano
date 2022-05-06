// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package klog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
)

func Logger(filePath string) (*zap.SugaredLogger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Println("Unable to initialize klog")
	}
	defer logger.Sync()

	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
		OutputPaths:      []string{filePath, "stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "message",
			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}
	logger, err = cfg.Build()
	if err != nil {
		logger.Sugar().Errorf("Unable to build logger due to %v", err)
		return nil, err
	}
	return logger.Sugar(), nil
}
