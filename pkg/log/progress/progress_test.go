// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package progress

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log"
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
	log.InitLogs(testOpts)
	msg := "test1"
	logger := fakeLogger{expectedMsg: msg}
	setFakeLogger(&logger)
	const rKey = "testns/test"
	rl := EnsureRootLogger(rKey, zap.S())
	l := rl.EnsureProgressLogger("comp1").SetFrequency(3)

	// 5 calls to log should result in only 2 log messages being written
	// since the frequency is 3 secs
	for i := 0; i < 5; i++ {
		l.Progress(msg)
		time.Sleep(time.Duration(1) * time.Second)
	}
	assert.Equal(t, 2, logger.count)
	assert.Equal(t, logger.actualMsg, logger.expectedMsg)
	DeleteRootLogger(rKey)
}

// TestLogNewMsg tests the ProgressLogger function periodic logging
// GIVEN a ProgressLogger with a frequency of 2 seconds
// WHEN log is called 5 times with 2 different message
// THEN ensure that 2 messages are logged
func TestLogNewMsg(t *testing.T) {
	testOpts := kzap.Options{}
	testOpts.Development = true
	testOpts.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	log.InitLogs(testOpts)
	msg := "test1"
	msg2 := "test2"
	logger := fakeLogger{expectedMsg: msg}
	setFakeLogger(&logger)
	const rKey = "testns/test2"
	rl := EnsureRootLogger(rKey, zap.S())
	l := rl.EnsureProgressLogger("comp1").SetFrequency(2)

	// Calls to log should result in only 2 log messages being written
	l.Progress(msg)
	l.Progress(msg)
	l.Progress(msg)
	l.Progress(msg2)
	l.Progress(msg2)
	assert.Equal(t, 2, logger.count)
	assert.Equal(t, logger.actualMsg, msg2)
	DeleteRootLogger(rKey)
}

// TestLogFormat tests the ProgressLogger function message formatting
// GIVEN a ProgressLogger
// WHEN log.Infof is called with a string and a template
// THEN ensure that the message is formatted correctly and logged
func TestLogFormat(t *testing.T) {
	testOpts := kzap.Options{}
	testOpts.Development = true
	testOpts.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	log.InitLogs(testOpts)
	template := "test %s"
	inStr := "foo"
	logger := fakeLogger{}
	setFakeLogger(&logger)
	logger.expectedMsg = fmt.Sprintf(template, inStr)
	const rKey = "testns/test3"
	rl := EnsureRootLogger(rKey, zap.S())
	l := rl.EnsureProgressLogger("comp1")
	l.Progressf(template, inStr)
	assert.Equal(t, 1, logger.count)
	assert.Equal(t, logger.actualMsg, logger.expectedMsg)
	DeleteRootLogger(rKey)
}

// TestDefault tests the DefaultProgressLogger
// GIVEN a DefaultProgressLogger
// WHEN log.Infof is called with a string and a template
// THEN ensure that the message is formatted correctly and logged
func TestDefault(t *testing.T) {
	testOpts := kzap.Options{}
	testOpts.Development = true
	testOpts.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	log.InitLogs(testOpts)
	template := "test %s"
	inStr := "foo"
	logger := fakeLogger{}
	setFakeLogger(&logger)
	logger.expectedMsg = fmt.Sprintf(template, inStr)
	const rKey = "testns/test3"
	l := EnsureRootLogger(rKey, zap.S()).DefaultProgressLogger()
	l.Progressf(template, inStr)
	assert.Equal(t, 1, logger.count)
	assert.Equal(t, logger.actualMsg, logger.expectedMsg)
	DeleteRootLogger(rKey)
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

// Error is a wrapper for SugaredLogger Error
func (l *fakeLogger) Error(args ...interface{}) {
}

// Errorf is a wrapper for SugaredLogger Errorf
func (l *fakeLogger) Errorf(template string, args ...interface{}) {
}


// Error is a wrapper for SugaredLogger Error
func (l *fakeLogger) Progress(args ...interface{}) {
}

// Errorf is a wrapper for SugaredLogger Errorf
func (l *fakeLogger) Progressf(template string, args ...interface{}) {
}
