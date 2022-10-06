// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzlog

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	"go.uber.org/zap"
)

type fakeLogger struct {
	expectedMsg string
	actualMsg   string
	count       int
}

// var _ SugaredLogger = fakeLogger{}

// TestLog tests the ProgressLogger function periodic logging
// GIVEN a ProgressLogger with a frequency of 3 seconds
// WHEN log is called 5 times in 5 seconds to log the same message
// THEN ensure that 2 messages are logged
func TestLog(t *testing.T) {
	msg := "test1"
	logger := fakeLogger{expectedMsg: msg}
	const rKey = "testns/test"
	rl := EnsureContext(rKey)
	l := rl.EnsureLogger("comp1", &logger, zap.S()).SetFrequency(3)

	// 5 calls to log should result in only 2 log messages being written
	// since the frequency is 3 secs
	for i := 0; i < 5; i++ {
		l.Progress(msg)
		time.Sleep(time.Duration(1) * time.Second)
	}
	assert.Equal(t, 2, logger.count)
	assert.Equal(t, logger.expectedMsg, logger.actualMsg)
	DeleteLogContext(rKey)
}

// TestLog2 tests the ProgressLogger function periodic logging
// GIVEN a ProgressLogger with a frequency of 3 seconds
// WHEN log is called 5 times in 5 seconds to log the 2 messages
// THEN ensure that 2 messages are logged twice each
func TestLog2(t *testing.T) {
	msg := "test1"
	msg2 := "test2"
	logger := fakeLogger{expectedMsg: msg2}
	const rKey = "testns/test"
	rl := EnsureContext(rKey)
	l := rl.EnsureLogger("comp1", &logger, zap.S()).SetFrequency(3)

	// 5 calls to log should result in only 2 log messages being written
	// since the frequency is 3 secs
	for i := 0; i < 5; i++ {
		l.Progress(msg)
		l.Progress(msg2)
		time.Sleep(time.Duration(1) * time.Second)
	}
	assert.Equal(t, 4, logger.count)
	assert.Equal(t, logger.expectedMsg, logger.actualMsg)
	DeleteLogContext(rKey)
}

// TestLogRepeat tests the ProgressLogger function ignore repeated logs
// GIVEN a ProgressLogger with a frequency of 2 seconds
// WHEN log is called 5 times with 1 message and no sleep
// THEN ensure that 1 message is logged
func TestLogRepeat(t *testing.T) {
	msg := "test1"
	logger := fakeLogger{expectedMsg: msg}
	const rKey = "testns/test2"
	rl := EnsureContext(rKey)
	l := rl.EnsureLogger("comp1", &logger, zap.S()).SetFrequency(2)

	// Calls to log should result in only 2 log messages being written
	l.Progress(msg)
	l.Progress(msg)
	l.Progress(msg)
	l.Progress(msg)
	l.Progress(msg)
	assert.Equal(t, 1, logger.count)
	assert.Equal(t, msg, logger.actualMsg)
	DeleteLogContext(rKey)
}

// TestErrorfThrottled tests the ProgressLogger function throttle repeated error messages
// GIVEN a ProgressLogger with a frequency of 30 seconds
// WHEN ErrorfThrottled is called 5 times with 1 message and no sleep
// THEN ensure that 1 message is logged
func TestErrorfThrottled(t *testing.T) {
	msg := "test1"
	logger := fakeLogger{expectedMsg: msg}
	const rKey = "testns/errors"
	rl := EnsureContext(rKey)
	l := rl.EnsureLogger("comp1", &logger, zap.S()).SetFrequency(30)

	// Calls to log should result in only 1 log messages being written
	l.ErrorfThrottled(msg)
	l.ErrorfThrottled(msg)
	l.ErrorfThrottled(msg)
	l.ErrorfThrottled(msg)
	l.ErrorfThrottled(msg)
	assert.Equal(t, 1, logger.count)
	assert.Equal(t, msg, logger.actualMsg)
	DeleteLogContext(rKey)
}

