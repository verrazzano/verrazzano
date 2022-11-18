// Copyright (C) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package log

import (
	"fmt"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"

	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestInitLogsDefaultInfo(t *testing.T) {
	InitLogs(kzap.Options{})
	zap.S().Errorf("greeting %v", "hello")
	zap.S().Infof("greeting %v", "hello")
	zap.S().Debugf("greeting %v", "hello")
	msg := "InfoLevel is enabled"
	assert.NotNil(t, zap.L().Check(zapcore.InfoLevel, msg), msg)
	msg = "ErrorLevel is enabled"
	assert.NotNil(t, zap.L().Check(zapcore.ErrorLevel, msg), msg)
	msg = "DebugLevel is disabled"
	assert.Nil(t, zap.L().Check(zapcore.DebugLevel, msg), msg)
}

func TestInitLogsNonDefaultInfo(t *testing.T) {
	testOpts := kzap.Options{}
	testOpts.Development = true
	testOpts.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	InitLogs(testOpts)
	zap.S().Errorf("greeting %v", "hello")
	zap.S().Infof("greeting %v", "hello")
	zap.S().Debugf("greeting %v", "hello")
	msg := "InfoLevel is enabled"
	assert.NotNil(t, zap.L().Check(zapcore.InfoLevel, msg), msg)
	msg = "ErrorLevel is enabled"
	assert.NotNil(t, zap.L().Check(zapcore.ErrorLevel, msg), msg)
	msg = "DebugLevel is disabled"
	assert.Nil(t, zap.L().Check(zapcore.DebugLevel, msg), msg)
}

func TestConflictWithLog(t *testing.T) {
	// Random error
	err := ConflictWithLog("test-message", fmt.Errorf("test-error"), zap.S())
	assert.NotNil(t, err)

	// Conflict Error
	err = ConflictWithLog("test-message", k8serrors.NewConflict(schema.GroupResource{}, "test-conflict", fmt.Errorf("test-error")), zap.S())
	assert.NotNil(t, err)

	// No Error
	err = ConflictWithLog("test-message", nil, zap.S())
	assert.Nil(t, err)
}

func TestIgnoreConflictWithLog(t *testing.T) {
	// Random error
	result, err := IgnoreConflictWithLog("test-message", fmt.Errorf("test-error"), zap.S())
	assert.Nil(t, err)
	assert.NotEqual(t, result, reconcile.Result{})

	// Conflict Error
	result, err = IgnoreConflictWithLog("test-message", k8serrors.NewConflict(schema.GroupResource{}, "test-conflict", fmt.Errorf("test-error")), zap.S())
	assert.Nil(t, err)
	assert.NotEqual(t, result, reconcile.Result{})

	// No Error
	result, err = IgnoreConflictWithLog("test-message", nil, zap.S())
	assert.Nil(t, err)
	assert.Equal(t, result, reconcile.Result{})
}

func TestBuildZapperLog(t *testing.T) {
	logger, err := BuildZapInfoLogger(2)
	assert.Nil(t, err)
	assert.NotNil(t, logger)
}
