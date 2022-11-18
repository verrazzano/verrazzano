// Copyright (C) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package log

import (
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime/pkg/log"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const timeFormat = "2006-01-02T15:04:05.000Z"

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
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(timeFormat)
	config.EncoderConfig.TimeKey = "@timestamp"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.CallerKey = "caller"
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
	//
	// Add the caller field as an option otherwise the controller runtime logger
	// will not include the caller field.
	opts.ZapOpts = append(opts.ZapOpts, zap.AddCaller())
	encoder := zapcore.NewJSONEncoder(config.EncoderConfig)
	ctrl.SetLogger(kzap.New(kzap.UseFlagOptions(&opts), kzap.Encoder(encoder)))
}

// ConflictWithLog returns a conflict error and logs a message
// Returned is an error
func ConflictWithLog(message string, err error, log *zap.SugaredLogger) error {
	if err == nil {
		return nil
	}
	if k8serrors.IsConflict(err) {
		log.Debugf("%s: %v", message, err)
	} else {
		log.Errorf("%s: %v", message, err)
	}
	return err
}

// ResultErrorsWithLog logs an error message for any error that is not a conflict error.  Conflict errors are logged
// with debug level messages.
func ResultErrorsWithLog(message string, errors []error, log *zap.SugaredLogger) {
	for _, err := range errors {
		ConflictWithLog(message, err, log)
	}
}

// IgnoreConflictWithLog ignores conflict error and logs a message
// Returned is a result and an error
func IgnoreConflictWithLog(message string, err error, log *zap.SugaredLogger) (reconcile.Result, error) {
	if err == nil {
		return reconcile.Result{}, nil
	}
	if k8serrors.IsConflict(err) {
		log.Debugf("%s: %v", message, err)
	} else {
		log.Errorf("%s: %v", message, err)
	}
	return vzctrl.NewRequeueWithDelay(2, 3, time.Second), nil
}

// BuildZapInfoLogger initializes a zap logger at info level
func BuildZapInfoLogger(callerSkip int) (*zap.SugaredLogger, error) {
	return BuildZapLoggerWithLevel(callerSkip, zapcore.InfoLevel)
}

// BuildZapLoggerWithLevel initializes a zap logger for a given log level
func BuildZapLoggerWithLevel(callerSkip int, level zapcore.Level) (*zap.SugaredLogger, error) {
	config := zap.NewProductionConfig()
	config.Level.SetLevel(level)

	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(timeFormat)
	config.EncoderConfig.TimeKey = "@timestamp"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.CallerKey = "caller"
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	l := logger.WithOptions(zap.AddCallerSkip(callerSkip))
	return l.Sugar(), nil
}
