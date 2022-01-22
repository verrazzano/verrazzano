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
	count int
}

// TestLog tests the ProgressLogger function periodic logging
// GIVEN a ProgressLogger with a frequency of 3 seconds
// WHEN log is called 5 times in 5 seconds
// THEN ensure that 2 messages are logged
func TestLog(t *testing.T) {
	testOpts := kzap.Options{}
	testOpts.Development = true
	testOpts.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	InitLogs(testOpts)
	logger := fakeLogger{}
	l := NewProgressLogger(&logger, "a").SetFrequency(3)

	// 5 calls to log should result in only 2 log messages being written
	// since the frequency is 3 secs
	for i := 0; i < 5; i++ {
		l.Log("test1")
		time.Sleep(time.Duration(1) * time.Second)
	}
	assert.Equal(t, 2, logger.count,)
}

func (l *fakeLogger) Info(args ...interface{}) {
	fmt.Println(args...)
	l.count = l.count + 1
}