// TestErrorfThrottledNewErr tests the ProgressLogger function throttle repeated error messages
// GIVEN a ProgressLogger with a frequency of 30 seconds
// WHEN ErrorfThrottledNewErr is called 5 times with 1 message and no sleep
// THEN ensure that 1 message is logged
func TestErrorfThrottledNewErr(t *testing.T) {
	const messageTemplate = "Mymessage %s"
	const messageParameter = "test2"
	msg := fmt.Sprintf(messageTemplate, messageParameter)
	logger := fakeLogger{expectedMsg: msg}
	const rKey = "testns/errorsNew"
	rl := EnsureContext(rKey)
	l := rl.EnsureLogger("comp1", &logger, zap.S()).SetFrequency(30)

	// Calls to log should result in only 1 log messages being written
	assert.Error(t, l.ErrorfThrottledNewErr(messageTemplate, messageParameter))
	assert.Error(t, l.ErrorfThrottledNewErr(messageTemplate, messageParameter))
	assert.Error(t, l.ErrorfThrottledNewErr(messageTemplate, messageParameter))
	assert.Error(t, l.ErrorfThrottledNewErr(messageTemplate, messageParameter))
	assert.Error(t, l.ErrorfThrottledNewErr(messageTemplate, messageParameter))
	assert.Equal(t, 1, logger.count)
	assert.Equal(t, msg, logger.actualMsg)
	DeleteLogContext(rKey)
}

// TestHistory tests the ProgressLogger function ignore previous progrsss messages
// GIVEN a ProgressLogger with a frequency of 2 seconds
// WHEN log is called 5 times with 2 message using repeats, and no sleep
// THEN ensure that 2 messages is logged
func TestHistory(t *testing.T) {
	msg := "test1"
	msg2 := "test2"
	logger := fakeLogger{expectedMsg: msg}
	const rKey = "testns/test2"
	rl := EnsureContext(rKey)
	l := rl.EnsureLogger("comp1", &logger, zap.S()).SetFrequency(2)

	// Calls to log should result in only 2 log messages being written
	l.Progress(msg)
	l.Progress(msg)
	l.Progress(msg)
	l.Progress(msg2)
	l.Progress(msg2)
	l.Progress(msg)
	l.Progress(msg2)
	l.Progress(msg)
	assert.Equal(t, 2, logger.count)
	assert.Equal(t, msg2, logger.actualMsg)
	DeleteLogContext(rKey)
}

// TestHistoryOnce tests the ProgressLogger function ignore previous once messages
// GIVEN a ProgressLogger with a frequency of 2 seconds
// WHEN log is called 5 times with 2 message using repeats, and no sleep
// THEN ensure that 2 messages is logged
func TestHistoryOnce(t *testing.T) {
	msg := "test1"
	msg2 := "test2"
	logger := fakeLogger{expectedMsg: msg}
	const rKey = "testns/test2"
	rl := EnsureContext(rKey)
	l := rl.EnsureLogger("comp1", &logger, zap.S())

	// Calls to log should result in only 2 log messages being written
	l.Once(msg)
	l.Once(msg)
	l.Once(msg2)
	l.Once(msg)
	l.Once(msg2)
	l.Once(msg)
	assert.Equal(t, 2, logger.count)
	assert.Equal(t, msg2, logger.actualMsg)
	DeleteLogContext(rKey)
}

// TestLogNewMsg tests the ProgressLogger function periodic logging
// GIVEN a ProgressLogger with a frequency of 2 seconds
// WHEN log is called 5 times with 2 different message
// THEN ensure that 2 messages are logged
func TestLogNewMsg(t *testing.T) {
	msg := "test1"
	msg2 := "test2"
	logger := fakeLogger{expectedMsg: msg}
	const rKey = "testns/test2"
	rl := EnsureContext(rKey)
	l := rl.EnsureLogger("comp1", &logger, zap.S()).SetFrequency(2)

	// Calls to log should result in only 2 log messages being written
	l.Progress(msg)
	l.Progress(msg)
	l.Progress(msg)
	l.Progress(msg2)
	l.Progress(msg2)
	assert.Equal(t, 2, logger.count)
	assert.Equal(t, msg2, logger.actualMsg)
	DeleteLogContext(rKey)
}

