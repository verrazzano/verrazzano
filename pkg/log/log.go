// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// InitLogs initializes logs with Time and Global Level of Logs set at Info
func InitLogs(opts kzap.Options) {
	var config zap.Config
	if opts.Development {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}
	if opts.Level != nil {
		config.Level = opts.Level.(zap.AtomicLevel)
	} else {
		config.Level.SetLevel(zapcore.InfoLevel)
	}
	config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	config.EncoderConfig.TimeKey = "@timestamp"
	config.EncoderConfig.MessageKey = "message"
	logger, err := config.Build()
	if err != nil {
		zap.S().Errorf("Error creating logger %v", err)
	} else {
		zap.ReplaceGlobals(logger)
	}

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	opts.ZapOpts = append(opts.ZapOpts, zap.AddCaller())
	encoder := zapcore.NewJSONEncoder(config.EncoderConfig)
	logf.SetLogger(kzap.New(kzap.UseFlagOptions(&opts), kzap.Encoder(encoder)))
}
