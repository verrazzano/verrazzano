// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package log

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
	"time"

	"go.uber.org/zap"
)

type fakeLogger struct {
	expectedMsg string
	actualMsg   string
	count       int
}

// TestLog tests the ProgressLogger function periodic logging
// GIVEN a ProgressLogger with a frequency of 3 seconds
// WHEN log is called 5 times in 5 seconds to log the same message
// THEN ensure that 2 messages are logged
func TestLog(t *testing.T) {
	testOpts := kzap.Options{}
	testOpts.Development = true
	testOpts.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	InitLogs(testOpts)
	msg := "test1"
	logger := fakeLogger{expectedMsg: msg}
	l := NewProgressLogger(&logger, "a").SetFrequency(3)

	// 5 calls to log should result in only 2 log messages being written
	// since the frequency is 3 secs
	for i := 0; i < 5; i++ {
		l.Info(msg)
		time.Sleep(time.Duration(1) * time.Second)
	}
	assert.Equal(t, 2, logger.count)
	assert.Equal(t, logger.actualMsg, logger.expectedMsg)
}

// TestLogNewMsg tests the ProgressLogger function periodic logging
// GIVEN a ProgressLogger with a frequency of 2 seconds
// WHEN log is called 5 times with 2 different message
// THEN ensure that 2 messages are logged
func TestLogNewMsg(t *testing.T) {
	testOpts := kzap.Options{}
	testOpts.Development = true
	testOpts.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	InitLogs(testOpts)
	msg := "test1"
	msg2 := "test2"
	logger := fakeLogger{expectedMsg: msg}
	l := NewProgressLogger(&logger, "a").SetFrequency(2)

	// Calls to log should result in only 2 log messages being written
	l.Info(msg)
	l.Info(msg)
	l.Info(msg)
	l.Info(msg2)
	l.Info(msg2)
	assert.Equal(t, 2, logger.count)
	assert.Equal(t, logger.actualMsg, msg2)
}

// TestLogFormat tests the ProgressLogger function message formatting
// GIVEN a ProgressLogger
// WHEN log.Infof is called with a string and a template
// THEN ensure that the message is formatted correctly and logged
func TestLogFormat(t *testing.T) {
	testOpts := kzap.Options{}
	testOpts.Development = true
	testOpts.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	InitLogs(testOpts)
	template := "test %s"
	inStr := "foo"
	logger := fakeLogger{}
	logger.expectedMsg = fmt.Sprintf(template, inStr)
	l := NewProgressLogger(&logger, "a")
	l.Infof(template, inStr)
	assert.Equal(t, 1, logger.count)
	assert.Equal(t, logger.actualMsg, logger.expectedMsg)
}

func (l *fakeLogger) Info(args ...interface{}) {
	s := fmt.Sprint(args...)
	l.actualMsg = s
	l.count = l.count + 1
}