// TestLogFormat tests the ProgressLogger function message formatting
// GIVEN a ProgressLogger
// WHEN log.Infof is called with a string and a template
// THEN ensure that the message is formatted correctly and logged
func TestLogFormat(t *testing.T) {
	template := "test %s"
	inStr := "foo"
	logger := fakeLogger{}
	logger.expectedMsg = fmt.Sprintf(template, inStr)
	const rKey = "testns/test3"
	rl := EnsureContext(rKey)
	l := rl.EnsureLogger("comp1", &logger, zap.S())
	l.Progressf(template, inStr)
	assert.Equal(t, 1, logger.count)
	assert.Equal(t, logger.expectedMsg, logger.actualMsg)
	DeleteLogContext(rKey)
}

// TestMultipleContexts tests the EnsureContext and DeleteLogContext
// WHEN EnsureContext is called multiple times
// THEN ensure that the context map has an entry for each context and that
//
//	the context map is empty when they the contexts are deleted
func TestMultipleContexts(t *testing.T) {
	const rKey1 = "k1"
	const rKey2 = "k2"
	c1 := EnsureContext(rKey1)
	c2 := EnsureContext(rKey2)

	assert.Equal(t, 2, len(LogContextMap))
	c1Actual := LogContextMap[rKey1]
	assert.Equal(t, c1, c1Actual)
	c2Actual := LogContextMap[rKey2]
	assert.Equal(t, c2, c2Actual)
	DeleteLogContext(rKey1)
	DeleteLogContext(rKey2)
	assert.Equal(t, 0, len(LogContextMap))
}

// TestZap tests the zap SugaredLogger
// GIVEN a zap SugaredLogger
// WHEN EnsureContext is called with the SugaredLogger
// THEN ensure that the ProgressMessage can be called
func TestZap(t *testing.T) {
	testOpts := kzap.Options{}
	testOpts.Development = true
	testOpts.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	log.InitLogs(testOpts)
	const rKey = "testns/test3"
	l := EnsureContext(rKey).EnsureLogger("test", zap.S(), zap.S())
	l.Progress("testmsg")
	DeleteLogContext(rKey)
}

// SetZapLogger gets the zap logger
func (l *fakeLogger) SetZapLogger(zap *zap.SugaredLogger) {
}

// GetZapLogger gets the zap logger
func (l *fakeLogger) GetZapLogger() *zap.SugaredLogger {
	return nil
}

func (l *fakeLogger) Info(args ...interface{}) {
	s := fmt.Sprint(args...)
	l.actualMsg = s
	l.count = l.count + 1
	fmt.Println(s)
}

// Infof formats a message and logs it
func (l *fakeLogger) Infof(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	l.Info(s)
}

// Infow formats a message and logs it
func (l *fakeLogger) Infow(msg string, keysAndValues ...interface{}) {
	s := fmt.Sprintf("%s %v", msg, keysAndValues)
	l.Info(s)
}

// Debug is a wrapper for SugaredLogger Debug
func (l *fakeLogger) Debug(args ...interface{}) {
}

// Debugf is a wrapper for SugaredLogger Debugf
func (l *fakeLogger) Debugf(template string, args ...interface{}) {
}

// Debugw formats a message and logs it
func (l *fakeLogger) Debugw(msg string, keysAndValues ...interface{}) {
}

// Error is a wrapper for SugaredLogger Error
func (l *fakeLogger) Error(args ...interface{}) {
	l.Info(args...)
}

// Errorf is a wrapper for SugaredLogger Errorf
func (l *fakeLogger) Errorf(template string, args ...interface{}) {
	l.Infof(template, args...)
}

// Errorw formats a message and logs it
func (l *fakeLogger) Errorw(msg string, keysAndValues ...interface{}) {
}
